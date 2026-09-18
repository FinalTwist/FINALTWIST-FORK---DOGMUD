package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The five defence pools are keyed by the defence's own name, so the YAML
// files under defense-messages/ (dodge.yaml ...) do not move.
func TestDefencePoolForIsTheDefenceName(t *testing.T) {
	for _, d := range combatvocab.Defences() {
		if got := DefencePoolFor(d); string(got) != string(d) {
			t.Errorf("DefencePoolFor(%s) = %q", d, got)
		}
	}
	if got := DefencePoolFor(combatvocab.DefenceNone); got != "" {
		t.Errorf("DefencePoolFor(none) = %q, want empty", got)
	}
}

func TestCounterPoolNamesAreTheShippedFileNames(t *testing.T) {
	want := map[DefencePool]bool{"counter-melee": true, "counter-ranged": true, "counter-quell": true, "counter-defy": true}
	for _, p := range []DefencePool{CounterPoolMelee, CounterPoolRanged, CounterPoolQuell, CounterPoolDefy} {
		if !want[p] {
			t.Errorf("counter pool %q is not a shipped file name", p)
		}
	}
}
