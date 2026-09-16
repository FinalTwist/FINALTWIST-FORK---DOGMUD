package items

import (
	"fmt"
	"reflect"
	"testing"
)

// firstPickerForTest always selects index 0.
func firstPickerForTest(n int) int { return 0 }

func TestPoolForUnionsTiersCumulatively(t *testing.T) {
	stm := SkillTieredMessages{
		Beginner: MessageOptions{"b1", "b2"},
		Expert:   MessageOptions{"e1"},
		Master:   MessageOptions{"m1"},
	}
	cases := []struct {
		skill int
		want  []string
	}{
		{10, []string{"b1", "b2"}},
		{33, []string{"b1", "b2"}},
		{34, []string{"b1", "b2", "e1"}},
		{66, []string{"b1", "b2", "e1"}},
		{67, []string{"b1", "b2", "e1", "m1"}},
		{100, []string{"b1", "b2", "e1", "m1"}},
	}
	for _, c := range cases {
		got := stm.PoolFor(c.skill)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("PoolFor(%d) = %v, want %v", c.skill, got, c.want)
		}
	}
}

func TestPoolForEmptyIsNil(t *testing.T) {
	var stm SkillTieredMessages
	if got := stm.PoolFor(100); got != nil {
		t.Errorf("PoolFor on an empty store = %v, want nil so the core treats the role as absent", got)
	}
}

// TestTogetherRenderCoordinatesOneIndexAcrossRoles is the point of the whole
// migration: variant N of each role describes the same moment, so all roles
// must come from ONE index. Picking per role is the defect that shipped twice
// before (melee defence PR #112, taunt PR #115).
func TestTogetherRenderCoordinatesOneIndexAcrossRoles(t *testing.T) {
	m := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a0", "a1", "a2"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d0", "d1", "d2"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r0", "r1", "r2"}},
	}
	for idx := 0; idx < 3; idx++ {
		pick := func(n int) int { return idx }
		roles := m.Render(10, nil, pick)
		wantA := fmt.Sprintf("a%d", idx)
		wantD := fmt.Sprintf("d%d", idx)
		wantR := fmt.Sprintf("r%d", idx)
		if roles.Actor != wantA || roles.Actee != wantD || roles.Observer != wantR {
			t.Errorf("index %d: got (%q,%q,%q), want (%q,%q,%q); all roles must come from ONE index",
				idx, roles.Actor, roles.Actee, roles.Observer, wantA, wantD, wantR)
		}
		if roles.ActeeObserver != "" {
			t.Errorf("index %d: together must leave ActeeObserver empty, got %q", idx, roles.ActeeObserver)
		}
	}
}

// TestSeparateRenderMapsFourRoles pins the role mapping for the ranged case.
// Swapping Actor and Actee here would invert every combat message in the game,
// and three same-shaped pools as adjacent struct fields is exactly how that
// mistake gets made.
func TestSeparateRenderMapsFourRoles(t *testing.T) {
	m := SeparateMessages{
		ToAttacker:     SkillTieredMessages{Beginner: MessageOptions{"atk"}},
		ToDefender:     SkillTieredMessages{Beginner: MessageOptions{"def"}},
		ToAttackerRoom: SkillTieredMessages{Beginner: MessageOptions{"atkroom"}},
		ToDefenderRoom: SkillTieredMessages{Beginner: MessageOptions{"defroom"}},
	}
	roles := m.Render(10, nil, firstPickerForTest)
	if roles.Actor != "atk" {
		t.Errorf("Actor = %q, want %q (toattacker ACTS, so it is the Actor)", roles.Actor, "atk")
	}
	if roles.Actee != "def" {
		t.Errorf("Actee = %q, want %q (todefender is ACTED UPON)", roles.Actee, "def")
	}
	if roles.Observer != "atkroom" {
		t.Errorf("Observer = %q, want %q (observers where the attacker is)", roles.Observer, "atkroom")
	}
	if roles.ActeeObserver != "defroom" {
		t.Errorf("ActeeObserver = %q, want %q (observers where the defender is)", roles.ActeeObserver, "defroom")
	}
}

func TestRenderSubstitutesTokens(t *testing.T) {
	m := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"You hit {target}!"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"{source} hits you!"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"{source} hits {target}!"}},
	}
	roles := m.Render(10, map[TokenName]string{
		TokenSource: "Sable",
		TokenTarget: "the goblin",
	}, firstPickerForTest)
	if roles.Actor != "You hit the goblin!" {
		t.Errorf("Actor = %q", roles.Actor)
	}
	if roles.Actee != "Sable hits you!" {
		t.Errorf("Actee = %q", roles.Actee)
	}
	if roles.Observer != "Sable hits the goblin!" {
		t.Errorf("Observer = %q", roles.Observer)
	}
}

// TestRenderTakesExactlyOneDrawPerMessage pins the draw count. It was three
// draws for together and four for separate, one per audience. The count is
// load-bearing twice over: it IS the coordination guarantee, and it is what
// shifts the global random stream when this lands.
func TestRenderTakesExactlyOneDrawPerMessage(t *testing.T) {
	draws := 0
	counting := func(n int) int { draws++; return 0 }

	together := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a0", "a1"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d0", "d1"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r0", "r1"}},
	}
	together.Render(10, nil, counting)
	if draws != 1 {
		t.Errorf("together took %d draws, want exactly 1; a draw per role is the defect this migration removes", draws)
	}

	draws = 0
	separate := SeparateMessages{
		ToAttacker:     SkillTieredMessages{Beginner: MessageOptions{"a0", "a1"}},
		ToDefender:     SkillTieredMessages{Beginner: MessageOptions{"d0", "d1"}},
		ToAttackerRoom: SkillTieredMessages{Beginner: MessageOptions{"ar0", "ar1"}},
		ToDefenderRoom: SkillTieredMessages{Beginner: MessageOptions{"dr0", "dr1"}},
	}
	separate.Render(10, nil, counting)
	if draws != 1 {
		t.Errorf("separate took %d draws, want exactly 1", draws)
	}
}

// TestRenderUnequalPoolsRenderNothing documents the contract the content pad
// exists to satisfy: the core refuses to invent a pairing.
func TestRenderUnequalPoolsRenderNothing(t *testing.T) {
	m := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a0", "a1", "a2"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d0", "d1"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r0", "r1", "r2"}},
	}
	roles := m.Render(10, nil, firstPickerForTest)
	if roles.Actor != "" || roles.Actee != "" || roles.Observer != "" {
		t.Errorf("unequal pools must render nothing, got %+v", roles)
	}
}
