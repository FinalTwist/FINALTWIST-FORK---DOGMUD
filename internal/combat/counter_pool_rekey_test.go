package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// markedCounterPools seeds one counter pool per defence whose every line
// carries the pool's name, so a render can be traced back to the pool that
// produced it.
func markedCounterPools() map[items.DefencePool]*items.DefenseMessageGroup {
	out := map[items.DefencePool]*items.DefenseMessageGroup{}
	for _, d := range combatvocab.Defences() {
		pool := items.CounterPoolFor(d)
		mk := func(band string) items.DefenseOptions {
			messages := func(audience string) items.MessageOptions {
				result := make(items.MessageOptions, 5)
				for i := range result {
					result[i] = items.ItemMessage("pool=" + string(pool) + " " + audience + " " + band +
						" {actee} answers {actor}")
				}
				return result
			}
			return items.DefenseOptions{Together: items.DefenseTogetherMessages{
				ToDefender: messages("defender"),
				ToAttacker: messages("attacker"),
				ToRoom:     messages("room"),
			}}
		}
		out[pool] = &items.DefenseMessageGroup{OptionId: pool, Options: items.DefenseIntensity{
			items.Weak: mk("weak"), items.Normal: mk("normal"), items.Heavy: mk("heavy"),
		}}
	}
	return out
}

// Every reachable (attack, defence) cell of the spec's matrix renders from
// the WINNER's pool, never the attack's. A melee move parried must read the
// parry pool even though on master every melee counter read one pool, and a
// physical spell dodged must read the dodge pool even though on master it
// read counter-quell. Keying by shape again fails the parry and block rows.
func TestExecuteCounter_NarratesFromTheDefenceThatWon(t *testing.T) {
	pinCounterConfig(t, 0.5)
	restore := items.SeedDefenseMessagesForTest(markedCounterPools())
	defer restore()

	single := combatvocab.TargetSingle
	cases := []struct {
		name    string
		shape   combatvocab.Attack
		defence combatvocab.Defence
	}{
		{"melee dodged", combatvocab.Melee(single), combatvocab.DefenceDodge},
		{"melee parried", combatvocab.Melee(single), combatvocab.DefenceParry},
		{"melee blocked", combatvocab.Melee(single), combatvocab.DefenceBlock},
		{"shot dodged", combatvocab.Ranged(single), combatvocab.DefenceDodge},
		{"shot blocked", combatvocab.Ranged(single), combatvocab.DefenceBlock},
		{"physical spell dodged", combatvocab.Spell(combatvocab.DamagePhysical, single), combatvocab.DefenceDodge},
		{"physical spell blocked", combatvocab.Spell(combatvocab.DamagePhysical, single), combatvocab.DefenceBlock},
		{"mental spell quelled", combatvocab.Spell(combatvocab.DamageMental, single), combatvocab.DefenceQuell},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counterer, countered := counterTestPair()
			calls := 0
			restoreRunner := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
			defer restoreRunner()

			res := ExecuteCounter(counterer, countered, tc.shape, tc.defence, true)
			if !res.Countered {
				t.Fatalf("%s: the counter did not fire", tc.name)
			}
			if res.Defence != tc.defence {
				t.Errorf("%s: CounterResult.Defence = %q, want %q", tc.name, res.Defence, tc.defence)
			}
			want := "pool=" + string(items.CounterPoolFor(tc.defence))
			for role, msg := range map[string]string{
				"counterer": res.DefenderMsg, "countered": res.AttackerMsg, "room": res.RoomMsg,
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("%s: %s line %q did not come from %s", tc.name, role, msg, want)
				}
			}
		})
	}
}
