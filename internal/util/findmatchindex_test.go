package util

import "testing"

// FindMatchIndexIn is FindMatchIn by position (baubles slice D): the same
// algorithm, so a shop's two rows with one name are told apart by index and
// `buy 2.trinket` takes the second. The cases are TestFindMatchIn's.
func TestFindMatchIndexIn_SameAnswersAsFindMatchIn(t *testing.T) {
	items := []string{"SWORD", "SHINING SWORD", "SHIELD", "BIG HELM", "HELMET", "GEM"}
	cases := []struct {
		search          string
		match, closeIdx int
	}{
		{"", -1, -1},
		{"SWORD", 0, 0},
		{"sword#2", -1, 1},
		{"HELM", -1, 4},
		{"G", -1, 5},
		{"iel", -1, 2},
		{"helm#2", -1, 4},
	}
	for _, tc := range cases {
		m, c := FindMatchIndexIn(tc.search, items...)
		if m != tc.match || c != tc.closeIdx {
			t.Errorf("FindMatchIndexIn(%q) = (%d, %d), want (%d, %d)", tc.search, m, c, tc.match, tc.closeIdx)
		}
		wantM, wantC := "", ""
		if tc.match >= 0 {
			wantM = items[tc.match]
		}
		if tc.closeIdx >= 0 {
			wantC = items[tc.closeIdx]
		}
		if gm, gc := FindMatchIn(tc.search, items...); gm != wantM || gc != wantC {
			t.Errorf("FindMatchIn(%q) = (%q, %q), want (%q, %q): the two must agree", tc.search, gm, gc, wantM, wantC)
		}
	}
}

// Two entries with one name: the index says which one.
func TestFindMatchIndexIn_TellsSameNamesApart(t *testing.T) {
	names := []string{"Iron Sword", "Trinket", "Trinket"}
	for search, want := range map[string][2]int{
		"trinket":   {1, 1},
		"2.trinket": {2, 2},
		"trinket#2": {2, 2},
		"3.trinket": {-1, -1},
		"trink":     {-1, 1},
		"2.trink":   {-1, 2},
	} {
		m, c := FindMatchIndexIn(search, names...)
		if m != want[0] || c != want[1] {
			t.Errorf("FindMatchIndexIn(%q) = (%d, %d), want (%d, %d)", search, m, c, want[0], want[1])
		}
	}
}
