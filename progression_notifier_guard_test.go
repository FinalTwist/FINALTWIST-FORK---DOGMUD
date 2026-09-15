package main

import (
	"os"
	"strings"
	"testing"
)

// characters.SetProgressionNotifier defaults to nil, which is silent on
// purpose so tests need no wiring. The cost of that default: deleting the
// registration from main.go silences every skill and stat banner in the game
// while every unit test stays green. This guard is the only thing that sees it.
func TestProgressionNotifierRegisteredAtBoot(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go (test must run from the repo root): %v", err)
	}
	const want = "characters.SetProgressionNotifier(hooks.ProgressionNotifyCallback)"
	if !strings.Contains(string(src), want) {
		t.Errorf("main.go does not register the progression notifier (%s); every progression line would be silent", want)
	}
}
