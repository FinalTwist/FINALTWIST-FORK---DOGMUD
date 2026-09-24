package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestProgressionNotifyCallback_SkillProgressCategory(t *testing.T) {
	u := users.NewTestUser(901, "prognotify", "Prognotify", 1901)
	restore := users.SeedUsersForTest(map[int]*users.UserRecord{901: u})
	defer restore()
	events.DrainQueuedMessagesForTest(901)

	ProgressionNotifyCallback(901, "BANNER")

	got := events.DrainQueuedMessagesForTest(901)
	want := `<ansi fg="skill-progress">BANNER</ansi>` + "\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("queued %q, want exactly [%q]", got, want)
	}
}

func TestProgressionNotifyCallback_UnknownUserSendsNothing(t *testing.T) {
	restore := users.SeedUsersForTest(map[int]*users.UserRecord{})
	defer restore()
	events.DrainQueuedMessagesForTest(902)

	ProgressionNotifyCallback(902, "BANNER")

	if got := events.DrainQueuedMessagesForTest(902); len(got) != 0 {
		t.Fatalf("queued %q for a user who is not online, want nothing", got)
	}
}
