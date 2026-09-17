package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// One door for the player-side condition line: authored text first, a generic line
// underneath, silence for a secret condition. Forty-six dogmud conditions had neither a
// start nor an end line before slice C.
func TestConditionNotices(t *testing.T) {
	authored := &ConditionSpec{ConditionId: 1, Name: "Venom", StartUserText: "Venom burns.", EndUserText: "The venom subsides."}
	assert.Equal(t, "Venom burns.", authored.StartUserNotice())
	assert.Equal(t, "The venom subsides.", authored.EndUserNotice())

	silent := &ConditionSpec{ConditionId: 2, Name: "Warrior's Brew"}
	assert.Equal(t, "Warrior's Brew takes effect.", silent.StartUserNotice())
	assert.Equal(t, "Warrior's Brew has expired.", silent.EndUserNotice())

	secret := &ConditionSpec{ConditionId: 3, Name: "Respawn Grace", Secret: true, StartUserText: "never shown"}
	assert.Equal(t, "", secret.StartUserNotice(), "a secret condition says nothing even with authored text")
	assert.Equal(t, "", secret.EndUserNotice())

	nameless := &ConditionSpec{ConditionId: 4}
	assert.Equal(t, "", nameless.StartUserNotice(), "no name, no generic line; the guard catches this")
	assert.Equal(t, "", nameless.EndUserNotice())

	namelessAuthored := &ConditionSpec{ConditionId: 5, StartUserText: "Still spoken.", EndUserText: "Still ended."}
	assert.Equal(t, "Still spoken.", namelessAuthored.StartUserNotice(), "authored text does not need a name")
	assert.Equal(t, "Still ended.", namelessAuthored.EndUserNotice())
}

func TestSilentNoticeConditionsListsOnlyNonSecretConditionsRelyingOnTheFallback(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		10: {ConditionId: 10, Name: "Authored", StartUserText: "a", EndUserText: "b"},
		11: {ConditionId: 11, Name: "Half", StartUserText: "a"},
		12: {ConditionId: 12, Name: "Bare"},
		13: {ConditionId: 13, Name: "Hidden", Secret: true},
	})
	defer restore()
	assert.Equal(t, []string{"11 Half (end)", "12 Bare (start, end)"}, SilentNoticeConditions(), "sorted by id")
}

// A silent-start condition leaves the start to whatever applies it. Warcry and
// rally bypass events.Condition entirely (Character.AddCondition); the bloom detox
// drink does reach Condition_ApplyConditions on the unscaled path, and the flag keeps
// the drink's own purge narration from being doubled. Either way the
// resolver must say nothing at start
// even when start_actee is (wrongly) authored, and the listing must not
// flag the missing start as a problem.
func TestSilentStartConditionHasNoStartNotice(t *testing.T) {
	noText := &ConditionSpec{ConditionId: 79, Name: "Warcry", EndUserText: "fades", Flags: []Flag{SilentStart}}
	assert.Equal(t, "", noText.StartUserNotice())
	assert.Equal(t, "fades", noText.EndUserNotice())

	withText := &ConditionSpec{ConditionId: 80, Name: "Rally", StartUserText: "never shown", EndUserText: "drains", Flags: []Flag{SilentStart}}
	assert.Equal(t, "", withText.StartUserNotice(), "silent-start wins even over authored start text")
}

func TestSilentStartConditionNotListedForMissingStart(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		20: {ConditionId: 20, Name: "Warcry", EndUserText: "fades", Flags: []Flag{SilentStart}},
	})
	defer restore()
	assert.Equal(t, []string{}, SilentNoticeConditions(), "a silent-start condition with an authored end is not silent by accident")
}

// A hidden condition must never announce its end: if you can't know who spotted
// you, you can't know you've been spotted. The flag wins even over authored
// end text, and the listing must not flag the missing end as a problem.
func TestHiddenConditionHasNoEndNotice(t *testing.T) {
	noText := &ConditionSpec{ConditionId: 9, Name: "Hidden", StartUserText: "sneaky", Flags: []Flag{Hidden}}
	assert.Equal(t, "sneaky", noText.StartUserNotice())
	assert.Equal(t, "", noText.EndUserNotice())

	withText := &ConditionSpec{ConditionId: 31, Name: "Empathic Shroud", StartUserText: "shrouded", EndUserText: "never shown", Flags: []Flag{Hidden}}
	assert.Equal(t, "", withText.EndUserNotice(), "hidden wins even over authored end text")
}

func TestHiddenConditionNotListedForMissingEnd(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		21: {ConditionId: 21, Name: "Hidden", StartUserText: "sneaky", Flags: []Flag{Hidden}},
	})
	defer restore()
	assert.Equal(t, []string{}, SilentNoticeConditions(), "a hidden condition with no end text is silent by design, not by accident")
}

// A quiet condition (prone recovery, the grapple exposure) is reapplied every
// round it persists, so it never emits a start or end line by design, not by
// missing authored text. The listing must not flag either as a problem.
func TestQuietConditionNotListedForMissingNotices(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		22: {ConditionId: 22, Name: "Off Balance", Flags: []Flag{Quiet}},
	})
	defer restore()
	assert.Equal(t, []string{}, SilentNoticeConditions(), "a quiet condition with no authored text is silent by design, not by accident")
}
