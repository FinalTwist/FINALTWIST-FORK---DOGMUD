package actions

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// fixedLight is a RoomVisibility test double: a room at one light level.
type fixedLight int

func (l fixedLight) LightLevel() int { return int(l) }

// The observer's eyes count in detection; the hider's already did.
func TestDetectionScoreTakesTheObserversEyes(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	o := characters.New()
	o.Stats.Perception.ValueAdj = 100
	comfortable := CalcDetectionScore(o, fixedLight(60))
	dazzled := CalcDetectionScore(o, fixedLight(90))
	if math.Abs(dazzled/comfortable-0.88) > 1e-9 {
		t.Errorf("dazzled/comfortable = %v, want 0.88", dazzled/comfortable)
	}
}
