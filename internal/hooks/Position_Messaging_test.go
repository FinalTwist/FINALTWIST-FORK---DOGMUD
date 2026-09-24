package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestBootLoadersRunEvenAfterTheOnceIsSpent is the guard on the two-tier
// loader policy's actual requirement, which is not "these loaders can panic"
// but "the check RUNS at boot".
//
// M3 already shipped this bug once: casting validated inside a sync.Once, so
// "fails at boot" was really "crashes on the first cast". Both grapple stores
// keep a lazy sync.Once for tests, so if LoadGrappleMessaging or
// LoadPositionMessages consulted that Once instead of spending it, a single
// earlier test, or in production a single earlier code path, would turn
// main.go's boot check into a silent no-op and nothing would say so.
//
// The probe is deliberately built on BROKEN data rather than absent data, so
// it cannot start passing for the wrong reason if something later seeds this
// test binary's temp world. Proven capable of failing on 2026-09-17: making
// LoadPositionMessages return early inside posMsgOnce.Do failed it by name.
func TestBootLoadersRunEvenAfterTheOnceIsSpent(t *testing.T) {
	// Spend both Onces through the lazy path. This is the state a boot would
	// find itself in if anything had touched the stores first.
	_ = loadPositionMessages()
	_ = loadGrappleLib()

	// Restore the package globals afterwards: a sibling test reading either
	// store must not see this probe's data.
	prevPos, prevGrapple := posMsgTemplates, grappleOutcomesLib
	t.Cleanup(func() { posMsgTemplates, grappleOutcomesLib = prevPos, prevGrapple })

	dir := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "messaging")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	const brokenYAML = "submission: [ this is not valid\n"
	for _, name := range []string{"position_control.yaml", "grapple_outcomes.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(brokenYAML), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustPanic(t, "LoadPositionMessages", LoadPositionMessages)
	mustPanic(t, "LoadGrappleMessaging", LoadGrappleMessaging)
}

// mustPanic fails when fn returns normally, naming the loader that did.
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s did not panic on broken data after its sync.Once was spent: "+
				"the boot check in main.go is a no-op, which is the M3 casting bug again", name)
		}
	}()
	fn()
}

// TestLoadPositionMessages_ParsesYAML verifies the YAML config at
// <configured world>/messaging/position_control.yaml is parseable. Skipped in
// environments without the data file, which in a test binary is the normal
// case: the config default world is _datafiles/world/default and the path is
// relative to this package's directory, so nothing resolves. The store's real
// build-time cover is TestShippedNarrationDataValidates at the repo root.
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
