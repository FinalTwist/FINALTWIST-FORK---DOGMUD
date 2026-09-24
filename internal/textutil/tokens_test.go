package textutil

import "testing"

func TestSubstituteTokens_AllTokens(t *testing.T) {
	ctx := TokenContext{
		ActorName:      `<ansi fg="yellow">Kael</ansi>`,
		ActorPlainName: `Kael`,
		ActeeName:      `<ansi fg="red">Goblin</ansi>`,
		ActeePlainName: `Goblin`,
	}
	input := `{actor} hurls a bolt at {actee}. {actor_plain}'s eyes glow. {actee_plain} staggers.`
	expected := `<ansi fg="yellow">Kael</ansi> hurls a bolt at <ansi fg="red">Goblin</ansi>. Kael's eyes glow. Goblin staggers.`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_EmptyActee(t *testing.T) {
	ctx := TokenContext{
		ActorName:      `Kael`,
		ActorPlainName: `Kael`,
	}
	input := `{actor} channels energy at {actee}.`
	expected := `Kael channels energy at .`
	result := SubstituteTokens(input, ctx)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSubstituteTokens_NoTokens(t *testing.T) {
	ctx := TokenContext{ActorName: `Kael`}
	input := `Energy crackles in the air.`
	result := SubstituteTokens(input, ctx)
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestSubstituteTokens_EmptyString(t *testing.T) {
	ctx := TokenContext{}
	result := SubstituteTokens("", ctx)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestValidateTokens_KnownTokens(t *testing.T) {
	warnings := ValidateTokens(`{actor} attacks {actee}`)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestValidateTokens_UnknownToken(t *testing.T) {
	warnings := ValidateTokens(`{actor} attacks {actae}`)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if warnings[0] != `unknown token: {actae}` {
		t.Errorf("got %q", warnings[0])
	}
}

func TestValidateTokens_EmptyString(t *testing.T) {
	warnings := ValidateTokens("")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}
