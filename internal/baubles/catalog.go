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

type catalog struct {
	mu      sync.RWMutex
	dir     string
	records map[string]*Record
	nextSeq uint64
	dirty   map[int]bool // shards whose last write failed
	metaBad bool         // the meta file's last write failed
}

var cat = &catalog{records: map[string]*Record{}, nextSeq: 1, dirty: map[int]bool{}}

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
	cat.mu.Unlock()

	for _, n := range res.quarantined {
		mudlog.Error(`baubles.Load`, `action`, `quarantined unreadable shard`, `file`, n)
	}
	if len(res.quarantined) > 0 {
		// The next id was pushed past the lost shard's ids; the shard file
		// is renamed aside, so only the meta file remembers that now. Write
		// it at once, or a reboot before any new find would forget it and
		// hand those ids out again while old items still carry them.
		cat.mu.Lock()
		cat.persistMetaLocked()
		cat.mu.Unlock()
	}
	if len(res.rewrite) > 0 {
		cat.mu.Lock()
		done := map[int]bool{}
		for _, shard := range res.rewrite {
			if !done[shard] {
				done[shard] = true
				_ = cat.persistShardLocked(shard)
			}
		}
		cat.mu.Unlock()
		mudlog.Info(`baubles.Load`, `action`, `moved records into their own shards`, `shards`, len(done))
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
	defer cat.mu.Unlock()
	if cat.dir == `` {
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

	cat.persistMetaLocked()
	shard := shardOf(seq)
	wasDirty := cat.dirty[shard]
	if err := cat.persistShardLocked(shard); err != nil {
		// Taken back: what is on disk for this shard is what memory now
		// holds again, unless an earlier write had already failed.
		delete(cat.records, r.Id)
		if !wasDirty {
			delete(cat.dirty, shard)
		}
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
	defer cat.mu.Unlock()
	r, ok := cat.records[id]
	if !ok {
		return Record{}, false
	}
	change(r)
	r.Id = id
	_ = cat.persistShardLocked(shardOf(seq))
	return *r, true
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
	cat.mu.Lock()
	defer cat.mu.Unlock()
	if cat.dir == `` {
		return
	}
	if cat.metaBad {
		cat.persistMetaLocked()
	}
	for shard := range cat.dirty {
		_ = cat.persistShardLocked(shard)
	}
}

// persistShardLocked writes every record of one shard. Caller holds mu.
func (c *catalog) persistShardLocked(shard int) error {
	recs := []*Record{}
	for id, r := range c.records {
		if seq, ok := seqOf(id); ok && shardOf(seq) == shard {
			cp := *r
			recs = append(recs, &cp)
		}
	}
	if err := writeShard(c.dir, shard, recs); err != nil {
		c.dirty[shard] = true
		mudlog.Error(`baubles`, `action`, `write shard`, `shard`, shard, `error`, err)
		return err
	}
	delete(c.dirty, shard)
	return nil
}

// persistMetaLocked writes the next id. Caller holds mu.
func (c *catalog) persistMetaLocked() {
	if err := writeMeta(c.dir, c.nextSeq); err != nil {
		c.metaBad = true
		mudlog.Error(`baubles`, `action`, `write meta`, `error`, err)
		return
	}
	c.metaBad = false
}
