package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestSetTips_HintsIsAnAliasForTheSameSetting(t *testing.T) {
	u := users.NewTestUser(8801, "tipper", "Tipper", 18801)

	if _, err := Set("tips", u, nil, events.EventFlag(0)); err != nil {
		t.Fatal(err)
	}
	if got, ok := u.GetConfigOption("tips").(bool); !ok || got {
		t.Fatalf("after `set tips` from the default (on), tips = %v, want false", u.GetConfigOption("tips"))
	}

	if _, err := Set("hints", u, nil, events.EventFlag(0)); err != nil {
		t.Fatal(err)
	}
	if got, ok := u.GetConfigOption("tips").(bool); !ok || !got {
		t.Fatalf("after `set hints`, tips = %v, want true", u.GetConfigOption("tips"))
	}
	if v := u.GetConfigOption("hints"); v != nil {
		t.Errorf("`set hints` wrote a hints option (%v); the alias must write tips", v)
	}
	events.DrainQueuedMessagesForTest(8801)
}
