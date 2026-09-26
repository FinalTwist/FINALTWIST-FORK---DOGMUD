package rooms

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
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
//  3. Everything anyone in the room carries, one term per light.
//
// Weather attenuates the SKY only, through each active mutator's skylight
// fraction: a blizzard does not dim a lantern.
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
	return r.lightLevelWithSkyFilter(cfg, celestial, r.mutatorSkyFilter())
}

// lightLevelWithSkyFilter is the composition itself, with the active mutators'
// sky filter already multiplied out, so tests can drive it directly.
func (r *Room) lightLevelWithSkyFilter(cfg configs.Lighting, celestial, skyFilter float64) int {
	return r.composeLight(cfg, celestial, skyFilter).Level
}

// LightTerms is the room's light broken into the terms LightLevel combines,
// for a caller that needs to know WHY the light is what it is.
// internal/lightnotice names the cause of a band change from them.
type LightTerms struct {
	// Level is exactly LightLevel(): both come from composeLight.
	Level int
	// Raw is the combined light before rounding and clamping; Absent when
	// nothing lights the room. A trim solves against it.
	Raw float64
	// Sky is the sky term after the sky fraction and the weather filter, in
	// light-scale units; lightscale.Absent() when the room has no sky.
	Sky float64
	// SkyFilter is the fraction of the sky active weather lets through: the
	// product of the active mutators' skylight values, 1 when clear.
	SkyFilter float64
	// Lamp is the room's own lamp; 0 when HasLamp is false.
	Lamp    int
	HasLamp bool
	// Carried reports that someone in the room carries a light.
	Carried bool
}

// LightTerms reports the terms behind LightLevel, from the same single
// computation.
func (r *Room) LightTerms() LightTerms {
	return r.composeLight(configs.GetLightingConfig(), gametime.CelestialLight(), r.mutatorSkyFilter())
}

// composeLight is the one computation behind LightLevel and LightTerms.
func (r *Room) composeLight(cfg configs.Lighting, celestial, skyFilter float64) LightTerms {
	return r.composeLightExcluding(cfg, celestial, skyFilter, nil)
}

// composeLightExcluding is composeLight with one carried record left out, which
// is the room a trimming source sees: everything except itself.
func (r *Room) composeLightExcluding(cfg configs.Lighting, celestial, skyFilter float64, exclude *conditions.Condition) LightTerms {
	return r.composeWith(cfg, celestial, skyFilter, r.carriedLight(exclude))
}

// composeWith is the composition with the carried terms supplied, so a test
// needs no users or mobs.
func (r *Room) composeWith(cfg configs.Lighting, celestial, skyFilter float64, carried []float64) LightTerms {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	out := LightTerms{SkyFilter: skyFilter}
	terms := make([]float64, 0, 2+len(carried))

	// 1. The sky, attenuated by this room's fraction and then by any weather
	// filtering it. A filter multiplies the fraction, which on this log scale
	// is a fixed subtraction: 0.5 removes one doubling step at any hour, so a
	// storm is merely gloomy at noon and blinding at midnight. Attenuate
	// returns Absent for a fraction of zero, so a cave contributes no term
	// rather than a term of zero.
	out.Sky = lightscale.Attenuate(step, celestial, r.skyLightFraction()*skyFilter)
	terms = append(terms, out.Sky)

	// 2. The room's own lamp.
	if lamp, ok := r.lampValue(); ok {
		out.Lamp, out.HasLamp = lamp, true
		terms = append(terms, float64(lamp))
	}

	// 3. Every light anyone here carries, each its own term (lighting plan 5a):
	// a candle and a torch are different sources, and two torches are one
	// doubling step brighter than one.
	if len(carried) > 0 {
		out.Carried = true
		terms = append(terms, carried...)
	}

	v := lightscale.Combine(step, terms...)
	out.Raw = v
	if math.IsInf(v, -1) {
		// No light of any kind. Zero is the darkest light that NATURALLY
		// occurs, which is what an unlit cave is. Magical darkness goes below
		// this and arrives in plan 5d.
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

// carriedLight is every carried light term in the room, leaving out one record
// (the source being trimmed) when exclude is non-nil.
func (r *Room) carriedLight(exclude *conditions.Condition) []float64 {
	var terms []float64
	add := func(c *characters.Character) {
		for _, rec := range c.Conditions.LightSources() {
			if rec == exclude {
				continue
			}
			if v, ok := rec.LightNow(conditions.GetConditionSpec(rec.ConditionId)); ok {
				terms = append(terms, v)
			}
		}
	}
	for _, id := range r.GetMobs(FindHasLight) {
		if m := mobs.GetInstance(id); m != nil {
			add(&m.Character)
		}
	}
	for _, id := range r.GetPlayers(FindHasLight) {
		if u := users.GetByUserId(id); u != nil {
			add(u.Character)
		}
	}
	return terms
}

// mutatorSkyFilter multiplies the skylight fraction of every active mutator
// that declares one; 1 means nothing filters the sky. Outdoor-only mutators
// never reach an indoor biome (ActiveMutators skips them), so weather stays
// out of roofed rooms.
func (r *Room) mutatorSkyFilter() float64 {
	filter := 1.0
	for mut := range r.ActiveMutators {
		if spec := mut.GetSpec(); spec != nil && spec.SkyLight != nil {
			filter *= *spec.SkyLight
		}
	}
	return filter
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
