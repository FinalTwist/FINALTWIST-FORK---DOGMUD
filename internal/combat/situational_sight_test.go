package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/contest"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Lighting plan 5b, Task 4: the sight row on the situational table and its
// defence-side twin. Light 60 sits inside a normal observer's comfortable
// band; light 90 is bright 0.6 for normal eyes, which the shipped DazzleCap
// of 0.80 prices at 0.88 (hardcoded here, not recomputed, so a broken
// SightScoreMultiplier cannot agree with itself).

const (
	sightComfortableLight = verdictLight(60)
	sightDazzledLight     = verdictLight(90)
	sightDazzledWant      = 0.88
)

func sightNear(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// pinSituationalSight pins the situational knobs and the ramp caps together;
// the second SetConfigForTest starts from the first one's config.
func pinSituationalSight(t *testing.T) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	pinSituationalKnobs(t)
	pinSightRampCaps(t)
}

func sightShapes() map[string]combatvocab.Attack {
	return map[string]combatvocab.Attack{
		"melee":    combatvocab.Melee(combatvocab.TargetSingle),
		"ranged":   combatvocab.Ranged(combatvocab.TargetSingle),
		"thrown":   combatvocab.Thrown(combatvocab.TargetSingle),
		"spell":    combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle),
		"rhetoric": combatvocab.Rhetoric(combatvocab.TargetSingle),
	}
}

// A dazzled attacker (light 90, normal eyes) pays 0.88 on melee, ranged,
// thrown and spell, and nothing on rhetoric; a comfortable one pays nothing.
func TestSituationalAttackMultSightRow(t *testing.T) {
	pinSituationalSight(t)
	c := newSituationalAttacker(t)
	for name, shape := range sightShapes() {
		comfortable := SituationalAttackMult(c, sightComfortableLight, shape)
		if !sightNear(comfortable, 1.0) {
			t.Errorf("%s: comfortable mult = %v, want 1.0", name, comfortable)
		}
		if nilRoom := SituationalAttackMult(c, nil, shape); !sightNear(nilRoom, 1.0) {
			t.Errorf("%s: nil-room mult = %v, want 1.0", name, nilRoom)
		}
		dazzled := SituationalAttackMult(c, sightDazzledLight, shape)
		want := sightDazzledWant
		if name == "rhetoric" {
			want = 1.0
		}
		if got := dazzled / comfortable; !sightNear(got, want) {
			t.Errorf("%s: dazzled/comfortable = %v, want %v", name, got, want)
		}
	}
}

// The defence side: a defender must see a swing, a shot, a throw or a cast
// coming; a defy needs no eyes. Nil defender and nil room are unity.
func TestSituationalDefenceMultSightRow(t *testing.T) {
	pinSituationalSight(t)
	d := characters.New()
	for name, shape := range sightShapes() {
		want := sightDazzledWant
		if name == "rhetoric" {
			want = 1.0
		}
		if got := SituationalDefenceMult(d, sightDazzledLight, shape); !sightNear(got, want) {
			t.Errorf("%s: dazzled defence mult = %v, want %v", name, got, want)
		}
		if got := SituationalDefenceMult(d, sightComfortableLight, shape); !sightNear(got, 1.0) {
			t.Errorf("%s: comfortable defence mult = %v, want 1.0", name, got)
		}
		if got := SituationalDefenceMult(d, nil, shape); !sightNear(got, 1.0) {
			t.Errorf("%s: nil-room defence mult = %v, want 1.0", name, got)
		}
		if got := SituationalDefenceMult(nil, sightDazzledLight, shape); !sightNear(got, 1.0) {
			t.Errorf("%s: nil-defender defence mult = %v, want 1.0", name, got)
		}
	}
}

