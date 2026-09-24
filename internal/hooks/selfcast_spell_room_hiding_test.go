package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File: selfcast_spell_room_hiding_test.go
//
// M4d PR 3 Task 3: applyPlayerEffect's four self-cast branches (purge, heal,
// condition, shield) each pair a caster line with a room line that names the
// caster via target.Character.Name (target == user for a self-cast, per PR
// 1's comment at spell_resolution.go, the map for this task). All four room
// lines wrap that name in an <ansi fg="username"> identity tag, hardcoded
// directly in the Sprintf format string, so today's shipped content is
// incidentally safe: RenderForRecipient's anonymize stage (pipeline.go) strips
// ANY tag-wrapped identity name for a SightShapes reader, regardless of which
// delivery call sends it. See internal/messaging/pipeline.go and
// internal/messaging/anonymize.go.
//
// Because the tag is baked into the Go source rather than caller-supplied
// content, there is no way to drive applyPlayerEffect itself into emitting a
// bare name the way internal/combat/wait_remote_room_hiding_test.go (a
// hand-authored AttackOptions message) and
// internal/questengine/narrate_room_hiding_test.go (a hand-authored
// narration.Variants) could for their own call sites. This test instead
// proves the MECHANISM directly, on the shield self-cast branch's exact room
// line with its ansi tag stripped off (spell_resolution.go's shield case:
// `A shimmering barrier surrounds <ansi fg="username">%s</ansi>.`), the same
// way those two tests proved theirs:
//
//   - sendVisualRoomText, what all four self-cast branches called before this
//     task, passes no names to Room.SendTextVisual, so a bare name in that
//     position leaks straight through to a shapes-only observer.
//   - messaging.SendTrio, what the four self-cast branches call now, passes
//     the caster's name, so HideNames catches a bare name too.
//
// Aliceia (user 1) self-casts; Bobrick (user 2) is the shapes-only bystander.

const selfcastHidingBareRoomLine = `A shimmering barrier surrounds Aliceia.`

func TestSelfCastSpellRoomLine_SendTrioHidesABareNameSendVisualRoomTextDoesNot(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(heatEyesConditionId, true))
	drainPlain(1)
	drainPlain(2)

	// Pre-fix shape: sendVisualRoomText, the call every self-cast branch used
	// before this task. It passes no names, so only the tag-based Anonymize
	// stage runs, and a bare name leaks straight through.
	sendVisualRoomText(rooms.LoadRoom(1), messaging.CategorySpellVital, selfcastHidingBareRoomLine, 1)
	leaked := drainPlain(2)
	require.Equal(t, 1, countContaining(leaked, "Aliceia"),
		"sendVisualRoomText should leak a bare name to a shapes-only observer (the pre-fix call shape): got %v", leaked)

	// Post-fix shape: messaging.SendTrio, what the migrated self-cast branches
	// call now. It passes the caster's name, so HideNames runs too.
	messaging.SendTrio(messaging.Trio{
		Observer: messaging.Say(messaging.CategorySpellVital, selfcastHidingBareRoomLine),
	}, messaging.Audience{
		Actor:     users.GetByUserId(1),
		ActorId:   1,
		ActorName: "Aliceia",
		ActeeName: messaging.NoName,
		Room:      rooms.LoadRoom(1),
	})
	hidden := drainPlain(2)
	require.Equal(t, 0, countContaining(hidden, "Aliceia"),
		"messaging.SendTrio must hide the caster's bare name from a shapes-only observer: got %v", hidden)
	require.Equal(t, 1, countContaining(hidden, "A shimmering barrier surrounds a figure."))
}

// TestSelfCastShield_RealBranch_ShapesOnlyThirdPartyReadsAFigure is a
// regression guard, not a red test (today's tag-wrapped content is
// incidentally safe either way, per the file comment above): it drives the
// real shield self-cast branch through applyPlayerEffect with a shapes-only
// third party in the room, and pins that the migration to messaging.SendTrio
// (spell_resolution.go's shield case) kept the caster exclusion, the category
// and the delivered wording all intact.
func TestSelfCastShield_RealBranch_ShapesOnlyThirdPartyReadsAFigure(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)

	caster := users.GetByUserId(1)
	target := users.GetByUserId(2)
	watcher := users.NewTestUser(3, "cara", "Carrow", 1003)
	watcher.Character.RoomId = 1
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: caster, 2: target, 3: watcher})
	defer restoreUsers()
	room := rooms.LoadRoom(1)
	room.AddPlayer(3)
	require.True(t, watcher.Character.Conditions.AddCondition(heatEyesConditionId, true))
	drainPlain(1)
	drainPlain(3)

	spell := &spells.SpellData{SpellId: "ward", Name: "Ward", EffectType: "shield"}
	applyPlayerEffect(caster, caster, room, spell, 100, spellContestAttackWin())

	casterLines, watcherLines := drainPlain(1), drainPlain(3)
	assert.Equal(t, 1, countContaining(casterLines, "A shimmering magical barrier forms around you, bolstering your defenses."))
	assert.Equal(t, 1, countContaining(watcherLines, "A shimmering barrier surrounds a figure."))
	assert.Equal(t, 0, countContaining(watcherLines, "Aliceia"))
}
