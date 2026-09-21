// Package movenarration holds the shipped wording for special-move narration:
// the player-side verbs (kick, bash, trip, ...) and their mob twins.
//
// Go decides WHICH event fires and names it by key. Go holds no wording. The
// three audiences of one event are variant-paired by index, which is why
// Validate refuses pools of unequal length: variant N of each role describes
// the same moment.
package movenarration

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/pkg/errors"
)

// TokenDamage is this store's one event token. Damage always arrives
// pre-formatted as prose from combat.GetDamageDescription; there is no numeric
// verb anywhere in the special-move surface.
const TokenDamage = "{damage}"

// EventKey names one outcome branch of one verb, such as "standard_hit".
//
// Where a verb has an orthogonal variant axis (kick resolves as stomp, knee or
// standard; trip as tailsweep or trip), the key is <variant>_<outcome>, because
// an outcome key alone would make three different moves collide on one pool.
type EventKey string

// EventMessages is one event as each audience is told it. The yaml keys are the
// arc's canonical role keys.
type EventMessages struct {
	Actor          []string `yaml:"actor"`
	Actee          []string `yaml:"actee"`
	Observer       []string `yaml:"observer"`
	RemoteObserver []string `yaml:"remote_observer"`
}

// MoveNarrationGroup is one verb's file.
type MoveNarrationGroup struct {
	MoveId string                      `yaml:"moveid"`
	Events map[EventKey]*EventMessages `yaml:"events"`
}

func (g *MoveNarrationGroup) Id() string { return g.MoveId }

func (g *MoveNarrationGroup) Filepath() string {
	return fmt.Sprintf("%s.yaml", g.MoveId)
}

// Validate fails the boot on any malformed event. This store is EVENT tier
// under the arc's two-tier loader policy: silence here means a move lands with
// no narration at all, so it must never be survivable.
func (g *MoveNarrationGroup) Validate() error {
	if g.MoveId == "" {
		return errors.New("moveid is empty")
	}
	if len(g.Events) == 0 {
		return errors.Errorf("move %q declares no events", g.MoveId)
	}
	for key, ev := range g.Events {
		if ev == nil {
			return errors.Errorf("move %q event %q is empty", g.MoveId, key)
		}
		if err := narration.ValidateVariants(ev.variants(), 1); err != nil {
			return errors.Wrapf(err, "move %q event %q", g.MoveId, key)
		}
	}
	return nil
}

func (e *EventMessages) variants() narration.Variants {
	return narration.Variants{
		Actor:         e.Actor,
		Actee:         e.Actee,
		Observer:      e.Observer,
		ActeeObserver: e.RemoteObserver,
	}
}

// Variants returns the event's pools, or ok=false if the verb does not declare
// that event.
func (g *MoveNarrationGroup) Variants(key EventKey) (narration.Variants, bool) {
	ev, ok := g.Events[key]
	if !ok || ev == nil {
		return narration.Variants{}, false
	}
	return ev.variants(), true
}

var loadedMoves map[string]*MoveNarrationGroup

// LoadMoveNarrationFiles loads the store at boot. It panics on any failure,
// matching combat.LoadTauntMessageFiles: this is event narration, not ambient.
func LoadMoveNarrationFiles() {
	dir := string(configs.GetFilePathsConfig().DataFiles) + `/narration/special-moves`
	loaded, err := fileloader.LoadAllFlatFiles[string, *MoveNarrationGroup](dir)
	if err != nil {
		panic(errors.Wrap(err, "loading special-move narration"))
	}
	loadedMoves = loaded
}

// GetMove returns one verb's group, or nil if the store is unloaded or the verb
// is absent.
func GetMove(moveId string) *MoveNarrationGroup {
	if loadedMoves == nil {
		return nil
	}
	return loadedMoves[moveId]
}
