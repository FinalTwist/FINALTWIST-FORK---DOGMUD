package mobcommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

var mobSpeechTag = regexp.MustCompile(`<[^>]*>`)

// mobSpeechHeard drains uid's queued lines the way its client prints them:
// minus those the deafen filter drops, tags stripped.
func mobSpeechHeard(uid int) []string {
	u := users.GetByUserId(uid)
	var out []string
	for _, m := range events.DrainQueuedMessageEventsForTest(uid) {
		if m.HiddenFromDeafened(u.Deafened) {
			continue
		}
		out = append(out, strings.TrimSpace(mobSpeechTag.ReplaceAllString(m.Text, "")))
	}
	return out
}

// mobSpeechRoom lights room 1 exactly (no sky, Lamp 60) and blinds Bobrick
// (user 2) with the perception machine; Aliceia (user 1) sees clearly. The
// speaker is mob 100, "Skeleton".
func mobSpeechRoom(t *testing.T) (*mobs.Mob, *rooms.Room) {
	t.Helper()
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(60)
	bob := users.GetByUserId(2)
	bob.Character.Perception = characters.New().Perception
	require.NoError(t, bob.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}))
	require.Equal(t, messaging.SightFull, messaging.ParticipantSight(users.GetByUserId(1).Character, room))
	require.Equal(t, messaging.SightNone, messaging.ParticipantSight(bob.Character, room))
	events.DrainQueuedMessagesForTest(1)
	events.DrainQueuedMessagesForTest(2)
	return mob, room
}

// The defect sendAudioRoomText's lit-room shortcut carried: a blinded
// listener in a lit room was told the speaker's name.
func TestMobSay_BlindedListenerInALitRoomHearsNoName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)

	_, err := Say("hello there", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton says, "hello there"`}, mobSpeechHeard(1))
	require.Equal(t, []string{`Someone says, "hello there"`}, mobSpeechHeard(2))
}

// Ruling 6: NPC speech is authored content, so a deafened player hears it.
func TestMobSay_ReachesADeafenedPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	users.GetByUserId(1).Deafened = true

	_, err := Say("hello there", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton says, "hello there"`}, mobSpeechHeard(1))
}

