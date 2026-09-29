package baubles

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// The catalog is the bauble "database": every record, in memory, written
// through to disk on every change. Changes are rare (a find, a naming, a
// sale), so write-through costs little and means a record is on disk before
// any save file can hold an item that points at it.
//
// It has its own lock because GetSpec (and so the resolver) is read from
// more places than the game loop. Callers never hold it across a call out.
// Disk writes happen OUTSIDE it (persistShard, persistMeta): a write takes
// writeMu, which orders writes, snapshots what it writes under a brief read
// lock, then marshals and writes with mu free, so a slow disk never holds
// up GetSpec. Each write takes its snapshot after the change that called
// it, so the last write of a shard always carries the latest records.

type catalog struct {
	mu      sync.RWMutex
	writeMu sync.Mutex // orders disk writes; never taken while holding mu
	dir     string
	records map[string]*Record
	nextSeq uint64
	dirty   map[int]bool // shards whose last write failed
	metaBad bool         // the meta file's last write failed

	// credits indexes the records whose return earned their thief
	// reputation (ReturnCreditAt set), by that thief, so ReturnCredits
	// reads a thief's few credits instead of the whole catalog.
	credits map[int]map[string]bool // userId -> record ids
}

var cat = &catalog{records: map[string]*Record{}, nextSeq: 1, dirty: map[int]bool{}, credits: map[int]map[string]bool{}}

// indexCreditLocked brings the credit index up to date for r. Caller holds
// mu for writing.
func (c *catalog) indexCreditLocked(r *Record) {
	for uid, ids := range c.credits {
		if ids[r.Id] && (r.ReturnCreditAt.IsZero() || r.ReturnCreditUserId != uid) {
			delete(ids, r.Id)
		}
	}
	if r.ReturnCreditAt.IsZero() || r.ReturnCreditUserId == 0 {
		return
	}
	if c.credits[r.ReturnCreditUserId] == nil {
		c.credits[r.ReturnCreditUserId] = map[string]bool{}
	}
	c.credits[r.ReturnCreditUserId][r.Id] = true
}

// rebuildCreditsLocked rebuilds the credit index from every record. Caller
// holds mu for writing.
func (c *catalog) rebuildCreditsLocked() {
	c.credits = map[int]map[string]bool{}
	for _, r := range c.records {
		c.indexCreditLocked(r)
	}
}

// catalogDir is where the catalog lives: <DataFiles>/baubles.
func catalogDir() string {
	return util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `baubles`)
}

// Load reads the catalog from disk and installs the item resolver. Call
// once at boot, after items.LoadDataFiles(). A corrupt shard is logged and
// skipped, never fatal.
func Load() error {
	return loadFrom(catalogDir())
}

func loadFrom(dir string) error {
	res, err := loadDir(dir)

	cat.mu.Lock()
	cat.dir = dir
	cat.records = res.records
	cat.nextSeq = res.nextSeq
	cat.dirty = map[int]bool{}
	cat.metaBad = false
	cat.rebuildCreditsLocked()
	cat.mu.Unlock()

	for _, n := range res.quarantined {
		mudlog.Error(`baubles.Load`, `action`, `quarantined unreadable shard`, `file`, n)
	}
	if len(res.quarantined) > 0 {
		// The next id was pushed past the lost shard's ids; the shard file
		// is renamed aside, so only the meta file remembers that now. Write
		// it at once, or a reboot before any new find would forget it and
		// hand those ids out again while old items still carry them.
		cat.persistMeta()
	}
	if len(res.rewrite) > 0 {
		done := map[int]bool{}
		for _, shard := range res.rewrite {
			if !done[shard] {
				done[shard] = true
				_ = cat.persistShard(shard)
			}
		}
		mudlog.Info(`baubles.Load`, `action`, `moved records into their own shards`, `shards`, len(done))
	}
	if n := Prune(time.Now()); n > 0 {
		mudlog.Info(`baubles.Load`, `action`, `pruned records`, `count`, n)
	}
	items.SetBaubleResolver(resolve)
	mudlog.Info(`baubles.Load()`, `records`, len(res.records), `nextId`, idFor(res.nextSeq))
	return err
}

