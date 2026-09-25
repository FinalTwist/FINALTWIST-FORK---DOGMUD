package lightnotice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

var (
	termsLit    = rooms.LightTerms{Level: 60, Sky: 60}
	termsDim    = rooms.LightTerms{Level: 40, Sky: 40}
	termsBright = rooms.LightTerms{Level: 80, Sky: 80}
)

func rec(room int, b messaging.Band, terms rooms.LightTerms) record {
	return record{roomId: room, band: b, terms: terms}
}

func obs(room int, b messaging.Band, terms rooms.LightTerms) observation {
	return observation{roomId: room, band: b, terms: terms}
}

func TestDecideTriggerRules(t *testing.T) {
	cases := []struct {
		name  string
		prev  record
		now   observation
		trig  Trigger
		speak bool
		tr    Transition
	}{
		{"move to darker speaks", rec(1, messaging.BandFaces, termsLit), obs(2, messaging.BandShapes, termsDim), TriggerMove, true, DarkerShapes},
		{"move to lighter is silent", rec(1, messaging.BandShapes, termsDim), obs(2, messaging.BandFaces, termsLit), TriggerMove, false, ""},
		{"move into dazzle speaks", rec(1, messaging.BandFaces, termsLit), obs(2, messaging.BandDazzled, termsBright), TriggerMove, true, IntoDazzle},
		{"combat darker speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerCombatRound, true, DarkerShapes},
		{"combat lighter speaks", rec(1, messaging.BandShapes, termsDim), obs(1, messaging.BandFaces, termsLit), TriggerCombatRound, true, LighterFaces},
		{"command darker speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerCommand, true, DarkerShapes},
		{"command lighter speaks", rec(1, messaging.BandShapes, termsDim), obs(1, messaging.BandFaces, termsLit), TriggerCommand, true, LighterFaces},
		{"quiet never speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerQuiet, false, ""},
		{"same band is silent", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandFaces, termsDim), TriggerCommand, false, ""},
	}
	for _, c := range cases {
		n, speak, next := decide(c.prev, true, c.now, c.trig)
		if speak != c.speak {
			t.Errorf("%s: speak = %v, want %v", c.name, speak, c.speak)
		}
		if speak && n.transition != c.tr {
			t.Errorf("%s: transition = %q, want %q", c.name, n.transition, c.tr)
		}
		if next.band != c.now.band || next.roomId != c.now.roomId || next.quiet {
			t.Errorf("%s: the observed band must always be recorded, got %+v", c.name, next)
		}
	}
}

func TestDecideSilentRecording(t *testing.T) {
	now := obs(1, messaging.BandShapes, termsDim)

	if _, speak, next := decide(record{}, false, now, TriggerCommand); speak || next.band != messaging.BandShapes {
		t.Errorf("first check must record silently, speak=%v next=%+v", speak, next)
	}

	asleep := now
	asleep.asleep = true
	_, speak, next := decide(rec(1, messaging.BandFaces, termsLit), true, asleep, TriggerCommand)
	if speak || !next.quiet {
		t.Errorf("a sleeper gets no notice and is marked quiet, speak=%v next=%+v", speak, next)
	}

	blind := now
	blind.blinded = true
	_, speak, next = decide(rec(1, messaging.BandFaces, termsLit), true, blind, TriggerCombatRound)
	if speak || !next.quiet {
		t.Errorf("a blinded player gets no notice and is marked quiet, speak=%v next=%+v", speak, next)
	}

	quiet := rec(1, messaging.BandFaces, termsLit)
	quiet.quiet = true
	_, speak, next = decide(quiet, true, now, TriggerCommand)
	if speak || next.quiet || next.band != messaging.BandShapes {
		t.Errorf("the first check after quiet records silently and clears quiet, speak=%v next=%+v", speak, next)
	}

	_, speak, _ = decide(next, true, obs(1, messaging.BandFaces, termsLit), TriggerCommand)
	if !speak {
		t.Error("the check after a silent resync must speak on a real change")
	}
}

func TestTransitionOf(t *testing.T) {
	cases := []struct {
		from, to messaging.Band
		want     Transition
	}{
		{messaging.BandDark, messaging.BandShapes, LighterShapes},
		{messaging.BandDark, messaging.BandFaces, LighterFaces},
		{messaging.BandShapes, messaging.BandFaces, LighterFaces},
		{messaging.BandDark, messaging.BandDazzled, IntoDazzle},
		{messaging.BandFaces, messaging.BandDazzled, IntoDazzle},
		{messaging.BandDazzled, messaging.BandFaces, DarkerFaces},
		{messaging.BandDazzled, messaging.BandShapes, DarkerShapes},
		{messaging.BandFaces, messaging.BandShapes, DarkerShapes},
		{messaging.BandFaces, messaging.BandDark, DarkerDark},
		{messaging.BandDazzled, messaging.BandDark, DarkerDark},
	}
	for _, c := range cases {
		if got := transitionOf(c.from, c.to); got != c.want {
			t.Errorf("%v -> %v = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestAttribution(t *testing.T) {
	base := rooms.LightTerms{Level: 60, Sky: 55, Lamp: 40, HasLamp: true}
	with := func(f func(*rooms.LightTerms)) rooms.LightTerms { t2 := base; f(&t2); return t2 }

	cases := []struct {
		name string
		prev record
		now  observation
		want Cause
	}{
		{"a different room is movement",
			rec(1, messaging.BandFaces, base), obs(2, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseMovement},
		{"carried light leaving",
			rec(1, messaging.BandFaces, with(func(x *rooms.LightTerms) { x.Carried = true })), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseCarried},
		{"lamp value changing",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Lamp = 20 })), CauseLamp},
		{"LightMod bridge changing is the lamp",
			rec(1, messaging.BandFaces, with(func(x *rooms.LightTerms) { x.LightMod = 2 })), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseLamp},
		{"weather occlusion",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.OcclusionSteps = 1; x.Sky = 30 })), CauseWeather},
		{"the sky alone",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Sky = 30 })), CauseSky},
		{"sky gone entirely counts as the sky",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Sky = lightscale.Absent() })), CauseSky},
		{"level unchanged is the eyes",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, base), CauseEyes},
		{"level rose but the band fell is the eyes, not the sky",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 61; x.Sky = 56 })), CauseEyes},
	}
	for _, c := range cases {
		tr := transitionOf(c.prev.band, c.now.band)
		if got := attribute(c.prev, c.now, tr); got != c.want {
			t.Errorf("%s: cause = %q, want %q", c.name, got, c.want)
		}
	}
}
