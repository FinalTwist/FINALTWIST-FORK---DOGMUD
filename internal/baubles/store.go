package baubles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

// The catalog on disk: records in shards of ShardSize by id, plus a meta
// file holding the next id. Only the shard a change touched is rewritten,
// through util.Save (temp file and rename). A shard that cannot be read is
// renamed aside, logged, and skipped; the next id is pushed past it so its
// ids are never handed out again while items may still carry them.

const (
	// ShardSize is how many records one shard file holds.
	ShardSize = 500

	schemaVersion = 1
	metaFileName  = `meta.yaml`
	shardPrefix   = `catalog-`
)

type shardFile struct {
	Schema  int       `yaml:"schema"`
	Records []*Record `yaml:"records"`
}

type metaFile struct {
	Schema  int    `yaml:"schema"`
	NextSeq uint64 `yaml:"next_seq"`
}

// idFor is the record id for a sequence number: B0000001, B0000002, ...
func idFor(seq uint64) string {
	return fmt.Sprintf(`B%07d`, seq)
}

// seqOf is the sequence number in an id, or false for anything else.
func seqOf(id string) (uint64, bool) {
	if len(id) < 2 || id[0] != 'B' {
		return 0, false
	}
	n, err := strconv.ParseUint(id[1:], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// shardOf is the shard a record lives in: ids start at 1, so shard 0
// holds B0000001 to B0000500, shard 1 B0000501 to B0001000, and so on.
func shardOf(seq uint64) int {
	if seq == 0 {
		return 0
	}
	return int((seq - 1) / ShardSize)
}

func shardPath(dir string, shard int) string {
	return filepath.Join(dir, fmt.Sprintf(`%s%04d.yaml`, shardPrefix, shard))
}

// loadResult is everything read from disk at boot.
type loadResult struct {
	records     map[string]*Record
	nextSeq     uint64
	quarantined []string
	// rewrite are shards to write back at once: ones holding a record that
	// belongs in another (a catalog written before shards began at id 1),
	// and the shards those records belong in, so every record ends up in
	// its own shard and nowhere else.
	rewrite []int
}

// loadDir reads every shard and the meta file. A missing directory is an
// empty catalog, not an error.
func loadDir(dir string) (loadResult, error) {
	res := loadResult{records: map[string]*Record{}, nextSeq: 1}

	if data, err := os.ReadFile(filepath.Join(dir, metaFileName)); err == nil {
		var m metaFile
		if err := yaml.Unmarshal(data, &m); err == nil && m.NextSeq > res.nextSeq {
			res.nextSeq = m.NextSeq
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, err
	}

	names := []string{}
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasPrefix(n, shardPrefix) && strings.HasSuffix(n, `.yaml`) {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	properSeen := map[string]bool{}
	for _, n := range names {
		path := filepath.Join(dir, n)
		shardNum, numErr := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(n, shardPrefix), `.yaml`))

		data, err := os.ReadFile(path)
		var sf shardFile
		if err == nil {
			err = yaml.Unmarshal(data, &sf)
		}
		if err != nil {
			aside := fmt.Sprintf(`%s.corrupt-%d`, path, time.Now().Unix())
			_ = os.Rename(path, aside)
			res.quarantined = append(res.quarantined, n)
			if numErr == nil {
				if past := uint64(shardNum+1)*ShardSize + 1; past > res.nextSeq {
					res.nextSeq = past
				}
			}
			continue
		}

		for _, r := range sf.Records {
			if r == nil {
				continue
			}
			seq, ok := seqOf(r.Id)
			if !ok {
				continue
			}
			proper := numErr == nil && shardNum == shardOf(seq)
			if !proper {
				// A copy outside its own shard: the one in its own shard,
				// which is the one kept up to date, wins whichever file is
				// read last. Both files are rewritten.
				res.rewrite = append(res.rewrite, shardOf(seq))
				if numErr == nil {
					res.rewrite = append(res.rewrite, shardNum)
				}
				if properSeen[r.Id] {
					continue
				}
			} else {
				properSeen[r.Id] = true
			}
			res.records[r.Id] = r
			if seq+1 > res.nextSeq {
				res.nextSeq = seq + 1
			}
		}
	}
	return res, nil
}

// writeShard writes one shard's records, sorted by id.
func writeShard(dir string, shard int, recs []*Record) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	sort.Slice(recs, func(a, b int) bool { return recs[a].Id < recs[b].Id })
	data, err := yaml.Marshal(shardFile{Schema: schemaVersion, Records: recs})
	if err != nil {
		return err
	}
	return util.Save(shardPath(dir, shard), data)
}

// writeMeta writes the next id.
func writeMeta(dir string, nextSeq uint64) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(metaFile{Schema: schemaVersion, NextSeq: nextSeq})
	if err != nil {
		return err
	}
	return util.Save(filepath.Join(dir, metaFileName), data)
}