// SetDirForTest points the catalog at an empty directory (use t.TempDir())
// and installs the resolver. It discards whatever was loaded before.
func SetDirForTest(dir string) {
	cat.mu.Lock()
	cat.dir = dir
	cat.records = map[string]*Record{}
	cat.nextSeq = 1
	cat.dirty = map[int]bool{}
	cat.metaBad = false
	cat.credits = map[int]map[string]bool{}
	cat.mu.Unlock()
	items.SetBaubleResolver(resolve)
}

// resolve is the items.BaubleResolver.
func resolve(id string) (items.BaubleView, bool) {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	r, ok := cat.records[id]
	if !ok {
		return items.BaubleView{}, false
	}
	return r.View(), true
}

// ErrNoCatalog is returned when the catalog has nowhere to write, which
// means Load (or SetDirForTest) has not run.
var ErrNoCatalog = errors.New(`bauble catalog not loaded`)

// Create stores a new record under the next id and writes it to disk
// before returning. Id is assigned here; FoundAt defaults to now. The
// returned copy is the stored record. On a write error NOTHING is created:
// the record is taken back out of memory and the error returned, so no
// item can ever be handed out pointing at a record that a crash would lose
// (Mint then refuses, and the find crumbles away). Its id is not reused.
func Create(r Record) (Record, error) {
	cat.mu.Lock()
	if cat.dir == `` {
		cat.mu.Unlock()
		return Record{}, ErrNoCatalog
	}

	seq := cat.nextSeq
	cat.nextSeq++
	r.Id = idFor(seq)
	if r.FoundAt.IsZero() {
		r.FoundAt = time.Now().UTC()
	}
	stored := r
	cat.records[r.Id] = &stored
	cat.indexCreditLocked(&stored)
	cat.mu.Unlock()

	cat.persistMeta()
	shard := shardOf(seq)
	if err := cat.persistShard(shard); err != nil {
		// Taken back, and the shard written again without it; should that
		// write fail too, the shard stays dirty for SaveAll to retry. No
		// item points at the record yet (Mint refuses on this error).
		cat.mu.Lock()
		delete(cat.records, r.Id)
		cat.indexCreditLocked(&Record{Id: r.Id})
		cat.mu.Unlock()
		_ = cat.persistShard(shard)
		return Record{}, err
	}
	return stored, nil
}

// Get returns a copy of the record.
func Get(id string) (Record, bool) {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	r, ok := cat.records[id]
	if !ok {
		return Record{}, false
	}
	return *r, true
}

// Update changes a record in place and writes its shard. The id cannot be
// changed. Returns the updated copy, or false if there is no such record.
func Update(id string, change func(r *Record)) (Record, bool) {
	seq, ok := seqOf(id)
	if !ok {
		return Record{}, false
	}
	cat.mu.Lock()
	r, ok := cat.records[id]
	if !ok {
		cat.mu.Unlock()
		return Record{}, false
	}
	change(r)
	r.Id = id
	cat.indexCreditLocked(r)
	out := *r
	cat.mu.Unlock()
	_ = cat.persistShard(shardOf(seq))
	return out, true
}

// Count is how many records the catalog holds.
func Count() int {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	return len(cat.records)
}

