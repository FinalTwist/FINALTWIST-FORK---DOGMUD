// Package conditionrename holds the one spelling map for conditions
// unification slice 3: every on-disk, wire and content spelling of "buff"
// becomes "condition". The Task 3 rewrite, the 0.17.0 save migration and the
// guard all read it, so the old and new spellings are listed nowhere else.
package conditionrename

import (
	"strconv"
	"strings"
)

// protected are case-sensitive words that contain "buff" but are not the
// concept. They are masked before any rename and restored after.
var protected = []string{
	"Buffer", "buffer", "BUFFER",
	"Buffet", "buffet",
	"buffed", "rebuff",
	"Buffalo",
}

// explicit are renames the generic case-preserving rule would get wrong,
// applied in order before it. Longer spellings come first so a shorter entry
// never splits a longer one.
var explicit = []struct{ old, new string }{
	{"melee_self_buff", "melee_self_empower"},
	{"aura_enemy_debuff", "aura_enemy_condition"},
	{"permabuff", "permanent"},
	{"PermaBuff", "Permanent"},
	{"Debuffs", "Harmful conditions"},
	{"debuffs", "harmful conditions"},
	{"Debuff", "Harmful condition"},
	{"debuff", "harmful condition"},
}

// generic is the case-preserving swap for everything else.
var generic = []struct{ old, new string }{
	{"BUFF", "CONDITION"},
	{"Buff", "Condition"},
	{"buff", "condition"},
}

const maskOpen, maskClose = "\x00", "\x01"

// Apply returns s with every buff spelling renamed. It is idempotent.
func Apply(s string) string {
	masked := s
	for i, word := range protected {
		masked = strings.ReplaceAll(masked, word, maskOpen+strconv.Itoa(i)+maskClose)
	}
	for _, r := range explicit {
		masked = strings.ReplaceAll(masked, r.old, r.new)
	}
	for _, r := range generic {
		masked = strings.ReplaceAll(masked, r.old, r.new)
	}
	for i, word := range protected {
		masked = strings.ReplaceAll(masked, maskOpen+strconv.Itoa(i)+maskClose, word)
	}
	return masked
}

// ContainsOldSpelling reports whether s still contains an old buff spelling
// outside the protected words. It masks protected words the same way Apply does, so
// removing one cannot splice its neighbours into a false match. Mixed-case
// spellings such as "BuFf" are reported here but not renamed by Apply; none
// exist in the repo (checked 2026-09-15), and the guard would surface one.
func ContainsOldSpelling(s string) bool {
	masked := s
	for i, word := range protected {
		masked = strings.ReplaceAll(masked, word, maskOpen+strconv.Itoa(i)+maskClose)
	}
	return strings.Contains(strings.ToLower(masked), "buff")
}
