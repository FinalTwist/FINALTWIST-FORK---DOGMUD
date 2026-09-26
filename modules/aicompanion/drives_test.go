package aicompanion

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// The engine's dismiss asks, per companion, whether this module drives it:
// only while the module is on, and only for a companion whose profile it
// has.
func TestDrivesBondedIsPerCompanion(t *testing.T) {
	m := &AICompanionModule{cfg: Config{Enabled: true}, byMob: map[int]*Profile{9800: {Id: `mara`, MobId: 9800}}}
	if !m.drivesBonded(9800) {
		t.Fatal("a companion whose profile is loaded is driven")
	}
	if m.drivesBonded(9902) {
		t.Fatal("a bonded companion with no profile here is driven by nobody")
	}
	m.cfg.Enabled = false
	if m.drivesBonded(9800) {
		t.Fatal("switched off, the module drives nothing")
	}
}

// A bonded companion her owner dismissed while the module was off does not
// come back on its own when it is switched on: the owner parted with her,
// which is what companion-part does while it is on, and that is for good.
// The bond record's Met is what keeps her away (a character who never met
// one is still met, AutoBondExisting).
func TestDismissedWhileOffDoesNotReturnOnItsOwn(t *testing.T) {
	owner, _, _, _ := harmWorld(t, configs.PVPDisabled)
	m, _ := consentModule(0)
	m.cfg.Enabled, m.cfg.AutoBond, m.cfg.AutoBondExisting = true, true, true
	m.cfg.MeetDelayRounds = 0

	// Met, and no bonded companion on the character any more.
	m.bonds.Users[1] = &bondRecord{Profile: `mara`, Met: true, Unix: time.Now().Unix()}
	// One round is enough to see it: a meeting starts by waiting on the
	// room (pendingMeet); the fixture cannot field a companion, so a later
	// round would drop the wait for that reason instead.
	m.considerMeeting(owner, 1)
	if _, waiting := m.pendingMeet[1]; waiting {
		t.Fatal("a companion her owner parted with does not walk back up to them")
	}

	// The control: a character who never met one is met.
	delete(m.bonds.Users, 1)
	m.considerMeeting(owner, 10)
	if _, waiting := m.pendingMeet[1]; !waiting {
		t.Fatal("control: a character with no bond at all is waited on for a meeting")
	}
}
