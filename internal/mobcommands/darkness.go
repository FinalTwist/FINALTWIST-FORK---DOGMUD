package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sendAudioRoomText handles audio messages (say/shout) in dark rooms.
// Players with nightvision see the full message with mob name.
// Players without nightvision see the anonymous version.
// In lit rooms, everyone sees the full message.
//
// cat selects the audio color/category (typically CategorySpeech,
// CategoryShout, CategoryNPCDialogue, etc.).
//
// Two-tier by construction: the (anonMsg, fullMsg) pair collapses SightShapes
// and SightNone into one string, the same defect canSeeInDark used to carry
// for the visual pipeline. This stays two-tier because it has 15 call sites
// across 7 files, four of them the speech commands (say.go, shout.go,
// rally.go, warcry.go), which are out of scope for M4e-1 Task 9. A caller
// that already knows the set of names to hide, rather than a hand-built
// anonymous string, wants sendAudioRoomTextHidingNames instead.
func sendAudioRoomText(room *rooms.Room, mob *mobs.Mob, cat messaging.Category, anonMsg string, fullMsg string, excludedUserIDs ...int) {
	excluded := make(map[int]struct{}, len(excludedUserIDs))
	for _, userID := range excludedUserIDs {
		excluded[userID] = struct{}{}
	}
	if room.IsLit() {
		room.SendText(cat, fullMsg, excludedUserIDs...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if _, skip := excluded[uid]; skip {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		if u.Character.HasFlagFromAnySource(conditions.NightVision) {
			u.SendText(cat, fullMsg)
		} else {
			u.SendText(cat, anonMsg)
		}
	}
}

// sendAudioRoomTextHidingNames broadcasts an audio line to the room, hiding
// each of names from every listener by that listener's own sight.
//
// The audio channel bypasses the pipeline's sight gate on purpose: you hear a
// howl whether or not you can see. But hearing it must not tell you WHO, so
// the names are hidden here, per listener, at all three tiers.
//
// messaging.ParticipantSight already folds room light, blindness, night
// vision and infrared into one verdict, and messaging.HideNames maps that
// verdict onto "a figure" or "something" -- there is no separate lit-room
// shortcut here because ParticipantSight already returns SightFull for a lit
// room on its own.
func sendAudioRoomTextHidingNames(room *rooms.Room, cat messaging.Category, fullMsg string, names []string, excludedUserIDs ...int) {
	excluded := make(map[int]struct{}, len(excludedUserIDs))
	for _, userID := range excludedUserIDs {
		excluded[userID] = struct{}{}
	}
	for _, uid := range room.GetPlayers() {
		if _, skip := excluded[uid]; skip {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		sight := messaging.ParticipantSight(u.Character, room)
		u.SendText(cat, messaging.HideNames(fullMsg, names, sight))
	}
}
