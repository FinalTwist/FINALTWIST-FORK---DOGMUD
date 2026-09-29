package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
)

// The context answers "who is who" the same way for every pairing, and takes
// refs from the actor: fixture mob 100 has no Character.MobInstanceId of its
// own in production-shaped saves either, which is why charActorRef is only
// the fallback.
func TestSpellEffectCtx_NamesAndRefsPerPairing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := physicalHarmSpellForCollapseTest()

	pm := newSpellEffectCtx(f.casterUser.Character, actions.NewUserActorInRoom(f.casterUser, f.room),
		actions.NewMobActorInRoom(f.targetMob, f.room), f.room, spell, 30, spellContestAttackWin())
	assert.Same(t, f.casterUser, pm.casterUser())
	assert.Nil(t, pm.casterMob())
	assert.Same(t, f.targetMob, pm.targetMob())
	assert.Nil(t, pm.targetUser())
	assert.Equal(t, state.ActorRef{UserId: 1}, pm.casterRef())
	assert.Equal(t, state.ActorRef{MobInstanceId: 101}, pm.targetRef())
	assert.Equal(t, `<ansi fg="username">Aliceia</ansi>`, pm.casterName())
	assert.Equal(t, mobDisplayName(f.targetMob, f.room, 1), pm.targetName())
	assert.Equal(t, 1, pm.viewerId())

	mp := newSpellEffectCtx(&f.casterMob.Character, actions.NewMobActorInRoom(f.casterMob, f.room),
		actions.NewUserActorInRoom(f.targetUser, f.room), f.room, spell, 30, spellContestAttackWin())
	assert.Same(t, f.casterMob, mp.casterMob())
	assert.Same(t, f.targetUser, mp.targetUser())
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, mp.casterRef())
	assert.Equal(t, state.ActorRef{UserId: 2}, mp.targetRef())
	assert.Equal(t, mobDisplayName(f.casterMob, f.room, 0), mp.casterName())
	assert.Equal(t, 0, mp.viewerId())

	anon := newSpellEffectCtx(nil, nil, actions.NewMobActorInRoom(f.targetMob, f.room),
		f.room, spell, 30, spellContestAttackWin())
	assert.Nil(t, anon.casterUser())
	assert.Nil(t, anon.casterMob())
	assert.Equal(t, "something", anon.casterName())
	assert.True(t, anon.casterRef().IsZero())
}
