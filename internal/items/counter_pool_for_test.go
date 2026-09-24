package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The counter earned by a defensive crit is narrated by the DEFENCE that won
// it, so its pool is the defence's name under a counter- prefix. None maps
// to the empty pool, which RenderDefenseMessage answers with an empty triad
// and combat then replaces with its generic fallback: never silent.
func TestCounterPoolForIsCounterDashTheDefence(t *testing.T) {
	for _, d := range combatvocab.Defences() {
		if got, want := CounterPoolFor(d), DefencePool("counter-"+string(d)); got != want {
			t.Errorf("CounterPoolFor(%s) = %q, want %q", d, got, want)
		}
	}
	if got := CounterPoolFor(combatvocab.DefenceNone); got != "" {
		t.Errorf("CounterPoolFor(none) = %q, want empty", got)
	}
}
