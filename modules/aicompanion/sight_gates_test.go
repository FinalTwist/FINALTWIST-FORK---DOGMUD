package aicompanion

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Her `remove` refuses a cursed worn item up front, as `get` refuses a
// household's bauble, so she does not record a futile attempt (spec R8).
func TestCompanionRemoveRefusesACursedItem(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	ring := items.Item{ItemId: 96501, Spec: &items.ItemSpec{ItemId: 96501, Name: "hexed ring", Type: items.Ring, Subtype: items.Wearable, Cursed: true}}
	her.Character.Equipment.Ring = ring
	m, c, _ := strangerModule()
	sc := &scene{RoomId: room.RoomId, byRef: map[string]*thing{}}
	sc.byRef[`w1`] = &thing{Ref: `w1`, Kind: `worn`, Name: `Hexed Ring`, Item: ring, HasItem: true}
	out := m.performAction(c, her, owner, sc, ActionProposal{Verb: `remove`, Ref: `w1`},
		[]stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if out.Issued || out.Refused != `it will not come off` {
		t.Fatalf("want a refusal before any command, got %+v", out)
	}
}

// She neither offers nor starts a recipe she cannot see to make (spec C5).
func TestCraftableHereIsEmptyInTheDark(t *testing.T) {
	_, _, room, her := harmWorld(t, configs.PVPDisabled)
	crafting.RegisterRecipeForTest(&crafting.RecipeSpec{RecipeId: `sg-twine`, Name: `Twine`, Skill: `tailoring`})
	t.Cleanup(func() { crafting.UnregisterRecipeForTest(`sg-twine`) })
	her.Character.KnownRecipes = map[string]int{`sg-twine`: 1}
	p := &Profile{Crafts: []string{`tailoring`}}

	if got := craftableHere(her, p, room); len(got) != 1 {
		t.Fatalf("control: lit, she can make twine, got %+v", got)
	}
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	if got := craftableHere(her, p, room); len(got) != 0 {
		t.Fatalf("dark: nothing is craftable here, got %+v", got)
	}
}

// A sleeping companion in a lit room offers no recipes: `craftableHere` must
// not disagree with `actions.TooDarkToCraft`, which the actual craft attempt
// asks (InitiateCraft). Before this, `craftableHere` used `cannotSee`, which
// does not consult sleep, so a sleeping companion listed recipes that the
// craft itself then silently refused.
func TestCraftableHereIsEmptyWhenSleepingInALitRoom(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		15: {ConditionId: 15, Name: "Sleeping", Flags: []conditions.Flag{conditions.Sleeping}, TriggerCount: 1000000},
	}))
	_, _, room, her := harmWorld(t, configs.PVPDisabled)
	crafting.RegisterRecipeForTest(&crafting.RecipeSpec{RecipeId: `sg-twine-sleep`, Name: `Twine`, Skill: `tailoring`})
	t.Cleanup(func() { crafting.UnregisterRecipeForTest(`sg-twine-sleep`) })
	her.Character.KnownRecipes = map[string]int{`sg-twine-sleep`: 1}
	p := &Profile{Crafts: []string{`tailoring`}}

	if got := craftableHere(her, p, room); len(got) != 1 {
		t.Fatalf("control: lit and awake, she can make twine, got %+v", got)
	}
	if err := her.Character.AddCondition(15, false); err != nil {
		t.Fatalf("could not put her to sleep: %v", err)
	}
	if got := craftableHere(her, p, room); len(got) != 0 {
		t.Fatalf("asleep in a lit room: nothing is craftable here, got %+v", got)
	}
}
