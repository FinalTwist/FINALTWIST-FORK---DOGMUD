package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/stretchr/testify/require"
)

// Review finding (movement parity 4b): a newcomer MOB rolled to spot its own
// allies. A charmed pet or AI companion following its hidden owner rolled
// against the owner every step, could reveal them, and trained its Search
// each time; an NPC party member following a hidden mob leader did the same.
// newcomerSpots now skips a mob mover's allies (alliesOf), as sneakerSpotted
// already did. Being revealed is the only way a line is written here, so a
// hider that stays hidden proves no line was sent.

// A sharp pet arriving beside its hidden owner, a hidden sibling pet and the
// owner's hidden party member rolls against none of them.
func TestEntryDetection_PetDoesNotSpotItsOwnerSide(t *testing.T) {
	w := newDetectWorld(t)
	_, owner := w.place(t, "player", 9810, "Owner")
	_, friend := w.place(t, "player", 9811, "Friend")
	_, sibling := w.place(t, "mob", 9851, "Sibling")
	inner, pet := w.place(t, "mob", 9850, "Pet")

	p := parties.New(9810)
	require.NotNil(t, p)
	require.True(t, p.InvitePlayer(9811))
	require.True(t, p.AcceptInvite(9811))
	t.Cleanup(p.Disband)

	pet.Charm(9810, -1, ``)
	sibling.Charm(9810, -1, ``)
	owner.TrackCharmed(9850, true)
	owner.TrackCharmed(9851, true)

	hideForMove(t, owner)
	hideForMove(t, friend)
	hideForMove(t, sibling)
	owner.Stats.Dexterity.ValueAdj = 0
	friend.Stats.Dexterity.ValueAdj = 0
	sibling.Stats.Dexterity.ValueAdj = 0
	pet.Stats.Perception.ValueAdj = 1000
	mover := &recordingMover{Actor: inner}

	EntryDetection(mover, w.dest, false)

	require.True(t, owner.IsHidden(), "a pet never reveals its owner")
	require.True(t, friend.IsHidden(), "a pet never reveals its owner's party")
	require.True(t, sibling.IsHidden(), "a pet never reveals its owner's other pets")
	require.Empty(t, mover.rec.awards, "no contest with an ally, so no Search award")
}

// A sharp NPC party member arriving beside its hidden mob leader leaves the
// leader hidden and earns nothing.
func TestEntryDetection_NpcPartyMemberDoesNotSpotItsLeader(t *testing.T) {
	w := newDetectWorld(t)
	leaderActor, leader := w.place(t, "mob", 9851, "Leader")
	inner, member := w.place(t, "mob", 9850, "Member")

	p := parties.NewByActor(leaderActor)
	require.NotNil(t, p)
	require.True(t, p.AddActor(inner))
	t.Cleanup(func() {
		p.RemoveActor(inner)
		p.RemoveActor(leaderActor)
	})

	hideForMove(t, leader)
	leader.Stats.Dexterity.ValueAdj = 0
	member.Stats.Perception.ValueAdj = 1000
	mover := &recordingMover{Actor: inner}

	EntryDetection(mover, w.dest, false)

	require.True(t, leader.IsHidden(), "a party member never reveals its leader")
	require.Empty(t, mover.rec.awards)
}

// A player newcomer keeps master's behaviour: it rolls against every hidden
// player but itself, its own party member included.
func TestEntryDetection_PlayerNewcomerStillRollsAgainstItsParty(t *testing.T) {
	w := newDetectWorld(t)
	_, friend := w.place(t, "player", 9811, "Friend")
	inner, mc := w.place(t, "player", 9810, "Newcomer")

	p := parties.New(9810)
	require.NotNil(t, p)
	require.True(t, p.InvitePlayer(9811))
	require.True(t, p.AcceptInvite(9811))
	t.Cleanup(p.Disband)

	hideForMove(t, friend)
	friend.Stats.Dexterity.ValueAdj = 0
	mc.Stats.Perception.ValueAdj = 1000
	mover := &recordingMover{Actor: inner}

	EntryDetection(mover, w.dest, false)

	require.False(t, friend.IsHidden(), "unchanged from master: a player spots a hidden party member")
	require.Len(t, mover.rec.awards, 1)
}
