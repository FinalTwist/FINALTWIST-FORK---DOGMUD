package baubles

import (
	"sync"
	"time"
)

// Roll windows: each room offers BaubleRollsPerWindow bauble rolls, then no
// more until BaubleWindowMinutes (real time) after the FIRST roll of that
// window. Every player search in the room spends one roll, found or not.
//
// Kept here rather than in the room's temp data because rooms unload when
// nobody is near: a window stored on the room would reset whenever a player
// walked out of range and back. Kept in memory rather than on disk because a
// restart reopening every window is harmless.
//
// With BaubleWindowPerPlayer the key also carries the player, so each player
// has their own rolls in each room; by default the rolls are shared, so a
// group searching together gets no more chances than one player.
//
// A feature of the room (`search bookshelf`) has a window of its own: it can
// be searched ONCE per BaubleFeatureWindowMinutes (60), whatever the search
// turns up, and a search of it never spends the room's rolls (nor the
// room's its). The search itself claims it (ClaimFeatureSearch), before any
// roll, so a feature in a room that offers no baubles is still searched only
// once an hour.

type windowKey struct {
	roomId  int
	userId  int    // 0 when windows are shared by the room
	feature string // "" for the room itself; a noun or container for a targeted search
}

type rollWindow struct {
	ends time.Time // when this window closes and the next search opens a new one
	used int
	by   int // the user who opened it (for "you searched it" versus "someone did")
}

var (
	windowsMu sync.Mutex
	windows   = map[windowKey]*rollWindow{}
)

// pruneAt is the map size past which expired windows are swept out on the
// next roll, so the map stays bounded by the rooms searched in one window.
const pruneAt = 2048

func keyFor(roomId int, userId int, feature string, perPlayer bool) windowKey {
	key := windowKey{roomId: roomId, feature: feature}
	if perPlayer {
		key.userId = userId
	}
	return key
}

// liveWindowLocked returns the window for key if it has not expired. Call
// with windowsMu held.
func liveWindowLocked(key windowKey, now time.Time) (*rollWindow, bool) {
	w, ok := windows[key]
	if !ok || !now.Before(w.ends) {
		return nil, false
	}
	return w, true
}

// pruneLocked sweeps expired windows once the map is large. Call with
// windowsMu held.
func pruneLocked(now time.Time) {
	if len(windows) <= pruneAt {
		return
	}
	for k, w := range windows {
		if !now.Before(w.ends) {
			delete(windows, k)
		}
	}
}

// takeRoll spends one of the room's rolls (and player's, when per-player)
// if its window has one left, opening a new window if the last has expired.
// It reports whether a roll was available.
func takeRoll(roomId int, userId int, perPlayer bool, rolls int, length time.Duration, now time.Time) bool {
	key := keyFor(roomId, userId, ``, perPlayer)

	windowsMu.Lock()
	defer windowsMu.Unlock()
	pruneLocked(now)

	w, ok := liveWindowLocked(key, now)
	if !ok {
		w = &rollWindow{ends: now.Add(length), by: userId}
		windows[key] = w
	}
	if w.used >= rolls {
		return false
	}
	w.used++
	return true
}

// FeatureSearchable reports whether this player may search the feature now.
// When not, byYou says whether the search that closed it was this player's,
// and reopens is when it may be searched again.
func FeatureSearchable(roomId int, userId int, feature string, now time.Time) (ok bool, byYou bool, reopens time.Time) {
	s := currentSettings()
	windowsMu.Lock()
	defer windowsMu.Unlock()
	w, live := liveWindowLocked(keyFor(roomId, userId, feature, s.perPlayer), now)
	if !live {
		return true, false, time.Time{}
	}
	return false, w.by == userId, w.ends
}

// ClaimFeatureSearch records a search of the feature, which may then not be
// searched again for BaubleFeatureWindowMinutes. It returns false, and
// records nothing, if the feature was already searched in its window. Call
// it for every feature search, before rolling (RollFind with Feature set
// does not claim).
func ClaimFeatureSearch(roomId int, userId int, feature string, now time.Time) bool {
	if feature == `` {
		return false
	}
	s := currentSettings()
	key := keyFor(roomId, userId, feature, s.perPlayer)

	windowsMu.Lock()
	defer windowsMu.Unlock()
	pruneLocked(now)

	if _, live := liveWindowLocked(key, now); live {
		return false
	}
	windows[key] = &rollWindow{ends: now.Add(s.featureWindow), used: 1, by: userId}
	return true
}

// WindowState reports a room's (or, per player, a room and player's) roll
// window: rolls used, rolls allowed, and when it reopens. open is false when
// no window is active (the next search starts a fresh one). For the admin
// command.
func WindowState(roomId int, userId int, now time.Time) (used int, allowed int, reopens time.Time, open bool) {
	return FeatureWindowState(roomId, userId, ``, now)
}

// FeatureWindowState is WindowState for one feature of the room (a noun or
// container searched on its own), whose window allows one search. An empty
// feature is the room itself.
func FeatureWindowState(roomId int, userId int, feature string, now time.Time) (used int, allowed int, reopens time.Time, open bool) {
	s := currentSettings()
	allowed = s.rolls
	if feature != `` {
		allowed = 1
	}
	windowsMu.Lock()
	defer windowsMu.Unlock()
	w, live := liveWindowLocked(keyFor(roomId, userId, feature, s.perPlayer), now)
	if !live {
		return 0, allowed, time.Time{}, false
	}
	return w.used, allowed, w.ends, true
}

// ResetWindow forgets a room's windows, its features' included (every
// player's, when per-player), so the next search starts a fresh one. For the
// admin command and tests.
func ResetWindow(roomId int) {
	windowsMu.Lock()
	defer windowsMu.Unlock()
	for k := range windows {
		if k.roomId == roomId {
			delete(windows, k)
		}
	}
}

// resetAllWindowsForTest clears every window.
func resetAllWindowsForTest() {
	windowsMu.Lock()
	windows = map[windowKey]*rollWindow{}
	windowsMu.Unlock()
}