func TestMobShout_BlindedListenerHearsTheWordsNotTheName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)

	_, err := Shout("intruders!", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton shouts, "intruders!"`}, mobSpeechHeard(1))
	require.Equal(t, []string{`Someone shouts, "intruders!"`}, mobSpeechHeard(2))
}

func TestMobEmote_SeenNotHeard(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	users.GetByUserId(1).Deafened = true

	_, err := Emote("growls.", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{"Skeleton growls."}, mobSpeechHeard(1), "a mob emote reaches the deafened (ruling 6)")
	require.Empty(t, mobSpeechHeard(2), "the blinded see nothing")
}

// The lit-room shortcut named a howling mob to a blinded listener.
func TestMobHowl_FumbleHeardWithoutAName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	mob.Character.SetAggro(1, 0, characters.DefaultAttack)
	original := executeTauntAction
	executeTauntAction = func(actions.Actor) actions.TauntResult {
		return actions.TauntResult{Executed: true, Fumble: true}
	}
	t.Cleanup(func() { executeTauntAction = original })

	_, err := Howl("", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{"Skeleton lets out a pitiful howl that trails off weakly."}, mobSpeechHeard(1))
	require.Equal(t, []string{"Something lets out a pitiful howl that trails off weakly."}, mobSpeechHeard(2))
}

// Ported from audio_room_text_sight_test.go (lighting plan 5c finding 1): a
// night-vision holder in a dark room cannot tell who is speaking.
func TestMobSay_NightVisionInTheDarkHearsNoName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", Name: "Cave", Symbol: ".", SkyLight: rooms.SkyLightPtr(0.0), MovementCost: 1},
	})
	defer restoreBiomes()
	const nightId = 9631
	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		nightId: {ConditionId: nightId, Name: "Test Night Sight", RoundInterval: 1, TriggerCount: 10,
			Flags:   []conditions.Flag{conditions.NightVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 24}}},
	})
	defer restoreConditions()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	for _, lamp := range []int{0, 10, 24} {
		room := rooms.LoadRoom(2)
		require.NotNil(t, room)
		room.Biome = "cave"
		room.Lamp = rooms.LampPtr(lamp)
		u := users.GetByUserId(1)
		rooms.LoadRoom(1).RemovePlayer(1)
		u.Character.RoomId = 2
		room.AddPlayer(1)
		if !u.Character.HasCondition(nightId) {
			require.True(t, u.Character.Conditions.AddCondition(nightId, true))
		}
		require.NotEqual(t, messaging.SightFull, messaging.ParticipantSight(u.Character, room), "light %d", lamp)

		events.DrainQueuedMessagesForTest(1)
		_, err := Say("growls", mob, room)
		require.NoError(t, err)
		got := strings.Join(mobSpeechHeard(1), "\n")
		require.Contains(t, got, `says, "growls"`, "light %d: the words arrive", lamp)
		require.NotContains(t, got, "Skeleton", "light %d: a nightvision holder in a dark room cannot tell who", lamp)
	}
}

// TestMobHowlTaunt_HitFallbackHidesTargetBySight closes a coverage hole a
// review found in sight gates 5b: howl's undefended hit-fallback line
// ("throws back its head", howl.go:58-60) and taunt's undefended hit-fallback
// line ("bellows a thunderous challenge", taunt.go:75-77) each carry the
// TARGET's name in the SendTextHidingNames names list alongside the mob's, but
// no test drove either path to that line and checked the target was hidden
// too. Every existing test either exercises a different branch (fumble,
// aggro-pull, the seeded-store triad, sendChannelDefenceMessages) or checks
// only the mob's own name.
//
// Table strings are the plan's visible-lines table (docs/superpowers/plans/
// 2026-09-29-sight-gates-5b.md lines 92-112) for taunt; howl has no row of its
// own there (only its fumble and aggro-pull rows are listed), so its expected
// text is the same HideNames transform applied to howl.go's literal, which
// carries the identical two-tag shape ("mobname ... username").
func TestMobHowlTaunt_HitFallbackHidesTargetBySight(t *testing.T) {
	commands := []struct {
		name       string
		run        func(string, *mobs.Mob, *rooms.Room) (bool, error)
		wantShapes string
		wantNone   string
	}{
		{
			name:       "howl",
			run:        Howl,
			wantShapes: "A figure throws back its head and lets out a bone-chilling howl at a figure!",
			wantNone:   "Something throws back its head and lets out a bone-chilling howl at something!",
		},
		{
			name:       "taunt",
			run:        Taunt,
			wantShapes: "A figure bellows a thunderous challenge at a figure!",
			wantNone:   "Something bellows a thunderous challenge at something!",
		},
	}

	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			cleanup := seedAllRegistries()
			defer cleanup()
			restoreBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
				"cave": {BiomeId: "cave", Name: "Cave", Symbol: ".", SkyLight: rooms.SkyLightPtr(0.0), MovementCost: 1},
			})
			defer restoreBiomes()
			const infraredId = 9102
			restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
				infraredId: {ConditionId: infraredId, Name: "Test Infrared", RoundInterval: 1, TriggerCount: 1,
					Flags:   []conditions.Flag{conditions.InfraredVision},
					Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}}},
			})
			defer restoreConditions()

			mob := mobs.GetInstance(100)
			require.NotNil(t, mob)
			targetMob := mobs.GetInstance(200)
			require.NotNil(t, targetMob)

			darkRoom := rooms.LoadRoom(2)
			require.NotNil(t, darkRoom)
			darkRoom.Biome = "cave"
			require.Equal(t, 0, darkRoom.LightLevel())

			mob.Character.RoomId = darkRoom.RoomId
			targetMob.Character.RoomId = darkRoom.RoomId
			darkRoom.AddMob(mob.InstanceId)
			darkRoom.AddMob(targetMob.InstanceId)
			mob.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			require.True(t, mob.Character.IsInCombat())

			shapesListener := users.GetByUserId(1)
			noneListener := users.GetByUserId(2)
			shapesListener.Character.RoomId = darkRoom.RoomId
			noneListener.Character.RoomId = darkRoom.RoomId
			darkRoom.AddPlayer(shapesListener.UserId)
			darkRoom.AddPlayer(noneListener.UserId)
			require.True(t, shapesListener.Character.Conditions.AddCondition(infraredId, true))
			require.Equal(t, messaging.SightShapes, messaging.ParticipantSight(shapesListener.Character, darkRoom))
			require.Equal(t, messaging.SightNone, messaging.ParticipantSight(noneListener.Character, darkRoom))

			original := executeTauntAction
			executeTauntAction = func(actions.Actor) actions.TauntResult {
				return actions.TauntResult{
					Executed: true,
					Hit:      true,
					Target: actions.AggroTarget{
						Char:          &targetMob.Character,
						Name:          targetMob.Character.Name,
						MobInstanceId: targetMob.InstanceId,
						Found:         true,
					},
					Defence: combat.ChannelDefenceResult{},
				}
			}
			t.Cleanup(func() { executeTauntAction = original })

			events.DrainQueuedMessagesForTest(shapesListener.UserId)
			events.DrainQueuedMessagesForTest(noneListener.UserId)

			handled, err := command.run("", mob, darkRoom)
			require.NoError(t, err)
			require.True(t, handled)

			require.Equal(t, []string{command.wantShapes}, mobSpeechHeard(shapesListener.UserId),
				"%s hit-fallback must hide the target at shapes too", command.name)
			require.Equal(t, []string{command.wantNone}, mobSpeechHeard(noneListener.UserId),
				"%s hit-fallback must hide the target with no sight too", command.name)
		})
	}
}
