package aicompanion

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	bramLooks    = `a tall man in a patched crimson cloak`
	bramOverhead = `the cellar key is under the third stone`
)

// privacyMind fills her mind with what another player showed and said that
// was not for her or her owner: an overheard line, and a look at Bram.
func privacyMind(c *controller) {
	now := time.Now().Unix()
	c.mind.addLine(Line{Speaker: `Bram`, Kind: `said`, Text: bramOverhead, Unix: now}, 50)
	c.mind.addLine(Line{Speaker: `Bram`, Kind: `said`, ToMe: true, Text: `Mara, well met`, Unix: now}, 50)
	c.mind.addLine(Line{Speaker: `Corvin`, Kind: `said`, Text: `onward, then`, Unix: now}, 50)
	c.mind.addLine(Line{Kind: `event`, Text: `You looked at Bram: ` + bramLooks + ` They look well.`,
		Plain: `You looked at Bram: They look well.`, Unix: now}, 50)
}

func TestRelaySafeLines(t *testing.T) {
	lines := []Line{
		{Speaker: `Bram`, Kind: `said`, Text: bramOverhead},
		{Speaker: `Bram`, Kind: `emoted`, Text: `winks at the barmaid`},
		{Speaker: `Bram`, Kind: `said`, ToMe: true, Text: `Mara, well met`},
		{Speaker: `Bram`, Kind: `asked`, ToMe: true, Text: `where to?`},
		{Speaker: `Corvin`, Kind: `said`, Text: `onward, then`},
		{Speaker: `Mara`, Kind: `said`, Text: `as you like`},
		{Kind: `event`, Text: `You looked at Bram: ` + bramLooks, Plain: `You looked at Bram: They look well.`},
		{Kind: `event`, Text: `Bram attacked you.`},
	}
	got := relaySafeLines(lines, `Corvin`, `Mara`)
	var texts []string
	for _, l := range got {
		texts = append(texts, l.Text)
	}
	all := strings.Join(texts, ` | `)
	for _, gone := range []string{bramOverhead, `winks at the barmaid`, bramLooks} {
		if strings.Contains(all, gone) {
			t.Errorf("a relay prompt must not carry %q: %s", gone, all)
		}
	}
	for _, kept := range []string{`Mara, well met`, `where to?`, `onward, then`, `as you like`,
		`You looked at Bram: They look well.`, `Bram attacked you.`} {
		if !strings.Contains(all, kept) {
			t.Errorf("names, deeds and what was said to her stay: missing %q in %s", kept, all)
		}
	}
	if lines[6].Text != `You looked at Bram: `+bramLooks {
		t.Fatal("the mind's own lines are not changed")
	}
}

// relayDispatchModule is a module whose owner Corvin has his own key up,
// with a fake browser that captures what is sent to it.
func relayDispatchModule(t *testing.T, relay bool) (*AICompanionModule, *controller, *fakeRelay) {
	t.Helper()
	_, other, _, her := harmWorld(t, `off`)
	other.Character.Description = bramLooks + `.`
	srv, _ := countingServer(t)
	m, c := senderModule(srv.URL, true)
	c.instanceId = her.InstanceId
	m.ctrls = map[int]*controller{1: c}
	m.minds = map[string]*Mind{mindIdentifier(c.mind.OwnerUserId, c.mind.MobId): c.mind}
	f := newFakeRelay()
	if relay {
		withWebDomain(t, `example.org`)
		m.cfg.PlayerKeys, m.cfg.RelayOrigin = true, `https://keys.example.org`
		m.cfg.RelayTimeoutSeconds = 5
		m.relays = newRelayTable()
		m.relays.ready(1, `player-model`)
		m.relayCalls = newPendingRelays()
		m.relaySend = f.send
	}
	return m, c, f
}

