package messaging

import "testing"

// The a/an stage must see through a run of <ansi ...> open tags between
// the article and the noun. A playtest showed "You sell a Ivory Handled
// Fan" because the item name arrives wrapped in <ansi fg="itemname">.
func TestNormalizeAAnSeesThroughAnsiTags(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			name: "one tag, lowercase article",
			in:   `you hold a <ansi fg="itemname">Ivory Handled Fan</ansi>.`,
			want: `You hold an <ansi fg="itemname">Ivory Handled Fan</ansi>.`,
		},
		{
			name: "one tag, capital article",
			in:   `A <ansi fg="itemname">apple</ansi> rolls away.`,
			want: `An <ansi fg="itemname">apple</ansi> rolls away.`,
		},
		{
			name: "nested tags keep every byte",
			in:   `you see a <ansi fg="red" bg="black"><ansi fg="itemname"><ansi fg="white-bold">old lamp</ansi></ansi></ansi>.`,
			want: `You see an <ansi fg="red" bg="black"><ansi fg="itemname"><ansi fg="white-bold">old lamp</ansi></ansi></ansi>.`,
		},
		{
			name: "consonant after tags stays a",
			in:   `you see a <ansi fg="itemname"><ansi fg="red">rusty sword</ansi></ansi>.`,
			want: `You see a <ansi fg="itemname"><ansi fg="red">rusty sword</ansi></ansi>.`,
		},
		{
			name: "tag without a following space is not an article",
			in:   `the <ansi fg="itemname">banana</ansi> is ripe.`,
			want: `The <ansi fg="itemname">banana</ansi> is ripe.`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Normalize(CategoryHitMelee, c.in)
			if got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// The exact line internal/usercommands/sell.go builds for a single sale.
func TestNormalizeAAnSellLine(t *testing.T) {
	in := `You sell a <ansi fg="itemname">Ivory Handled Fan</ansi> for <ansi fg="gold">75 gold</ansi>.`
	want := `You sell an <ansi fg="itemname">Ivory Handled Fan</ansi> for <ansi fg="gold">75 gold</ansi>.`
	if got := Normalize(CategorySystem, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestNormalizeAAnThroughTagsIdempotent(t *testing.T) {
	for _, in := range []string{
		`you sell a <ansi fg="itemname">Ivory Handled Fan</ansi>`,
		`A <ansi fg="a"><ansi fg="b">apple</ansi></ansi>`,
		`a <ansi fg="itemname">rusty sword</ansi>`,
	} {
		once := Normalize(CategoryHitMelee, in)
		twice := Normalize(CategoryHitMelee, once)
		if once != twice {
			t.Errorf("not idempotent for %q: %q vs %q", in, once, twice)
		}
	}
}

// Categories that opt out of normalization keep "a" even before a tagged
// vowel.
func TestNormalizeAAnThroughTagsSkippedForOptOutCategory(t *testing.T) {
	in := `a <ansi fg="itemname">apple</ansi> lies here`
	if got := Normalize(CategoryRoomDescription, in); got != in {
		t.Errorf("CategoryRoomDescription must skip a/an, got %q", got)
	}
}
