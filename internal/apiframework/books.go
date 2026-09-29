package apiframework

import (
	"sync"
	"time"
)

// Books is the server key's daily budget and circuit breakers together. A
// running server has ONE set, Shared(), which every feature spends from and
// which the package-level functions (Reserve, Settle, Admit, Record and the
// rest) use. A caller may hold a *Books instead (the AI companion does, nil
// meaning Shared), so a test can give each module under test its own
// isolated set: a call a finished test left in flight then settles into
// that test's books, never the next test's.
type Books struct {
	l *ledger
	b *breaker // the provider's, every feature's

	mu         sync.Mutex
	byConsumer map[string]*breaker // each feature's own
}

var shared = &Books{l: budget, b: serverBreaker}

var serverBreaker = &breaker{}

// Shared is the server's one set of books.
func Shared() *Books { return shared }

// NewBooksForTest is a fresh, isolated set: a new day, nothing spent, every
// breaker closed, nothing read from or written to disk.
func NewBooksForTest() *Books {
	return &Books{l: &ledger{now: time.Now, loaded: true}, b: &breaker{}}
}

// SetClockForTest replaces these books' ledger clock.
func (k *Books) SetClockForTest(now func() time.Time) {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.now = now
}
