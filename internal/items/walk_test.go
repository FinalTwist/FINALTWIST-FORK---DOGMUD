package items

import "testing"

// WalkSlice visits real items only (an empty or disabled slot is skipped),
// in order, through pointers into the slice itself.
func TestWalkSliceVisitsRealItemsInPlace(t *testing.T) {
	s := []Item{{ItemId: 5}, {}, ItemDisabledSlot, {ItemId: 7}}
	seen := []int{}
	WalkSlice(s, func(it *Item) {
		seen = append(seen, it.ItemId)
		it.Uses = 3
	})
	if len(seen) != 2 || seen[0] != 5 || seen[1] != 7 {
		t.Fatalf("visited %v, want [5 7]", seen)
	}
	if s[0].Uses != 3 || s[3].Uses != 3 {
		t.Fatal("the pointers are into the slice itself")
	}
	WalkSlice(nil, func(*Item) { t.Fatal("nothing to visit in nil") })
}
