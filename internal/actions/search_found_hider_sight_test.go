package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// Lighting plan 5c playtest finding 1, the search sibling. A searcher who
// spots a hider in a room they make out only as shapes must not read the
// hider's name: the found list goes through the same who template as the
// room roster, so it follows the same rule.
func TestFoundHiderEntry_FollowsTheSearchersSight(t *testing.T) {
	for _, kind := range []string{"username", "mob"} {
		full := foundHiderEntry("Kesh", kind, messaging.SightFull)
		if !strings.Contains(full, "Kesh") || !strings.Contains(full, "hiding") {
			t.Errorf("%s at full sight = %q, want the name and the hiding mark", kind, full)
		}
		for _, d := range []messaging.SightDecision{messaging.SightShapes, messaging.SightNone} {
			got := foundHiderEntry("Kesh", kind, d)
			if got != messaging.UnseenFigure(d) {
				t.Errorf("%s at sight %v = %q, want %q", kind, d, got, messaging.UnseenFigure(d))
			}
		}
	}
}
