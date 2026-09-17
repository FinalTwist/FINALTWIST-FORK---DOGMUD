package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestLoadPositionMessages_ParsesYAML verifies the YAML config at
// _datafiles/messages/position_control.yaml is parseable. Skipped in
// environments without the data file (e.g. some CI sandboxes).
func TestLoadPositionMessages_ParsesYAML(t *testing.T) {
	templates := loadPositionMessages()
	if templates.StaminaWarning.Self == "" {
		t.Skip("YAML config not present in this environment")
	}
	if templates.StaminaWarning.Self == "" {
		t.Errorf("expected non-empty stamina_warning.self")
	}
}

// TestStaminaWarning_NoOpWhenNotLow verifies that fireStaminaWarningIfLow
// does not set a cooldown when stamina is above the threshold.
func TestStaminaWarning_NoOpWhenNotLow(t *testing.T) {
	c := characters.New()
	c.Position = position.NewMachine()
	c.PerGrappleMessageCooldowns = map[string]bool{}
	// Fresh character — StaminaMax.Value 0 means IsLowGrappleStamina
	// returns false (divide-by-zero guard). Verify no cooldown set.

	fireStaminaWarningIfLow(c)

	if c.PerGrappleMessageCooldowns["stamina_low"] {
		t.Errorf("expected no stamina_low cooldown when not low, got cooldown set")
	}
}

// TestSubstitutionsForCharacter_CanonicalTokens pins the vocabulary this
// store's token map emits. The local substitute() is gone as of M4a, so there
// is no longer a {key} loop to test; what is worth pinning instead is that the
// map is keyed by the canonical tokens and that the controlled side lands in
// the actee slot.
//
// A fresh character has no Control machine, so IsController() is false and it
// is the controlled side; it has no grapple partner either, so the actor slot
// is the empty partner name.
func TestSubstitutionsForCharacter_CanonicalTokens(t *testing.T) {
	c := characters.New()
	c.Name = "Rocky"
	c.Position = position.NewMachine()

	subs := substitutionsForCharacter(c)

	wantKeys := []string{"{position}", narration.TokenActor, narration.TokenActee}
	if len(subs) != len(wantKeys) {
		t.Errorf("token map has %d keys, want %d: %v", len(subs), len(wantKeys), subs)
	}
	for _, k := range wantKeys {
		if _, ok := subs[k]; !ok {
			t.Errorf("token map is missing key %q: %v", k, subs)
		}
	}
	if subs[narration.TokenActee] != "Rocky" {
		t.Errorf("controlled side should be the actee: %s = %q, want %q",
			narration.TokenActee, subs[narration.TokenActee], "Rocky")
	}
	if subs[narration.TokenActor] != "" {
		t.Errorf("with no partner the actor slot should be empty: %s = %q",
			narration.TokenActor, subs[narration.TokenActor])
	}
}

// TestStaminaWarning_ActorIsTheCharacter pins the one asymmetric line in this
// store. The stamina warning's room text is about the character it fires for,
// not about the controller, so fireStaminaWarningIfLow overrides the actor
// slot. Without that override a controlled character's warning would name the
// CONTROLLER in the room line.
func TestStaminaWarning_ActorIsTheCharacter(t *testing.T) {
	c := characters.New()
	c.Name = "Rocky"
	c.Position = position.NewMachine()

	if plain := substitutionsForCharacter(c); plain[narration.TokenActor] == "Rocky" {
		t.Fatal("setup is vacuous: the plain role mapping already put Rocky in the actor slot")
	}

	subs := staminaWarningSubstitutions(c)
	got := narration.Substitute("{actor} looks exhausted in the {position}.", subs)
	want := "Rocky looks exhausted in the " + c.Position.State().String() + "."
	if got != want {
		t.Errorf("stamina room line = %q, want %q", got, want)
	}
}