// SightRoom turns a typed-nil *rooms.Room into a nil interface, so the sight
// rows read unity instead of panicking on LightLevel.
func TestSightRoomTypedNilIsNilInterface(t *testing.T) {
	var r *rooms.Room
	if got := SightRoom(r); got != nil {
		t.Fatalf("SightRoom(typed nil) = %#v, want a nil interface", got)
	}
	pinSituationalSight(t)
	c := newSituationalAttacker(t)
	if got := SituationalAttackMult(c, SightRoom(r), combatvocab.Melee(combatvocab.TargetSingle)); !sightNear(got, 1.0) {
		t.Errorf("typed-nil room mult = %v, want 1.0", got)
	}
}

// SkillMoveParams.Room reaches the seam: a special move in a dazzling room
// hands the contest defence entries at 0.88 of the comfortable ones.
func TestSkillMoveRoomReachesTheDefenceSightRow(t *testing.T) {
	pinDefenceAdmissionConfig(t)
	pinSightRampCaps(t)

	// One pair for both captures: characters.New() does not promise equal
	// scores across instances, and the ratio must compare like with like.
	atk, def := newDefenceTestCharacter(t), newDefenceTestCharacter(t)
	capture := func(room verdictLight) map[string]float64 {
		got := map[string]float64{}
		runner := func(_ float64, entries []contest.Entry) contest.Result {
			for _, e := range entries {
				got[e.Name] = e.Score
			}
			return contest.Result{Contested: false}
		}
		executeSkillMoveWithRunner(SkillMoveParams{
			Attacker: atk,
			Defender: def,
			Shape:    combatvocab.Melee(combatvocab.TargetSingle),
			Room:     room,
			Attack:   side(148, 52),
		}, runner)
		return got
	}

	clean, dazzled := capture(sightComfortableLight), capture(sightDazzledLight)
	if len(clean) == 0 {
		t.Fatal("fixture guard: the runner saw no defence entries")
	}
	for name, score := range clean {
		if score <= 0 {
			t.Fatalf("fixture guard: %s scored %v at light 60", name, score)
		}
		if got := dazzled[name] / score; !sightNear(got, sightDazzledWant) {
			t.Errorf("%s: dazzled/comfortable = %v, want %v", name, got, sightDazzledWant)
		}
	}
}

// Through the funnel: every defence entry the contest sees at light 90 is
// 0.88 of the same entry at light 60 for a melee shape, and unchanged for a
// rhetoric shape.
func TestResolveChannelAttackDefenceEntriesRideTheSightRow(t *testing.T) {
	pinDefenceAdmissionConfig(t)
	pinSightRampCaps(t)

	// One pair for every capture, as above.
	atk, def := newDefenceTestCharacter(t), newDefenceTestCharacter(t)
	capture := func(room verdictLight, shape combatvocab.Attack) map[string]float64 {
		got := map[string]float64{}
		runner := func(_ float64, entries []contest.Entry) contest.Result {
			for _, e := range entries {
				got[e.Name] = e.Score
			}
			return contest.Result{Contested: false}
		}
		resolveChannelAttackWithRunner(room, shape, side(148, 52), atk, def, runner)
		return got
	}

	for _, tc := range []struct {
		name  string
		shape combatvocab.Attack
		want  float64
	}{
		{"melee", combatvocab.Melee(combatvocab.TargetSingle), sightDazzledWant},
		{"rhetoric", combatvocab.Rhetoric(combatvocab.TargetSingle), 1.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clean := capture(sightComfortableLight, tc.shape)
			dazzled := capture(sightDazzledLight, tc.shape)
			if len(clean) == 0 {
				t.Fatal("fixture guard: the runner saw no defence entries")
			}
			for name, score := range clean {
				if score <= 0 {
					t.Fatalf("fixture guard: %s scored %v at light 60; every ratio would pass", name, score)
				}
				if got := dazzled[name] / score; !sightNear(got, tc.want) {
					t.Errorf("%s: dazzled/comfortable = %v, want %v", name, got, tc.want)
				}
			}
		})
	}
}
