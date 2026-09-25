package rooms

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// LightLevel reports the room's light on the graded -100 to 100 scale.
//
// Three terms compose it, all on one logarithmic operator:
//
//  1. The sky, which is the celestial term attenuated by this room's sky
//     fraction. A room with no sky receives no term at all, which is not the
//     same as receiving a term of zero.
//  2. The room's own lamp, if it has one, joining the combine rather than
//     acting as a floor, so a lantern-lit tavern plus a carried torch does not
//     double-count.
//  3. Anyone in the room carrying a light.
//
// Weather and mutators attenuate the SKY only: a blizzard does not dim a
// lantern. Plan 4 gives them a real occlusion fraction; until then the old
// -2 to 2 LightMod vocabulary is bridged, one point per doubling step.
func (r *Room) LightLevel() int {
	return r.lightLevel(configs.GetLightingConfig(), gametime.CelestialLight())
}

// IsLit reports whether a normal observer can see anything at all here.
//
// 🔑 This is the predicate plan 1's design promised and never built. Fifteen
// call sites were hand-rolling `LightLevel() >= GetBalanceConfig().LightBlindBelow`,
// each copying a 424-field struct (99.75 ns measured) to read one int. This
// reads the narrow lighting config once.
func (r *Room) IsLit() bool {
	cfg := configs.GetLightingConfig()
	return r.lightLevel(cfg, gametime.CelestialLight()) >= cfg.BlindBelow
}

// lightLevel is LightLevel with its two reads injected, so it is testable
// without global state and so a caller holding both can avoid reading twice.
func (r *Room) lightLevel(cfg configs.Lighting, celestial float64) int {
	lightMod, occlusionSteps := r.mutatorLightTerms()
	return r.lightLevelWithMutatorBridge(cfg, celestial, lightMod, occlusionSteps)
}

// lightLevelWithMutatorBridge is the composition itself, with the mutator
// contribution already summarised, so tests can drive it directly.
//
// lightMod is the total POSITIVE LightMod across active mutators, and
// occlusionSteps the total NEGATIVE, expressed as doubling steps of sky removed.
func (r *Room) lightLevelWithMutatorBridge(cfg configs.Lighting, celestial float64, lightMod, occlusionSteps int) int {
	return r.composeLight(cfg, celestial, lightMod, occlusionSteps).Level
}

// LightTerms is the room's light broken into the terms LightLevel combines,
// for a caller that needs to know WHY the light is what it is.
// internal/lightnotice names the cause of a band change from them.
type LightTerms struct {
	// Level is exactly LightLevel(): both come from composeLight.
	Level int
	// Sky is the sky term after the sky fraction and weather occlusion, in
	// light-scale units; lightscale.Absent() when the room has no sky.
	Sky float64
	// OcclusionSteps is the doubling steps of sky removed by weather mutators.
	OcclusionSteps int
	// Lamp is the room's own lamp; 0 when HasLamp is false.
	Lamp    int
	HasLamp bool
	// LightMod is the positive LightMod bridge total.
	LightMod int
	// Carried reports that someone in the room carries a light.
	Carried bool
}

// LightTerms reports the terms behind LightLevel, from the same single
// computation.
func (r *Room) LightTerms() LightTerms {
	lightMod, occlusionSteps := r.mutatorLightTerms()
	return r.composeLight(configs.GetLightingConfig(), gametime.CelestialLight(), lightMod, occlusionSteps)
}

// composeLight is the one computation behind LightLevel and LightTerms.
func (r *Room) composeLight(cfg configs.Lighting, celestial float64, lightMod, occlusionSteps int) LightTerms {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	out := LightTerms{OcclusionSteps: occlusionSteps}
	terms := make([]float64, 0, 4)

	// 1. The sky, attenuated by this room's fraction and then by any weather
	// blocking it. Attenuate returns Absent for a fraction of zero, so a cave
	// contributes no term rather than a term of zero.
	sky := r.skyLightFraction()
	if occlusionSteps > 0 {
		sky *= math.Exp2(-float64(occlusionSteps))
	}
	out.Sky = lightscale.Attenuate(step, celestial, sky)
	terms = append(terms, out.Sky)

	// 2. The room's own lamp.
	if lamp, ok := r.lampValue(); ok {
		out.Lamp, out.HasLamp = lamp, true
		terms = append(terms, float64(lamp))
	}

	// 3. The positive LightMod bridge. A +1 mutator lands exactly on
	// LightDimBelow and +2 one step above it, so the 31 Crash Site Interior
	// rooms and 12 Foldweave rooms that a static `lightmod: 2` holds lit today
	// stay fully visible. Plan 4 replaces this with an authored lamp value.
	if lightMod > 0 {
		out.LightMod = lightMod
		terms = append(terms, float64(cfg.DimBelow)+float64(lightMod-1)*step)
	}

	// 4. Anyone carrying a light. Plan 5 gives carried sources real magnitudes
	// that scale from stat and skill; until then any light source lifts the
	// room to the bottom of the perfect band, which is what the old model's
	// "someone has light, cancel the darkness" rule effectively did.
	if len(r.GetMobs(FindHasLight)) > 0 || len(r.GetPlayers(FindHasLight)) > 0 {
		out.Carried = true
		terms = append(terms, float64(cfg.DimBelow))
	}

	v := lightscale.Combine(step, terms...)
	if math.IsInf(v, -1) {
		// No light of any kind. Zero is the darkest light that NATURALLY
		// occurs, which is what an unlit cave is. Magical darkness goes below
		// this and arrives in plan 5.
		v = 0
	}

	n := int(math.Round(v))
	if n < -100 {
		n = -100
	} else if n > 100 {
		n = 100
	}
	out.Level = n
	return out
}

// mutatorLightTerms sums the active mutators' LightMod into a positive
// contribution and a count of negative doubling steps.
//
// ⚠️ This is a BRIDGE. The -2 to 2 LightMod vocabulary predates the graded
// scale and plan 4 retires it in favour of an authored occlusion fraction and
// lamp value. Do not extend it.
func (r *Room) mutatorLightTerms() (lightMod, occlusionSteps int) {
	for mut := range r.ActiveMutators {
		spec := mut.GetSpec()
		if spec == nil || spec.LightMod == 0 {
			continue
		}
		if spec.LightMod > 0 {
			lightMod += spec.LightMod
		} else {
			occlusionSteps += -spec.LightMod
		}
	}
	return lightMod, occlusionSteps
}

// skyLightFraction is this room's sky fraction: its own override if it has one,
// otherwise its biome's.
//
// ⚠️ GetBiome can return nil when the biome registry has not been loaded, which
// is the normal state in a unit test that does not read _datafiles. A nil check
// here is not defensive padding: without it every table-driven lighting test
// must load the whole world first, and a nil dereference in LightLevel would
// take down a live room read.
func (r *Room) skyLightFraction() float64 {
	if r.SkyLight != nil {
		return *r.SkyLight
	}
	if b := r.GetBiome(); b != nil {
		return b.SkyLightFraction()
	}
	return 1.0
}

// lampValue is this room's own light source, and whether it has one at all.
// Nil-safe for the same reason as skyLightFraction.
func (r *Room) lampValue() (int, bool) {
	if r.Lamp != nil {
		return *r.Lamp, true
	}
	if b := r.GetBiome(); b != nil {
		return b.LampValue()
	}
	return 0, false
}
