package crafting

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/textutil"
)

func narrationTestRecipe() *RecipeSpec {
	return &RecipeSpec{
		RecipeId:       "test-stew",
		Name:           "Test Stew",
		Skill:          "cooking",
		Output:         RecipeOutput{ItemId: 30022, Quantity: 1},
		SuccessMessage: "You stir the pot. Stew!",
		FailureMessage: "The stew burns.",
	}
}

var narrationTestCrafter = textutil.TokenContext{
	ActorName:      `<ansi fg="username">Aliceia</ansi>`,
	ActorPlainName: "Aliceia",
}

func TestRecipeNarration_ActorIsTheCrafterLine(t *testing.T) {
	r := narrationTestRecipe()

	success := r.Narration(PhaseSuccess)
	if len(success.Actor) != 1 || success.Actor[0] != r.SuccessMessage {
		t.Errorf("success Actor = %q, want [%q]", success.Actor, r.SuccessMessage)
	}
	if len(success.Observer) != 0 || len(success.Actee) != 0 {
		t.Errorf("success with no room message: Observer %q Actee %q, want both empty", success.Observer, success.Actee)
	}

	failure := r.Narration(PhaseFailure)
	if len(failure.Actor) != 1 || failure.Actor[0] != r.FailureMessage {
		t.Errorf("failure Actor = %q, want [%q]", failure.Actor, r.FailureMessage)
	}
}

func TestRecipeNarrate_ObserverNamesTheCrafter(t *testing.T) {
	r := narrationTestRecipe()
	r.SuccessRoomMessage = "{actor} ladles out a steaming stew."
	r.FailureRoomMessage = "Smoke pours from {actor}'s pot."

	s := r.Narrate(PhaseSuccess, narrationTestCrafter)
	if s.Actor != r.SuccessMessage {
		t.Errorf("success Actor = %q, want %q", s.Actor, r.SuccessMessage)
	}
	if want := `<ansi fg="username">Aliceia</ansi> ladles out a steaming stew.`; s.Observer != want {
		t.Errorf("success Observer = %q, want %q", s.Observer, want)
	}

	f := r.Narrate(PhaseFailure, narrationTestCrafter)
	if want := `Smoke pours from <ansi fg="username">Aliceia</ansi>'s pot.`; f.Observer != want {
		t.Errorf("failure Observer = %q, want %q", f.Observer, want)
	}
}

func TestRecipeValidate_Narration(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(r *RecipeSpec)
		wantErr string
	}{
		{"shipped shape passes", func(r *RecipeSpec) {}, ""},
		{"room lines naming the crafter pass", func(r *RecipeSpec) {
			r.SuccessRoomMessage = "{actor} finishes a stew."
			r.FailureRoomMessage = "{actor} burns a stew."
		}, ""},
		{"empty success refused", func(r *RecipeSpec) { r.SuccessMessage = "" }, "success_message cannot be empty"},
		{"empty failure refused", func(r *RecipeSpec) { r.FailureMessage = "" }, "failure_message cannot be empty"},
		{"whitespace failure refused", func(r *RecipeSpec) { r.FailureMessage = "   " }, "is empty"},
		{"whitespace room line refused", func(r *RecipeSpec) { r.SuccessRoomMessage = "  " }, "is empty"},
		{"room line without source refused", func(r *RecipeSpec) { r.FailureRoomMessage = "A pot burns." }, "failure_room_message must name the crafter with {actor}"},
		{"success room line without source refused", func(r *RecipeSpec) { r.SuccessRoomMessage = "A stew is finished." }, "success_room_message must name the crafter with {actor}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := narrationTestRecipe()
			tc.mutate(r)
			err := r.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestMobRoomLine_AuthoredLineElseFallback(t *testing.T) {
	r := narrationTestRecipe()
	const fallback = `<ansi fg="mobname">Smith</ansi> finishes their work.`

	if got := r.MobRoomLine(PhaseSuccess, "Smith", fallback); got != fallback {
		t.Errorf("no authored room line: got %q, want the fallback %q", got, fallback)
	}
	if got := r.MobRoomLine(PhaseFailure, "Smith", ""); got != "" {
		t.Errorf("no authored failure line and no fallback: got %q, want empty", got)
	}

	r.SuccessRoomMessage = "{actor} sets down a finished stew."
	if want := `<ansi fg="mobname">Smith</ansi> sets down a finished stew.`; r.MobRoomLine(PhaseSuccess, "Smith", fallback) != want {
		t.Errorf("authored room line: got %q, want %q", r.MobRoomLine(PhaseSuccess, "Smith", fallback), want)
	}
}