// Recent returns copies of up to n records, newest first.
func Recent(n int) []Record {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	out := make([]Record, 0, len(cat.records))
	for _, r := range cat.records {
		out = append(out, *r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Id > out[b].Id })
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// SaveAll retries any shard (and the meta file) whose last write failed.
// Everything else is already on disk. Call at shutdown and copyover.
func SaveAll() {
	cat.mu.RLock()
	dir, metaBad := cat.dir, cat.metaBad
	dirty := make([]int, 0, len(cat.dirty))
	for shard := range cat.dirty {
		dirty = append(dirty, shard)
	}
	cat.mu.RUnlock()
	if dir == `` {
		return
	}
	if metaBad {
		cat.persistMeta()
	}
	for _, shard := range dirty {
		_ = cat.persistShard(shard)
	}
	if n := Prune(time.Now()); n > 0 {
		mudlog.Info(`baubles`, `action`, `pruned records`, `count`, n)
	}
}

// shardWriter is writeShard. A variable so a test can hold a write open
// and show that reads do not wait for it.
var shardWriter = writeShard

// persistShard writes every record of one shard, with mu free while it
// marshals and writes (see the catalog comment). Never call it holding mu.
func (c *catalog) persistShard(shard int) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	dir := c.dir
	recs := []*Record{}
	for id, r := range c.records {
		if seq, ok := seqOf(id); ok && shardOf(seq) == shard {
			cp := *r
			recs = append(recs, &cp)
		}
	}
	c.mu.RUnlock()

	err := shardWriter(dir, shard, recs)

	c.mu.Lock()
	if err != nil {
		c.dirty[shard] = true
	} else {
		delete(c.dirty, shard)
	}
	c.mu.Unlock()
	if err != nil {
		mudlog.Error(`baubles`, `action`, `write shard`, `shard`, shard, `error`, err)
	}
	return err
}

// persistMeta writes the next id, with mu free while it writes. Never call
// it holding mu.
func (c *catalog) persistMeta() {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	dir, next := c.dir, c.nextSeq
	c.mu.RUnlock()

	err := writeMeta(dir, next)

	c.mu.Lock()
	c.metaBad = err != nil
	c.mu.Unlock()
	if err != nil {
		mudlog.Error(`baubles`, `action`, `write meta`, `error`, err)
	}
}

// KeepDuration is how long a record whose bauble is gone from the world
// (sold, or vanished untaken) is kept before Prune removes it
// (Balance.BaubleCatalogKeepDays). Sales stats read the last seven days,
// and a crash can put a just-sold bauble back in a player's pack, so it is
// never shorter than a week.
func KeepDuration() time.Duration {
	return time.Duration(configs.GetBalanceConfig().BaubleCatalogKeepDays) * 24 * time.Hour
}

// goneAt is when the record's bauble left the world for good: the later of
// its sale and its vanishing, zero while it may still be somewhere.
func (r Record) goneAt() time.Time {
	gone := r.SoldAt
	if r.VanishedAt.After(gone) {
		gone = r.VanishedAt
	}
	return gone
}

// prunable reports whether Prune may remove r now. Only a record whose
// bauble is gone from the world goes, once KeepDuration has passed: sold,
// or vanished untaken; a retired record goes with them once it too is sold
// or has vanished. A retired record whose bauble may still be in someone's
// pack stays, since the item keeps its value and would otherwise become
// an unknown, unsellable carrier. A record whose return earned its thief
// reputation (ReturnCreditAt) always stays: the credit history lives only
// there, and dropping it would let the bauble earn credit again.
func (r Record) prunable(now time.Time, keep time.Duration) bool {
	gone := r.goneAt()
	if gone.IsZero() || !r.ReturnCreditAt.IsZero() {
		return false
	}
	return now.Sub(gone) >= keep
}

// Prune removes the records prunable now and rewrites their shards, with
// the catalog lock held only to take them out (the writes happen outside
// it). It returns how many it removed. Only catalog shard files are
// rewritten: nothing else in the baubles directory (the fallback corpus
// overlay, for one) is touched. Load and SaveAll call it.
func Prune(now time.Time) int {
	keep := KeepDuration()
	cat.mu.Lock()
	if cat.dir == `` {
		cat.mu.Unlock()
		return 0
	}
	shards := map[int]bool{}
	removed := 0
	for id, r := range cat.records {
		if !r.prunable(now, keep) {
			continue
		}
		seq, ok := seqOf(id)
		if !ok {
			continue
		}
		delete(cat.records, id)
		shards[shardOf(seq)] = true
		removed++
	}
	cat.mu.Unlock()

	for shard := range shards {
		_ = cat.persistShard(shard)
	}
	return removed
}
