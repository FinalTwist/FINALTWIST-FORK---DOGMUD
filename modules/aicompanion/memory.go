package aicompanion

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Retrieval (F3.6, F3.7). The model never receives every memory. Each
// decision gets the few that score highest on a mix of how important the
// memory is, how recent it is, and how relevant it is to what is happening:
// the same words, the same place, the same people.

// recallContext is what the current moment is about.
type recallContext struct {
	NowUnix  int64
	PlaceId  int
	People   map[string]bool // lower-cased names present or speaking
	Keywords map[string]bool // lower-cased content words from stimuli and place
}

const (
	weightImportance = 0.45
	weightRecency    = 0.25
	weightRelevance  = 0.60
	recencyHalfLife  = 7 * 86400.0 // seconds
)

var stopwords = map[string]bool{
	`about`: true, `after`: true, `again`: true, `also`: true, `been`: true,
	`before`: true, `being`: true, `could`: true, `does`: true, `down`: true,
	`from`: true, `have`: true, `here`: true, `into`: true, `just`: true,
	`like`: true, `more`: true, `much`: true, `only`: true, `over`: true,
	`said`: true, `some`: true, `than`: true, `that`: true, `their`: true,
	`them`: true, `then`: true, `there`: true, `they`: true, `this`: true,
	`very`: true, `want`: true, `were`: true, `what`: true, `when`: true,
	`where`: true, `which`: true, `while`: true, `will`: true, `with`: true,
	`would`: true, `your`: true, `yours`: true, `you're`: true, `don't`: true,
	`know`: true, `think`: true, `really`: true, `well`: true, `going`: true,
}

// keywordsOf extracts lower-cased content words of four letters or more.
func keywordsOf(text string) map[string]bool {
	out := map[string]bool{}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	})
	for _, w := range words {
		w = strings.Trim(w, `'`)
		w = strings.TrimSuffix(w, `'s`)
		if len(w) < 4 || stopwords[w] {
			continue
		}
		out[w] = true
	}
	return out
}

func recency(nowUnix int64, unix int64) float64 {
	age := float64(nowUnix - unix)
	if age < 0 {
		age = 0
	}
	return math.Pow(0.5, age/recencyHalfLife)
}

// scoreMemory rates one memory against the moment. Range roughly 0..1.3.
func scoreMemory(mem Memory, ctx recallContext) float64 {
	imp := float64(mem.Importance) / 10.0
	rec := recency(ctx.NowUnix, mem.Unix)

	rel := 0.0
	if len(ctx.Keywords) > 0 {
		hits := 0
		for w := range keywordsOf(mem.Text) {
			if ctx.Keywords[w] {
				hits++
			}
		}
		rel += math.Min(float64(hits)/3.0, 1.0) * 0.6
	}
	if ctx.PlaceId > 0 && mem.PlaceId == ctx.PlaceId {
		rel += 0.3
	}
	for _, p := range mem.People {
		if ctx.People[strings.ToLower(p)] {
			rel += 0.1
			break
		}
	}
	if rel > 1 {
		rel = 1
	}
	return weightImportance*imp + weightRecency*rec + weightRelevance*rel
}

// selectMemories returns up to k memories for the prompt, oldest first.
// Reflections are excluded here; they are always shown separately.
func selectMemories(mems []Memory, ctx recallContext, k int) []Memory {
	type scored struct {
		mem   Memory
		score float64
	}
	var all []scored
	for _, mem := range mems {
		if mem.Kind == `reflection` {
			continue
		}
		all = append(all, scored{mem, scoreMemory(mem, ctx)})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score > all[j].score })
	if len(all) > k {
		all = all[:k]
	}
	out := make([]Memory, len(all))
	for i, s := range all {
		out[i] = s.mem
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Unix < out[j].Unix })
	return out
}

// markRecalled stamps the memories that were offered to the model, so the
// admin view can show what she has been thinking about.
func (m *Mind) markRecalled(chosen []Memory, nowUnix int64) {
	ids := map[int64]bool{}
	for _, c := range chosen {
		ids[c.Id] = true
	}
	for i := range m.Memories {
		if ids[m.Memories[i].Id] {
			m.Memories[i].RecalledUnix = nowUnix
		}
	}
}

// pruneMemories forgets the least important, least recent memories when the
// store is over max (F3.13). Memories of importance 8 or more are never
// forgotten; if only those remain over the cap, the cap is exceeded.
func pruneMemories(mems []Memory, max int, nowUnix int64) []Memory {
	if max <= 0 || len(mems) <= max {
		return mems
	}
	type cand struct {
		idx  int
		keep float64
	}
	var cands []cand
	for i, mem := range mems {
		if mem.Importance >= 8 {
			continue
		}
		cands = append(cands, cand{i, 0.6*float64(mem.Importance)/10.0 + 0.4*recency(nowUnix, mem.Unix)})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].keep < cands[j].keep })

	drop := map[int]bool{}
	excess := len(mems) - max
	for i := 0; i < excess && i < len(cands); i++ {
		drop[cands[i].idx] = true
	}
	out := make([]Memory, 0, len(mems)-len(drop))
	for i, mem := range mems {
		if !drop[i] {
			out = append(out, mem)
		}
	}
	return out
}

// isVague reports whether a memory has faded enough to be recalled only
// vaguely: older than thirty days and never very important.
func isVague(mem Memory, nowUnix int64) bool {
	return mem.Importance < 6 && nowUnix-mem.Unix > 30*86400
}
