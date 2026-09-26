package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Lighting plan 5b, score-only sites: the actor who needs to see pays
// SightMult on their score. These sites hand-build a score straight into
// combat.RunContest with no recording seam, so each test compares success
// RATES at a comfortable lamp (60, SightMult 1.0) and a dazzling one (90,
// SightMult 0.88) with a 175-vs-125 matchup (~80% comfortable through the
// floored opposed seam; the dazzled 154 falls well below). A 5-point gap at
// 3000 trials is many standard errors wide. Sabotage-checked 2026-09-26: with
// the site's SightMult replaced by 1 both tests fail (rates within 1.5 points).

const sightRampTrials = 3000

// The thief's attack score pays the thief's eyes.
func TestStealPaysTheThiefsEyes(t *testing.T) {
	rate := func(lamp int) float64 {
		target := newStealTestMob(testMobInstId, 50, 125) // victim score 125
		mobs.SetInstanceForTest(testMobInstId, target)
		defer mobs.SetInstanceForTest(testMobInstId, nil)

		won := 0
		for i := 0; i < sightRampTrials; i++ {
			// Rank 2 is the steal floor; Dex tops the score up to 175.
			sw := float64(configs.GetBalanceConfig().SkillWeight)
			actor := newStealPlayerActor(175-int(2*sw), 2)
			actor.room.Lamp = rooms.LampPtr(lamp)
			target.Character.Gold = 50
			if Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId}).Succeeded {
				won++
			}
		}
		return float64(won) / sightRampTrials
	}
	comfortable, dazzled := rate(60), rate(90)
	if dazzled > comfortable-0.05 {
		t.Fatalf("dazzled thief won %.1f%% vs %.1f%% comfortable; the sight ramp "+
			"is not reaching the theft contest", dazzled*100, comfortable*100)
	}
}

// The defuser's score pays the defuser's eyes.
func TestDefusePaysTheDefusersEyes(t *testing.T) {
	rate := func(lamp int) float64 {
		won := 0
		for i := 0; i < sightRampTrials; i++ {
			// Perception 100 + rank 3 x 25 = 175 against lock difficulty 12 x 10.
			actor := newDefuseActor(100, 3)
			actor.room.Lamp = rooms.LampPtr(lamp)
			actor.room.Containers = map[string]rooms.Container{
				defuseTestContainerName: {Lock: gamelock.Lock{
					Difficulty:       12,
					TrapConditionIds: []int{defuseTestTrapConditionId},
				}},
			}
			seedDisarmKit(actor, 9950, 0)
			if Defuse(actor, DefuseOptions{TargetNoun: defuseTestContainerName}).Succeeded {
				won++
			}
		}
		return float64(won) / sightRampTrials
	}
	comfortable, dazzled := rate(60), rate(90)
	if dazzled > comfortable-0.05 {
		t.Fatalf("dazzled defuser won %.1f%% vs %.1f%% comfortable; the sight ramp "+
			"is not reaching the defuse contest", dazzled*100, comfortable*100)
	}
}
