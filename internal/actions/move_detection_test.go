package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/progression"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// pinDetectionKnobs removes the contest floor and gap compression so scores
// 1000 apart decide every roll (fact V35).
func pinDetectionKnobs(t *testing.T) {
	t.Helper()
	c := configs.GetConfig()
	c.Balance.ContestFloor = 0
	c.Balance.ContestGapSaturation = 0
	configs.SetConfigForTest(t, c)
}

// hideForMove puts c into the Hidden awareness state. (Named apart from the
// household bauble tests' hide helper.)
func hideForMove(t *testing.T, c *characters.Character) {
	t.Helper()
	if c.Awareness == nil {
		c.Awareness = awareness.NewMachine()
	}
	reason := state.TransitionReason{Trigger: "move_detection_test"}
	require.NoError(t, c.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
	c.Awareness.ResolveConcealment(true, reason)
	require.True(t, c.IsHidden())
}

// recordingMover wraps an Actor and records AwardResolved instead of paying it.
type recordingMover struct {
	Actor
	rec awardRecorder
}

func (r *recordingMover) AwardResolved(won bool, cands ...progression.Candidate) {
	r.rec.AwardResolved(won, cands...)
}

type detectWorld struct {
	dest *rooms.Room
}

func newDetectWorld(t *testing.T) *detectWorld {
	t.Helper()
	pinDetectionKnobs(t)
	dest := &rooms.Room{RoomId: 9800, Zone: "MoveDetect", Lamp: rooms.LampPtr(80)}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{9800: dest}, map[string]*rooms.ZoneConfig{}))
	return &detectWorld{dest: dest}
}

// place puts a character of the given kind into dest and returns its actor
// and character. Player ids start at 9810, mob instances at 9850.
func (w *detectWorld) place(t *testing.T, kind string, id int, name string) (Actor, *characters.Character) {
	t.Helper()
	if kind == "player" {
		u := users.NewTestUser(id, name, name, 0)
		u.Character.RoomId = w.dest.RoomId
		existing := map[int]*users.UserRecord{}
		for _, pid := range w.dest.GetPlayers() {
			if p := users.GetByUserId(pid); p != nil {
				existing[pid] = p
			}
		}
		existing[id] = u
		t.Cleanup(users.SeedUsersForTest(existing))
		w.dest.AddPlayer(id)
		return NewUserActorInRoom(u, w.dest), u.Character
	}
	m := &mobs.Mob{InstanceId: id}
	m.Character = *characters.New()
	m.Character.Name = name
	m.Character.RoomId = w.dest.RoomId
	mobs.SetInstanceForTest(id, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(id, nil) })
	w.dest.AddMob(id)
	return NewMobActorInRoom(m, w.dest), &m.Character
}

var detectPairings = []struct{ mover, other string }{
	{"player", "player"}, {"player", "mob"}, {"mob", "player"}, {"mob", "mob"},
}

func otherId(kind string) int {
	if kind == "player" {
		return 9811
	}
	return 9851
}

func moverId(kind string) int {
	if kind == "player" {
		return 9810
	}
	return 9850
}

// A clumsy sneaking mover walking in on a sharp observer is spotted: it stops
// sneaking and is no longer hidden. All four mover and observer pairings.
func TestEntryDetection_SneakingMoverIsSpotted(t *testing.T) {
	for _, p := range detectPairings {
		t.Run(p.mover+" sneaks past a "+p.other, func(t *testing.T) {
			w := newDetectWorld(t)
			_, obs := w.place(t, p.other, otherId(p.other), "Watcher")
			obs.Stats.Perception.ValueAdj = 1000
			mover, mc := w.place(t, p.mover, moverId(p.mover), "Sneak")
			mc.Stats.Dexterity.ValueAdj = 0
			hideForMove(t, mc)
			mc.SetMiscData(`sneaking`, true)

			got := EntryDetection(mover, w.dest, true)

			require.False(t, got.StillSneaking)
			require.False(t, mc.IsHidden(), "a spotted sneaker is revealed")
			require.Nil(t, mc.GetMiscData(`sneaking`))
		})
	}
}

// A skilled sneaker walking in on a blind observer stays hidden.
func TestEntryDetection_SkilledSneakerSlipsIn(t *testing.T) {
	for _, p := range detectPairings {
		t.Run(p.mover+" sneaks past a "+p.other, func(t *testing.T) {
			w := newDetectWorld(t)
			_, obs := w.place(t, p.other, otherId(p.other), "Watcher")
			obs.Stats.Perception.ValueAdj = 0
			mover, mc := w.place(t, p.mover, moverId(p.mover), "Sneak")
			mc.Stats.Dexterity.ValueAdj = 1000
			hideForMove(t, mc)

			got := EntryDetection(mover, w.dest, true)

			require.True(t, got.StillSneaking)
			require.True(t, mc.IsHidden())
		})
	}
}

// A sharp newcomer spots a clumsy hider and earns a winning Search award; a
// blind one misses a skilled hider and still earns a losing award (U10b-2:
// both outcomes fire).
func TestEntryDetection_NewcomerRollsForHiders(t *testing.T) {
	for _, p := range detectPairings {
		for _, sharp := range []bool{true, false} {
			name := p.mover + " walks in on a hidden " + p.other
			if !sharp {
				name += " and misses it"
			}
			t.Run(name, func(t *testing.T) {
				w := newDetectWorld(t)
				_, hc := w.place(t, p.other, otherId(p.other), "Lurker")
				hideForMove(t, hc)
				inner, mc := w.place(t, p.mover, moverId(p.mover), "Newcomer")
				if sharp {
					mc.Stats.Perception.ValueAdj = 1000
					hc.Stats.Dexterity.ValueAdj = 0
				} else {
					mc.Stats.Perception.ValueAdj = 0
					hc.Stats.Dexterity.ValueAdj = 1000
				}
				mover := &recordingMover{Actor: inner}

				got := EntryDetection(mover, w.dest, false)

				require.False(t, got.StillSneaking)
				require.Equal(t, !sharp, hc.IsHidden(), "the hider is revealed exactly when spotted")
				require.Len(t, mover.rec.awards, 1, "one Search award per hidden occupant")
				require.Equal(t, sharp, mover.rec.awards[0].won)
				require.Equal(t, string(skills.Search), mover.rec.awards[0].cands[0].Skill)
			})
		}
	}
}

// The mover never rolls against itself: a hidden mob walking in with no one
// else present awards nothing.
func TestEntryDetection_SkipsTheMover(t *testing.T) {
	w := newDetectWorld(t)
	inner, mc := w.place(t, "mob", 9850, "Loner")
	hideForMove(t, mc)
	mover := &recordingMover{Actor: inner}
	EntryDetection(mover, w.dest, false)
	require.Empty(t, mover.rec.awards)
	require.True(t, mc.IsHidden())
}
