package baubles

import (
	"time"
)

// The catalog sweep. A record is only useful while some item points at it,
// and items leave the world in many ways the catalog never hears about
// (junked, eaten by a script, left on a corpse that decayed, on a character
// that was deleted, in a room file that was wiped). So rather than hook
// every one of them, a sweep periodically collects every bauble id any item
// still points at, in the live world and in every save file, and prunes the
// records nothing has pointed at for KeepDuration.
//
// Two phases. Collect (runSweep): the live world under the mud lock,
// through the sources registered with RegisterLiveSource (users, rooms,
// mobs, shops, guilds, the auction house), then every save file under
// DataFiles off the lock (sweep_disk.go). Apply (applySweep): under the
// catalog lock, a record something points at gets LastSeenAt = now and
// UnseenSweeps = 0, any other counts one more unseen sweep; then each
// changed shard is written without the records now prunable (prunableAt),
// before they leave memory (persistShardPruning).
//
// Fail closed: a collection that went wrong anywhere (a live source that
// panicked, a file that could not be read, or that names a bauble and does
// not parse) applies nothing at all, so a store it could not see never
// looks empty.
//
// Why a record must stay unseen for minUnseenSweeps sweeps AND
// KeepDuration: an item moving between stores while a sweep looks (from a
// room file into a live room, say) can be missed by that one sweep. Nothing
// is lost to a miss unless it repeats for the whole keep window. A store
// the sweep never looks at would repeat forever; the repo-root guards
// (TestItemWalkersVisitEveryItemField,
// TestEveryItemHolderIsASweepRootOrTransient) are what stop that.
//
// A sold record that something points at again (a crash rolled the
// seller's save back past the sale) is simply seen, so it stays. Its status
// is left as sold: every record is sellable already (sales.go), a save file
// on disk can lag a sale by one autosave, and rewriting the sale on that
// evidence would erase real ones.

// applySweep folds one complete collection (refs: every record id some item
// points at) into the catalog at now, and returns how many records were
// referenced and how many it pruned. Records found after now are left
// alone.
func applySweep(now time.Time, refs map[string]bool, keep time.Duration) (referenced int, pruned int) {
	shards := map[int]bool{}

	cat.mu.Lock()
	if cat.dir == `` {
		cat.mu.Unlock()
		return 0, 0
	}
	for id, r := range cat.records {
		seq, ok := seqOf(id)
		if !ok || r.FoundAt.After(now) {
			continue
		}
		shard := shardOf(seq)
		if refs[id] {
			referenced++
			r.LastSeenAt, r.UnseenSweeps = now, 0
			shards[shard] = true
			continue
		}
		if r.LastSeenAt.IsZero() {
			// Deploy grace: a record from before the sweep existed was never
			// seen. Its keep window starts at the first sweep that counts it,
			// or everything found over KeepDuration ago would go at the second
			// sweep, hours after the deploy.
			r.LastSeenAt = now
			shards[shard] = true
		}
		if r.UnseenSweeps < minUnseenSweeps {
			r.UnseenSweeps++
			shards[shard] = true
		}
		if r.prunableAt(now, keep) {
			shards[shard] = true
		}
	}
	cat.mu.Unlock()

	prune := func(r *Record) bool { return r.prunableAt(now, keep) }
	for shard := range shards {
		n, _ := cat.persistShardPruning(shard, prune)
		pruned += n
	}
	return referenced, pruned
}