// On her owner's own key the decision prompt, which the owner's browser
// carries, leaves out what Bram looks like and what he said to nobody in
// particular; his name and his words to her stay. On the server's key
// nothing changes.
func TestRelayDecisionPromptLeavesOutOtherPlayersPrivateDetail(t *testing.T) {
	for _, relay := range []bool{false, true} {
		m, c, f := relayDispatchModule(t, relay)
		util.LockMud()
		privacyMind(c)
		c.push(stimulus{Kind: `looked`, Text: `You looked at Bram: ` + bramLooks + ` They look well.`,
			Plain: `You looked at Bram: They look well.`, Chain: 1})
		m.dispatch(c)
		sent := ``
		for _, msg := range c.lastPrompt {
			sent += msg.Content
		}
		util.UnlockMud()
		if relay {
			sent = string(f.next(t).Body)
		}
		util.LockMud()
		c.cancelInFlight()
		util.UnlockMud()
		// The cancelled call still settles and applies under the lock; the
		// next pass rebuilds the world, so it must be finished first.
		m.decisions.Wait()

		if sent == `` {
			t.Fatalf("relay=%v: fixture: a prompt was sent", relay)
		}
		hasLooks, hasOverheard := strings.Contains(sent, bramLooks), strings.Contains(sent, bramOverhead)
		if relay && (hasLooks || hasOverheard) {
			t.Fatalf("a prompt through the owner's browser carries Bram's looks (%v) or overheard words (%v)", hasLooks, hasOverheard)
		}
		if !relay && (!hasLooks || !hasOverheard) {
			t.Fatalf("control: on the server's key the prompt is unchanged: looks=%v overheard=%v", hasLooks, hasOverheard)
		}
		if !strings.Contains(sent, `Bram`) || !strings.Contains(sent, `Mara, well met`) {
			t.Fatalf("relay=%v: his name and his words to her stay", relay)
		}
	}
}

// The reflection and a core memory, sent through the owner's browser, are
// held to the same rule.
func TestRelayBackgroundPromptsLeaveOutOtherPlayersPrivateDetail(t *testing.T) {
	m, c, f := relayDispatchModule(t, true)
	util.LockMud()
	privacyMind(c)
	in := reflectionInput{Profile: c.profile, OwnerName: `Corvin`, Lines: append([]Line(nil), c.mind.RecentLines...)}
	m.launchReflection(&deferredReflection{mind: c.mind, in: in})
	util.UnlockMud()
	reflection := string(f.next(t).Body)

	util.LockMud()
	m.recordCore(c, `Corvin`, romanceCourting, true)
	util.UnlockMud()
	core := string(f.next(t).Body)

	for name, body := range map[string]string{`reflection`: reflection, `core memory`: core} {
		if strings.Contains(body, bramLooks) || strings.Contains(body, bramOverhead) {
			t.Errorf("the %s prompt carries Bram's looks or overheard words", name)
		}
		if !strings.Contains(body, `Mara, well met`) {
			t.Errorf("control: the %s prompt still carries what he said to her", name)
		}
	}
}

// look_closer at another player, answered for a call through the owner's
// browser, tells how they are and what kind, never their description.
func TestRelayLookCloserLeavesOutTheirDescription(t *testing.T) {
	m, c, _ := relayDispatchModule(t, true)
	owner := users.GetByUserId(1)
	her := mobs.GetInstance(c.instanceId)
	room := rooms.LoadRoom(1)
	if owner == nil || her == nil || room == nil {
		t.Fatal("fixture: owner, companion and room")
	}
	sc := harmScene(0, 2)
	for _, relay := range []bool{false, true} {
		util.LockMud()
		got := m.answerTool(c, her, owner, room, sc, `look_closer`, toolArgs{Ref: `t2`}, relay)
		util.UnlockMud()
		if relay == strings.Contains(got, bramLooks) {
			t.Fatalf("relay=%v: look_closer said %q", relay, got)
		}
		if !strings.Contains(got, `Bram`) || !strings.Contains(got, `look`) {
			t.Fatalf("relay=%v: his name and how he is stay: %q", relay, got)
		}
	}
}

// A look at another player keeps a plain form beside the full one, in her
// mind and in the follow-up it earns, for a prompt through her owner's key.
func TestLookAtAPlayerKeepsAPlainForm(t *testing.T) {
	m, c, _ := relayDispatchModule(t, true)
	her := mobs.GetInstance(c.instanceId)
	room := rooms.LoadRoom(1)
	util.LockMud()
	out := m.lookAt(c, her, room, &thing{Kind: `player`, Name: `Bram`, UserId: 2, Key: `player:2`}, 0, false)
	util.UnlockMud()
	last := c.mind.RecentLines[len(c.mind.RecentLines)-1]
	if !strings.Contains(out.Perceived, bramLooks) || !strings.Contains(last.Text, bramLooks) {
		t.Fatalf("fixture: the full look carries his description: %q", out.Perceived)
	}
	for _, plain := range []string{out.Plain, last.Plain} {
		if plain == `` || strings.Contains(plain, bramLooks) || !strings.Contains(plain, `Bram`) {
			t.Fatalf("the plain form names him and leaves out his looks: %q", plain)
		}
	}
}
