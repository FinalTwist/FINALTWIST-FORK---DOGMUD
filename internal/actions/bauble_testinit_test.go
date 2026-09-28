package actions

import (
	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Every search test in this package that is not about baubles must not take
// a real bauble roll: a find (1 to 5% per roll by biome) would start a
// background delivery and make the search read as a find, so a test about
// hidden exits could skip or flake. The bauble tests replace this with
// stubBaubleSearch (search_bauble_test.go), which restores it afterwards.
//
// A player's pickpocket holds its outcome back for a pause on a goroutine
// (steal_pocket.go). Here it resolves in line, with no pause, against the
// test's own fake actor, and no bauble is made unless a test says so
// (pickpocket_test.go), so every steal test reads its outcome from
// the StealResult as before.
func init() {
	searchBaubleRoll = func(baubles.FindOpts) (baubles.ValueTier, bool) { return ``, false }
	runPocketAttempt = resolvePocketInLine
	pocketThief = func(p *pocketAttempt) (Actor, bool) { return p.actor, p.actor != nil }
	pocketBaubleRoll = func(*rooms.Room) bool { return false }
}

// resolvePocketInLine names any bauble at once (blocking) and reveals the
// outcome, with no pause.
func resolvePocketInLine(p *pocketAttempt) StealResult {
	if p.newBauble {
		p.name()
	}
	p.claim()
	return p.resolve()
}
