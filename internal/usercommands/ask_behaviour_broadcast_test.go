package usercommands

import (
	"os"
	"regexp"
	"testing"
)

// A question answered by a behaviour tree ends the command where the
// original Ask ended it: the neighbouring rooms hear nothing. The chain was
// extracted from Ask so a bonded companion could reuse it, and this is the
// behaviour that extraction is apt to lose. Both callers must skip their
// exit broadcast when the chain reports it handled the question.
func TestBehaviourTreeAnswerSkipsTheExitBroadcast(t *testing.T) {
	src, err := os.ReadFile(`ask.go`)
	if err != nil {
		t.Fatalf("read ask.go: %v", err)
	}
	text := string(src)

	if !regexp.MustCompile(`func askNpcChain\([^)]*\) \(handled bool\)`).MatchString(text) {
		t.Fatal("askNpcChain must report whether a behaviour tree took the question")
	}

	// Every call of the chain must be inside a condition, and every exit
	// broadcast must sit after one.
	calls := regexp.MustCompile(`(?m)^\s*(if )?askNpcChain\(`).FindAllStringSubmatch(text, -1)
	if len(calls) < 2 {
		t.Fatalf("expected both callers to be present, found %d", len(calls))
	}
	for _, c := range calls {
		if c[1] == `` {
			t.Fatal("a caller uses askNpcChain without checking whether it was handled, so a behaviour-tree answer would leak a line into the adjacent rooms")
		}
	}
}
