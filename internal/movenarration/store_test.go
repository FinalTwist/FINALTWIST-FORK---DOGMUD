package movenarration

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/narration"
)

func TestEventVariantsRoundTrip(t *testing.T) {
	g := &MoveNarrationGroup{
		MoveId: "kick",
		Events: map[EventKey]*EventMessages{
			"standard_hit": {
				Actor:    []string{`You kick {actee} hard! ({damage})`},
				Actee:    []string{`{actor} kicks you hard! ({damage})`},
				Observer: []string{`{actor} kicks {actee}!`},
			},
		},
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	v, ok := g.Variants("standard_hit")
	if !ok {
		t.Fatal(`Variants("standard_hit") not found`)
	}
	if v.Len() != 1 {
		t.Fatalf("Len = %d, want 1", v.Len())
	}
	roles := narration.Render(v, map[string]string{
		narration.TokenActor: `<ansi fg="mobname">Goblin</ansi>`,
		narration.TokenActee: `<ansi fg="username">Kesh</ansi>`,
		TokenDamage:          `<ansi fg="damage">a solid hit</ansi>`,
	}, narration.FirstPicker)
	want := `<ansi fg="mobname">Goblin</ansi> kicks you hard! (<ansi fg="damage">a solid hit</ansi>)`
	if roles.Actee != want {
		t.Errorf("Actee =\n%q\nwant\n%q", roles.Actee, want)
	}
}

func TestValidateRejectsRaggedPools(t *testing.T) {
	g := &MoveNarrationGroup{
		MoveId: "kick",
		Events: map[EventKey]*EventMessages{
			"standard_hit": {
				Actee:    []string{`a`, `b`},
				Observer: []string{`c`},
			},
		},
	}
	if err := g.Validate(); err == nil {
		t.Fatal("Validate accepted ragged pools; it must reject them")
	}
}

func TestVariantsMissingEventReportsNotFound(t *testing.T) {
	g := &MoveNarrationGroup{MoveId: "kick", Events: map[EventKey]*EventMessages{}}
	if _, ok := g.Variants("nope"); ok {
		t.Fatal(`Variants("nope") reported found for an absent event`)
	}
}
