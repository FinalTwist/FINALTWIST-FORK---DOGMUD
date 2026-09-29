package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// Nobody may pickpocket a companion, the thief's own included: a charmed
// one (the predicate mobs.CheckPlayerHarm refuses first) or one bonded to
// the AI companion (owner ruling 2026-09-29). Nothing is rolled, taken or
// trained.
func TestSteal_RefusesAnyCompanion(t *testing.T) {
	actor := newStealPlayerActor(200, 8)
	cases := map[string]func(m *mobs.Mob){
		`someone's charmed companion`:       func(m *mobs.Mob) { m.Character.Charmed = characters.NewCharm(4242, 10, ``) },
		`the thief's own charmed companion`: func(m *mobs.Mob) { m.Character.Charmed = characters.NewCharm(actor.GetUserId(), 10, ``) },
		`an AI companion`: func(m *mobs.Mob) {
			companionai.SetBondedCheck(func(id int) bool { return id == testMobInstId })
		},
	}
	for name, setup := range cases {
		target := newStealTestMob(testMobInstId, 50, 1)
		mobs.SetInstanceForTest(testMobInstId, target)
		setup(target)
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))
		awards := len(actor.awards)

		result := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId})

		companionai.SetBondedCheck(nil)
		mobs.SetInstanceForTest(testMobInstId, nil)
		if result.Reason != "companion" || result.Succeeded || result.Pending || target.Character.Gold != 50 || len(actor.awards) != awards {
			t.Errorf("%v: refused, nothing taken or trained: %+v gold=%d", name, result, target.Character.Gold)
		}
	}
}
