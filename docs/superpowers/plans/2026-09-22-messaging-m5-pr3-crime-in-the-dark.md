# M5 PR 3: Crime in the Dark Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Crime witnessing stops being sight-blind: a mob that cannot see a crime does not witness it, and a mob that makes out only shapes records the crime against an unknown perpetrator.

**Architecture:** `crimes.WitnessesInRoom` classifies each candidate mob through the two composed messaging predicates rather than returning a flat list, so the result carries both tiers and the compiler enumerates every consumer. Nothing new is invented: `PerpUnknown` already exists, the three sight tiers already exist, and sleep is already composed by `CanSeeClearly`/`CanSeeShapes`.

**Tech Stack:** Go, `internal/crimes`, `internal/messaging` (predicates), `internal/actions`, `internal/hooks`, `internal/seeders`, `internal/knowledge`.

---

## Facts verified against source

Read from the tree on 2026-09-22 at `a5613be95`. Rows marked PROBED were
reproduced by running code, not by reading it.

| # | Fact | Evidence |
|---|------|----------|
| 1 | `WitnessesInRoom` consults no sight, no lighting and no sleep. A witness is any mob whose `Groups` overlap the victim's factions | `internal/crimes/crimes.go:198-228` |
| 2 | Witnesses are MOBS, never players. It walks `room.GetMobs()` | `crimes.go:207` |
| 3 | `IdentifiedPerp` already returns `PerpUnknown` for an empty witness list | `crimes.go:232-237` |
| 4 | `crimes` already imports `rooms`, `mobs`, `factions`, `configs`, `mudlog`, `util`. It does NOT import `messaging` yet | `crimes.go:3-10` |
| 5 | Adding `crimes -> messaging` is cycle-safe: `messaging` imports nothing back toward `crimes`, `rooms`, `mobs`, `actions` or `hooks` | `internal/messaging/predicates.go:1-7` |
| 6 | `CanSeeClearly` is `awake(observer) && ParticipantSight(...) == SightFull` | `predicates.go:92-94` |
| 7 | `CanSeeShapes` is `awake(observer)` then `SightFull || SightShapes` | `predicates.go:139-145` |
| 8 | `awake` is `observer == nil \|\| !observer.HasConditionFlag(conditions.Sleeping)` | `predicates.go:70-74` |
| 9 | `ParticipantSight` deliberately excludes sleep, and its docstring routes non-party observers to the two composed predicates | `predicates.go:31-47` |
| 10 | `mobs.Mob.Character` is a plain `characters.Character` value, so `&mob.Character` is the argument | used at `crimes.go` sites and throughout `internal/actions` |
| 11 | There are FIVE call sites, not four | `aggression.go:31,36`; `steal.go:313,316`; `plant.go:223,226`; `MobDeath_FactionRep.go:60`; `seeders/aggressive_action_to_revenge.go:68` |
| 12 | All four crime call sites wrap knowledge writes in `if perp.Type == crimes.PerpPlayer`, and iterate the FULL witness list inside it | `aggression.go:44-58`, `steal.go:322-338`, `plant.go:236-248`, `MobDeath_FactionRep.go` `writeKnowledgeForWitnesses` |
| 13 | 🔴 PROBED. `(&rooms.Room{RoomId: 467}).GetVisibility()` PANICS in the `crimes` test binary: no biomes are loaded, `GetBiome` returns nil, and `IsDark()` dereferences it | run 2026-09-22; `rooms.go:145-155`, `biomes.go:65-67`, `biomes.go:150-156` |
| 14 | 🔴 PROBED. Loading conditions in a test binary panics on `conditionId 0 (Meditating) has a TriggersCount of < 1` unless `Network.LogoutRounds` is set first | run 2026-09-22; the standing `ConditionSpec.Validate` trap |
| 15 | 🔴 PROBED. A `characters.Character` struct literal panics with "assignment to entry in nil map" on `AddCondition`. Use `characters.New()` | run 2026-09-22; `internal/characters/character.go:382` |
| 16 | PROBED. With biomes loaded, a default-biome room is visibility 2 and a `Biome: "cave"` room is visibility 0 | run 2026-09-22 |
| 17 | PROBED. The full tier matrix (see below) behaves exactly as the design requires, including the sleep gate firing in a LIT room | run 2026-09-22 |
| 18 | Two existing tests call `WitnessesInRoom` and index the result as a slice | `crimes_test.go:253-256`, `:269-272` |
| 19 | One existing test calls `IdentifiedPerp` with a slice literal | `crimes_test.go:222-227` |
| 20 | `crimes` `TestMain` today only sets up the logger | `internal/crimes/test_main_test.go` |
| 21 | `classifyWitnessResponse` already sorts witnesses three ways: guard to `ResponseReportOnly` (a no-op), noncombatant to `ResponseAlarm`, everything else to `ResponseRevenge` | `internal/seeders/witness_response.go:26-36` |
| 22 | The two responses differ in exactly the way the sight tiers care about. `alarmReaction` emotes and steps toward an exit and NAMES NOBODY; `seedRevengeGoalIfAbsent(m, "player", playerId, priority)` targets the player BY ID | `witness_response.go:57-75` and `:44-46` |
| 23 | `alarmReaction` is unexported and reachable only through `seedWitnessResponse` | `witness_response.go:64` |
| 24 | 🔑 **The suspected free-reputation exploit does NOT exist.** `FindRecentAssault` skips any row where `c.Perpetrator.Type != PerpPlayer \|\| c.Perpetrator.Id != userId`, so an unattributed assault row is never found, murder Case C is unreachable for it, and the kill takes the fresh-record path instead. Chased and disproved 2026-09-22; do not re-raise | `internal/crimes/crimes.go`, `FindRecentAssault` perp filter |
| 25 | `crimes.Record` fires unconditionally, OUTSIDE the `perp.Type == PerpPlayer` guard, and already stores `RoomId` and `Zone` | `aggression.go:39-41`; `crimes.go:85-91` |
| 26 | Condition 15 (Sleeping) wakes on `cancel-on-action`, `cancel-on-combat` and `cancel-on-damage` only. There is no proximity or noise wake | `_datafiles/world/dogmud/conditions/15-sleeping.yaml:14-19` |

### The probed tier matrix

Built with `characters.New()`, conditions loaded, against a default-biome room
(lit) and a `Biome: "cave"` room (unlit). This is measured output, not a
prediction.

| Mob | lit: clear | lit: shapes | dark: clear | dark: shapes | Tier when lit | Tier when dark |
|---|---|---|---|---|---|---|
| plain | true | true | false | false | Identifying | **not a witness** |
| condition 29 NightVision | true | true | true | true | Identifying | Identifying |
| condition 85 InfraredVision | true | true | false | true | Identifying | **ShapesOnly** |
| condition 15 Sleeping | **false** | **false** | false | false | **not a witness** | not a witness |

The sleeping row is the owner's sleep gate, and it fires in a LIT room, which
is the case that proves attention is being consulted and not just optics.

### 🔴 A correction to the design's knowledge rule

The spec says a shapes-only witness should get `RecordCrimeWitnessed` but not
`RecordMet`. **Read against source, that split would itself leak identity.**
Both calls are keyed on `knowledge.PlayerSubject(userId)`, so
`RecordCrimeWitnessed` records "this mob knows player X committed crime Y".
A witness that made out only a figure does not know that.

The correct rule is simpler and needs no split: **the knowledge loop iterates
`Identifying` only.** Both calls stay together.

This matters in exactly one situation, and it is the situation that makes it a
real defect rather than a technicality: when a room holds BOTH an identifying
witness and a shapes-only one, `perp` is `PerpPlayer`, the
`if perp.Type == PerpPlayer` guard opens, and today's loop would write
player-subject knowledge for **every** witness including the blind one.

---

## Task 1: Give the crimes test binary a world to see by

Nothing in this PR can be tested until the test binary can compute room
lighting at all. Fact 13 says it currently panics.

**Files:**
- Modify: `internal/crimes/test_main_test.go`
- Create: `internal/crimes/sight_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/crimes/sight_test.go`:

```go
package crimes

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// newCharWithCondition builds a character the way production does. A struct
// literal panics on AddCondition with "assignment to entry in nil map",
// because characters.New() is what allocates the condition maps.
func newCharWithCondition(t *testing.T, name string, conditionId int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Name = name
	if conditionId > 0 {
		if err := c.AddCondition(conditionId, true); err != nil {
			t.Fatalf("AddCondition(%d): %v", conditionId, err)
		}
	}
	return c
}

// TestSightTiersBehaveAsWitnessGateExpects pins the behaviour the witness gate
// is built on, in this package's own test binary. If TestMain stops loading
// biomes or conditions, this fails here rather than somewhere confusing.
func TestSightTiersBehaveAsWitnessGateExpects(t *testing.T) {
	lit := &rooms.Room{RoomId: 467}
	dark := &rooms.Room{RoomId: 468, Biome: "cave"}

	if got := lit.GetVisibility(); got < 1 {
		t.Fatalf("default-biome room visibility = %d, want >= 1 (lit)", got)
	}
	if got := dark.GetVisibility(); got != 0 {
		t.Fatalf("cave room visibility = %d, want 0 (unlit)", got)
	}

	tests := []struct {
		name        string
		conditionId int
		room        *rooms.Room
		wantClear   bool
		wantShapes  bool
	}{
		{"plain mob in a lit room", 0, lit, true, true},
		{"plain mob in the dark", 0, dark, false, false},
		{"nightvision mob in the dark", 29, dark, true, true},
		{"infrared mob in the dark", 85, dark, false, true},
		{"sleeping mob in a LIT room", 15, lit, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := newCharWithCondition(t, tc.name, tc.conditionId)
			if got := messaging.CanSeeClearly(ch, tc.room); got != tc.wantClear {
				t.Errorf("CanSeeClearly = %v, want %v", got, tc.wantClear)
			}
			if got := messaging.CanSeeShapes(ch, tc.room); got != tc.wantShapes {
				t.Errorf("CanSeeShapes = %v, want %v", got, tc.wantShapes)
			}
		})
	}
}
```

- [ ] **Step 2: Run it and watch it panic**

Run: `go test ./internal/crimes/ -run TestSightTiersBehaveAsWitnessGateExpects -v`

Expected: PANIC, `runtime error: invalid memory address or nil pointer
dereference`, raised inside `GetVisibility`. This is fact 13 and it is the
reason Task 1 exists.

- [ ] **Step 3: Teach TestMain to load biomes and conditions**

Replace `internal/crimes/test_main_test.go` in full:

```go
package crimes

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// The witness gate reads room lighting and condition flags, so this
	// binary needs real biome and condition data.
	//
	// Network.LogoutRounds must be set BEFORE conditions load. A test binary
	// gets the Go default of 0, ConditionSpec.Validate overwrites condition
	// 0's triggercount from it, and then refuses any value below 1, so the
	// load panics on 0-meditating.yaml.
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":  "../../_datafiles/world/dogmud",
		"Network.LogoutRounds": 3,
	})
	rooms.LoadBiomeDataFiles()
	conditions.LoadDataFiles()

	os.Exit(m.Run())
}
```

- [ ] **Step 4: Run it and watch it pass**

Run: `go test ./internal/crimes/ -run TestSightTiersBehaveAsWitnessGateExpects -v`

Expected: PASS, with all five subtests green and two log lines reporting
`loadedCount=18` biomes and `loadedCount=109` conditions.

- [ ] **Step 5: Run the whole package to prove nothing else broke**

Run: `go test ./internal/crimes/`

Expected: `ok  github.com/GoMudEngine/GoMud/internal/crimes`

- [ ] **Step 6: Commit**

```bash
git add internal/crimes/test_main_test.go internal/crimes/sight_test.go
git commit -F - <<'EOF'
test(crimes): give the test binary biomes and conditions

The witness gate about to land reads room lighting and condition flags.
Neither works in this binary today: with no biomes loaded, GetBiome
returns nil and Room.GetVisibility dereferences it, so any test that
touches lighting panics rather than failing.

Loading conditions needs Network.LogoutRounds set first. A test binary
gets the Go default of 0, ConditionSpec.Validate overwrites condition
0's triggercount from it and then refuses anything below 1, so the load
panics on 0-meditating.yaml.

The new test pins the tier matrix the gate is built on, including the
case that proves attention is consulted and not just optics: a sleeping
mob in a LIT room can see nothing.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: The Witnesses type and the gate

Changing the return type is what makes the compiler enumerate every consumer,
which is this project's refactoring idiom: change the declaration first and let
the build find the call sites.

**Names fixed here and used unchanged by every later task:** type `Witnesses`,
fields `Identifying` and `ShapesOnly`. **No helper methods.**

🔑 **Why no `All()` or `Any()`.** An earlier draft of this plan gave the type
both. After the owner's 2026-09-22 rulings, every production consumer reads
`Identifying`: `HadExternalWitness` and `currentExternal` take it (Tasks 3 to
6), and the witness response splits by classification rather than by list
(Task 7). `ShapesOnly` is read in exactly one place, Task 7's second loop.
Shipping two exported methods that nothing calls would be dead API on day one.
If a future consumer genuinely wants "did anybody notice at all", add the
method then, with its caller.

**Files:**
- Modify: `internal/crimes/crimes.go:193-237`
- Modify: `internal/crimes/crimes_test.go:222-273`
- Test: `internal/crimes/crimes_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/crimes/sight_test.go`:

```go
// TestWitnessesInRoom_SplitsBySight is the gate itself: three mobs of the
// same faction in one unlit room, one of each sight tier.
func TestWitnessesInRoom_SplitsBySight(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	dark := &rooms.Room{RoomId: 468, Biome: "cave"}

	blind := &mobs.Mob{MobId: 100, InstanceId: 100, Groups: []string{"thornwall_citizens"}}
	blind.Character = *newCharWithCondition(t, "Blind Citizen", 0)
	shapes := &mobs.Mob{MobId: 101, InstanceId: 101, Groups: []string{"thornwall_citizens"}}
	shapes.Character = *newCharWithCondition(t, "Infrared Citizen", 85)
	// Not named `clear`: that shadows the builtin added in Go 1.21.
	clearSight := &mobs.Mob{MobId: 102, InstanceId: 102, Groups: []string{"thornwall_citizens"}}
	clearSight.Character = *newCharWithCondition(t, "Nightvision Citizen", 29)

	for _, m := range []*mobs.Mob{blind, shapes, clearSight} {
		mobs.SetInstanceForTest(m.InstanceId, m)
		defer mobs.SetInstanceForTest(m.InstanceId, nil)
		dark.AddMob(m.InstanceId)
	}

	got := WitnessesInRoom([]string{"thornwall_citizens"}, dark, 0)

	if len(got.Identifying) != 1 || got.Identifying[0] != 102 {
		t.Errorf("Identifying = %v, want [102] (the nightvision mob)", got.Identifying)
	}
	if len(got.ShapesOnly) != 1 || got.ShapesOnly[0] != 101 {
		t.Errorf("ShapesOnly = %v, want [101] (the infrared mob)", got.ShapesOnly)
	}
	// The blind mob appears in neither list: it is not a witness at all.
	if total := len(got.Identifying) + len(got.ShapesOnly); total != 2 {
		t.Errorf("total witnesses = %d, want 2: the blind mob must not be a witness", total)
	}
}

// TestIdentifiedPerp_ShapesOnlyIsUnknown is the owner's ruling in one
// assertion: a crime nobody could identify is recorded against nobody.
func TestIdentifiedPerp_ShapesOnlyIsUnknown(t *testing.T) {
	got := IdentifiedPerp(17, Witnesses{ShapesOnly: []int{101}})
	if got.Type != PerpUnknown {
		t.Errorf("got %+v, want PerpUnknown: a shapes-only witness cannot name anyone", got)
	}
}

// TestWitnessesInRoom_SleeperInLitRoomIsNotAWitness proves the sleep gate
// reaches the crime path, not just the predicates.
func TestWitnessesInRoom_SleeperInLitRoomIsNotAWitness(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	lit := &rooms.Room{RoomId: 467}
	sleeper := &mobs.Mob{MobId: 103, InstanceId: 103, Groups: []string{"thornwall_citizens"}}
	sleeper.Character = *newCharWithCondition(t, "Sleeping Citizen", 15)
	mobs.SetInstanceForTest(103, sleeper)
	defer mobs.SetInstanceForTest(103, nil)
	lit.AddMob(103)

	got := WitnessesInRoom([]string{"thornwall_citizens"}, lit, 0)
	if len(got.Identifying) > 0 || len(got.ShapesOnly) > 0 {
		t.Errorf("witnesses = %+v, want empty: a sleeping mob witnesses nothing even in a lit room", got)
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/mobs"` to `sight_test.go`'s imports.

- [ ] **Step 2: Run it and watch it fail to compile**

Run: `go test ./internal/crimes/ -run TestWitnessesInRoom_SplitsBySight`

Expected: build failure, `got.Identifying undefined (type []int has no field
or method Identifying)`. The type does not exist yet.

- [ ] **Step 3: Add the type and the gate**

In `internal/crimes/crimes.go`, add `"github.com/GoMudEngine/GoMud/internal/messaging"` to the import block, then replace `WitnessesInRoom` and `IdentifiedPerp` (currently `:193-237`) with:

```go
// Witnesses splits the mobs that noticed a crime by whether they could make
// out who did it. A mob that saw only shapes knows a crime happened and
// cannot name anyone for it.
//
// The split exists because a flat list forced one answer to two different
// questions: "did anybody notice?" and "can anybody name the perpetrator?".
// Every consumer turned out to want the second one, which is why this type
// carries two plain fields and no helper methods: see the note below.
type Witnesses struct {
	// Identifying witnesses saw the room clearly and can name the perp.
	Identifying []int
	// ShapesOnly witnesses made out movement but no faces. They record that
	// a crime happened; they never attribute it.
	ShapesOnly []int
}

// WitnessesInRoom returns the mob instance IDs in the given room whose mob
// template's Groups overlap any of factionIds, split by what each one could
// actually see.
//
// Pass excludeInstanceId = victim's instance for murder (victim is dead, not
// a self-witness); pass 0 for assault and theft (victim is alive and a
// self-witness).
//
// SIGHT. A mob judges the crime through messaging.CanSeeClearly and
// messaging.CanSeeShapes, NOT the raw ParticipantSight. The primitive is
// optics only and deliberately excludes sleep; these two compose attention on
// top of it, and its own docstring routes observers who are not a party to
// the event through exactly this pair. A crime witness is that observer, so
// a sleeping mob witnesses nothing, in a lit room or a dark one, with no
// separate sleep check written here.
//
// Order matters: CanSeeShapes is true for full sight as well, so the clear
// test runs first.
func WitnessesInRoom(factionIds []string, room *rooms.Room, excludeInstanceId int) Witnesses {
	var out Witnesses
	if room == nil || len(factionIds) == 0 {
		return out
	}
	wantSet := make(map[string]struct{}, len(factionIds))
	for _, fid := range factionIds {
		wantSet[fid] = struct{}{}
	}

	for _, instId := range room.GetMobs() {
		if instId == excludeInstanceId {
			continue
		}
		mob := mobs.GetInstance(instId)
		if mob == nil {
			continue
		}
		// FactionsForMob would re-walk the registry; we just need
		// "does mob.Groups overlap factionIds" - cheap to do inline.
		for _, g := range mob.Groups {
			if _, hit := wantSet[g]; hit {
				if factions.GetDefinition(g) != nil {
					switch {
					case messaging.CanSeeClearly(&mob.Character, room):
						out.Identifying = append(out.Identifying, instId)
					case messaging.CanSeeShapes(&mob.Character, room):
						out.ShapesOnly = append(out.ShapesOnly, instId)
					}
					break
				}
			}
		}
	}
	return out
}

// IdentifiedPerp returns PerpPlayer if at least one witness saw the room
// clearly, otherwise PerpUnknown.
//
// A room full of witnesses who made out only shapes yields PerpUnknown: the
// crime is recorded and nobody can be charged for it.
func IdentifiedPerp(userId int, w Witnesses) Perpetrator {
	if len(w.Identifying) == 0 {
		return Perpetrator{Type: PerpUnknown}
	}
	return Perpetrator{Type: PerpPlayer, Id: userId}
}
```

- [ ] **Step 4: Update the three existing tests the compiler now rejects**

In `internal/crimes/crimes_test.go`:

There are exactly two `IdentifiedPerp` calls in the whole test tree, both in
this file. At `:216`, change `IdentifiedPerp(17, []int{})` to
`IdentifiedPerp(17, Witnesses{})`. At `:223`, change
`IdentifiedPerp(17, []int{42})` to
`IdentifiedPerp(17, Witnesses{Identifying: []int{42}})`.

At `:253-256`, change to:

```go
	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 300)
	if len(got.Identifying) != 1 || got.Identifying[0] != 100 {
		t.Errorf("WitnessesInRoom Identifying = %v, want [100]", got.Identifying)
	}
```

At `:269-272`, change to:

```go
	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 0) // assault: include victim
	if len(got.Identifying) != 1 || got.Identifying[0] != 300 {
		t.Errorf("WitnessesInRoom Identifying = %v, want [300] (victim self-witness)", got.Identifying)
	}
```

Both rooms are `&rooms.Room{RoomId: 467}`, which Task 1 established is lit at
visibility 2, and both test mobs carry a zero-value Character, which is awake
and unblinded. So both land in `Identifying` and the assertions hold.

If either test also carries an `IdentifiedPerp` call with a slice, give it
`Witnesses{Identifying: ...}` the same way.

- [ ] **Step 5: Run the package**

Run: `go test ./internal/crimes/ -v 2>&1 | tail -30`

Expected: PASS for every test, including the three new ones from Step 1.

- [ ] **Step 6: Commit**

```bash
git add internal/crimes/crimes.go internal/crimes/crimes_test.go internal/crimes/sight_test.go
git commit -F - <<'EOF'
feat(crimes): witnesses are split by what they could see

A witness was any faction-aligned mob standing in the room. Lighting,
blindness and sleep were never consulted, so a crime committed in a
pitch dark room was witnessed and attributed exactly as one committed
in daylight.

WitnessesInRoom now returns Witnesses, carrying Identifying and
ShapesOnly. IdentifiedPerp answers from Identifying alone, so a room of
witnesses who made out only shapes records the crime against
PerpUnknown, which is the value the package already used for a crime
nobody saw.

The gate reads CanSeeClearly then CanSeeShapes rather than the raw
ParticipantSight. The primitive is optics only and excludes sleep on
purpose; those two compose attention, and ParticipantSight's own
docstring routes non-party observers through them. A crime witness is
exactly that observer, so the sleep gate needed no code of its own.

Changing the return type rather than adding a second function is
deliberate: it makes the compiler enumerate every consumer.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 3: Assault

**Files:**
- Modify: `internal/actions/aggression.go:24-60`

- [ ] **Step 1: Let the compiler show you the site**

Run: `go build ./... 2>&1 | head -20`

Expected: errors at `internal/actions/aggression.go` reporting that
`crimes.Witnesses` cannot be used as `[]int`, and that `len(externalWitnesses)`
is invalid.

- [ ] **Step 2: Rewrite the body**

Replace the body of `RecordAssaultCrime` from `witnesses :=` through the
closing brace of the `for _, fid := range factionIds` loop with:

```go
	// All witnesses including the victim (excludeInstanceId=0) drive
	// perp/rep determination (victim is alive and a self-witness).
	witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
	perp := crimes.IdentifiedPerp(user.UserId, witnesses)
	// External witnesses: same call but exclude the victim - used to
	// set HadExternalWitness so the murder-upgrade path knows whether
	// the assault was seen by someone other than the victim.
	//
	// Identifying, NOT Any() (owner ruling 2026-09-22). This flag's only
	// consumer is the murder upgrade, which asks whether the ORIGINAL
	// ASSAULT WAS IDENTIFIED by someone other than the victim. A bystander
	// that made out only a figure contributes nothing to identification, so
	// counting it would let a creature that never saw a face be treated as
	// able to testify to who was there.
	externalWitnesses := crimes.WitnessesInRoom(factionIds, room, mob.InstanceId)
	hadExternal := len(externalWitnesses.Identifying) > 0
	delta := int(configs.GetBalanceConfig().CrimeRepDeltaAssault)
	for _, fid := range factionIds {
		crimeIds := crimes.Record([]string{fid}, crimes.KindAssault, perp,
			mob, mob.InstanceId, room.RoomId, mob.Character.Zone, hadExternal)
		if perp.Type == crimes.PerpPlayer {
			factions.BumpRep(fid, user.UserId, delta)
			justice.MaybeDeclareBounty(fid, user.UserId, crimes.KindAssault)
			// Knowledge: each witness records the player as the perp of
			// these crimes.
			//
			// Identifying only. Both calls below are keyed on the player
			// subject, so writing them for a shapes-only witness would
			// record that a mob knows who did it when all it saw was a
			// figure. That case is reachable: one clear witness makes perp
			// PerpPlayer for everyone in the room.
			subject := knowledge.PlayerSubject(user.UserId)
			for _, witnessInstId := range witnesses.Identifying {
				w := mobs.GetInstance(witnessInstId)
				if w == nil {
					continue
				}
				for _, crimeId := range crimeIds {
					knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
				}
				knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
					knowledge.SourceWitnessed)
			}
		}
	}
```

- [ ] **Step 3: Build**

Run: `go build ./internal/actions/`

Expected: no output.

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/actions/ 2>&1 | tail -5`

Expected: `ok  github.com/GoMudEngine/GoMud/internal/actions`. If a test fails
because a fixture room now computes as unlit, fix the fixture to a lit room
rather than weakening the gate.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/aggression.go
git commit -F - <<'EOF'
feat(crimes): assault reads the witness split

HadExternalWitness asks whether anybody noticed, so it reads Any().
Knowledge writes ask who can name the player, so they iterate
Identifying.

That second distinction is load-bearing rather than tidy. perp is
computed once for the whole room, so a single clear witness makes it
PerpPlayer for everyone present, and the old loop would then have
recorded a shapes-only witness as knowing exactly who did it.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 4: Theft

**Files:**
- Modify: `internal/actions/steal.go:310-340`

- [ ] **Step 1: Read the site as it stands**

Run: `sed -n '308,342p' internal/actions/steal.go`

Expected: the `if factionIds := factions.FactionsForMob(m); len(factionIds) > 0`
block, structurally identical to assault's but with `actor.GetUserId()` in
place of `user.UserId`, `m` in place of `mob`, and `CrimeRepDeltaTheft`.

- [ ] **Step 2: Apply the same three changes**

Inside that block:

```go
		// All witnesses including the victim (excludeInstanceId=0).
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		perp := crimes.IdentifiedPerp(actor.GetUserId(), witnesses)
		// External witnesses (excluding victim) for HadExternalWitness.
		// Identifying, not Any() (owner ruling 2026-09-22): the flag feeds
		// the murder upgrade's "was the assault identified by someone other
		// than the victim", and a shapes-only bystander identifies nobody.
		externalWitnesses := crimes.WitnessesInRoom(factionIds, room, m.InstanceId)
		hadExternal := len(externalWitnesses.Identifying) > 0
```

and change the knowledge loop's range from `witnesses` to
`witnesses.Identifying`, adding the same reason as a comment:

```go
				// Identifying only: both calls are keyed on the player
				// subject, so a shapes-only witness must not be recorded as
				// knowing who did it.
				subject := knowledge.PlayerSubject(actor.GetUserId())
				for _, witnessInstId := range witnesses.Identifying {
```

- [ ] **Step 3: Build and test**

Run: `go build ./internal/actions/ && go test ./internal/actions/ 2>&1 | tail -5`

Expected: no build output, then `ok  github.com/GoMudEngine/GoMud/internal/actions`.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/steal.go
git commit -F - <<'EOF'
feat(crimes): theft reads the witness split

Same three changes as assault, and all three read Identifying: for
HadExternalWitness, for the perpetrator, and for the knowledge writes.
Stealing in an unlit room is now unattributable unless something in the
room can see in the dark.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 5: Planting

**Files:**
- Modify: `internal/actions/plant.go:220-250`

- [ ] **Step 1: Read the site as it stands**

Run: `sed -n '218,252p' internal/actions/plant.go`

Expected: the same block shape as theft. Note its `WitnessesInRoom` call for
external witnesses is split across two lines.

- [ ] **Step 2: Apply the same three changes**

```go
		// All witnesses including the victim (excludeInstanceId=0).
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		perp := crimes.IdentifiedPerp(actor.GetUserId(), witnesses)
		// External witnesses (excluding victim) for HadExternalWitness.
		// Identifying, not Any() (owner ruling 2026-09-22): a shapes-only
		// bystander identifies nobody, so it cannot make an assault
		// "externally identified" for the murder upgrade.
		externalWitnesses := crimes.WitnessesInRoom(factionIds, room,
			m.InstanceId)
		hadExternal := len(externalWitnesses.Identifying) > 0
```

and:

```go
				// Identifying only: both calls are keyed on the player
				// subject, so a shapes-only witness must not be recorded as
				// knowing who did it.
				subject := knowledge.PlayerSubject(actor.GetUserId())
				for _, witnessInstId := range witnesses.Identifying {
```

- [ ] **Step 3: Build and test**

Run: `go build ./internal/actions/ && go test ./internal/actions/ 2>&1 | tail -5`

Expected: no build output, then `ok  github.com/GoMudEngine/GoMud/internal/actions`.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/plant.go
git commit -F - <<'EOF'
feat(crimes): planting reads the witness split

The third of the three identical action-side sites. Same rule: the
identifying list everywhere, because every consumer here asks who can
name the player rather than who noticed something happen.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 6: Murder, and its four-case upgrade

This site is the subtle one: `currentExternal` feeds a documented four-case
model for upgrading an assault to a murder, and the cases turn on whether the
killing blow was witnessed.

**Files:**
- Modify: `internal/hooks/MobDeath_FactionRep.go:58-72` and its
  `writeKnowledgeForWitnesses` helper

- [ ] **Step 1: Read the whole decision block**

Run: `sed -n '55,140p' internal/hooks/MobDeath_FactionRep.go`

Expected: `witnesses := crimes.WitnessesInRoom(factionIds, room, evt.InstanceId)`,
then `currentExternal := len(witnesses) > 0`, then the Case A/B/C comments.

- [ ] **Step 2: Change the two reads**

```go
	// Witnesses: faction-aligned mob instances in the room, EXCLUDING
	// the victim's instance (victim is dead, not a self-witness).
	witnesses := crimes.WitnessesInRoom(factionIds, room, evt.InstanceId)
```

and

```go
	// Identifying, NOT Any() (owner ruling 2026-09-22). This drives the
	// assault-to-murder upgrade's four-case model, and every case turns on
	// whether the killing was IDENTIFIED by someone other than the victim,
	// not merely sensed. A shapes-only witness identifies nobody, so it
	// must not push the upgrade into Case A.
	currentExternal := len(witnesses.Identifying) > 0
```

`perp := crimes.IdentifiedPerp(userId, witnesses)` needs no edit: it now takes
`Witnesses` and reads `Identifying` itself.

- [ ] **Step 3: Narrow the knowledge helper**

Replace `writeKnowledgeForWitnesses` with:

```go
// writeKnowledgeForWitnesses writes a knowledge record for each witness who
// could identify the perpetrator, linking them to the given crime IDs and the
// player perp. Skips when perp is not a player (e.g. PerpUnknown in Case C).
//
// It takes Witnesses rather than a slice so the Identifying/ShapesOnly choice
// is made here, once, and cannot be got wrong by a caller. Both calls below
// are keyed on the player subject, so a shapes-only witness must not appear:
// it would record a mob as knowing who did it when all it saw was a figure.
func writeKnowledgeForWitnesses(witnesses crimes.Witnesses, perp crimes.Perpetrator,
	crimeIds []int, roomId int) {
	if perp.Type != crimes.PerpPlayer {
		return
	}
	subject := knowledge.PlayerSubject(perp.Id)
	for _, witnessInstId := range witnesses.Identifying {
		w := mobs.GetInstance(witnessInstId)
		if w == nil {
			continue
		}
		for _, crimeId := range crimeIds {
			knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
		}
		knowledge.RecordMet(int(w.MobId), subject, roomId, knowledge.SourceWitnessed)
	}
}
```

Both call sites already pass `witnesses`, so they need no edit.

- [ ] **Step 4: Build and test**

Run: `go build ./internal/hooks/ && go test ./internal/hooks/ 2>&1 | tail -5`

Expected: no build output, then `ok  github.com/GoMudEngine/GoMud/internal/hooks`.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/MobDeath_FactionRep.go
git commit -F - <<'EOF'
feat(crimes): murder reads the witness split

currentExternal takes the identifying list, because every case of the
assault-to-murder upgrade turns on whether the killing was identified by
someone other than the victim, not merely sensed. A witness that made
out a figure identifies nobody and must not push the upgrade into
Case A.

Not a free-reputation exploit in the other direction either:
FindRecentAssault skips any row whose perpetrator is not this player, so
an unattributed assault row is never found and Case C is unreachable for
it. The kill takes the fresh-record path instead. Chased and disproved,
recorded in the plan's facts table so nobody chases it twice.

writeKnowledgeForWitnesses now takes Witnesses rather than a slice, so
the Identifying choice is made once inside the helper where it cannot be
got wrong by a caller.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 7: Witness response, split by RESPONSE and not by list

⚠️ **This site is NOT in the spec.** The compiler found it when
`WitnessesInRoom` changed shape.

🔵 **OWNER RULING 2026-09-22, settled, do not relitigate.** The answer is
neither `All()` nor `Identifying`. **Split by the RESPONSE, not by the list.**

`classifyWitnessResponse` (fact 21) already sorts witnesses three ways, and
that classification answers the question by itself, because the three branches
differ in exactly the way sight cares about (fact 22):

| Classification | Effect today | Names anyone? |
|---|---|---|
| guard, `ResponseReportOnly` | no-op | no |
| noncombatant, `ResponseAlarm` | `alarmReaction(m)`: emote, step toward an exit | **no** |
| anything else, `ResponseRevenge` | `seedRevengeGoalIfAbsent(m, "player", playerId, priority)` | **yes, by player ID** |

So the rule is:

- **Identifying witnesses** go through `seedWitnessResponse` exactly as today.
- **ShapesOnly witnesses** get `alarmReaction` ONLY, whatever they would
  otherwise classify as, because a creature that sensed a scuffle can recoil
  and run but cannot hunt a person it never saw.
- **Guards stay on ReportOnly in BOTH tiers.** A guard reports through the
  crime record; a personal reaction would derail enforcement.

**Files:**
- Modify: `internal/seeders/witness_response.go`
- Modify: `internal/seeders/aggressive_action_to_revenge.go:62-75`
- Test: `internal/seeders/witness_response_test.go`

- [ ] **Step 1: Read both sites**

Run: `cat internal/seeders/witness_response.go`

Expected: `WitnessResponse`, `classifyWitnessResponse`, `seedWitnessResponse`
and the unexported `alarmReaction`.

Run: `sed -n '58,78p' internal/seeders/aggressive_action_to_revenge.go`

Expected: a `for _, witnessInstId := range crimes.WitnessesInRoom(victimFactions, room, attackedMob.InstanceId)` loop that skips `AutoAggro` witnesses and calls `seedWitnessResponse`.

- [ ] **Step 2: Write the failing test**

Create `internal/seeders/witness_response_test.go` (if the file exists, append
these three tests to it):

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestShapesOnlyWitness_NoncombatantGetsAlarmOnly is the baseline: a
// noncombatant behaves the same in both tiers, because its response never
// named anybody to begin with.
func TestShapesOnlyWitness_NoncombatantGetsAlarmOnly(t *testing.T) {
	m := newNoncombatantWitnessForTest(t)
	seedShapesOnlyWitnessResponse(m)

	if hasRevengeGoalForTest(t, m) {
		t.Error("a shapes-only noncombatant seeded a revenge goal; it saw no face to hunt")
	}
	if !issuedAlarmForTest(t, m) {
		t.Error("a shapes-only noncombatant issued no alarm reaction")
	}
}

// TestShapesOnlyWitness_FighterAlsoGetsAlarmOnly is the ruling's actual
// content: a combat-capable mob that would normally take revenge collapses
// into the alarm, because seedRevengeGoalIfAbsent targets the player BY ID
// and this witness has no ID to target.
func TestShapesOnlyWitness_FighterAlsoGetsAlarmOnly(t *testing.T) {
	m := newFighterWitnessForTest(t)
	if got := classifyWitnessResponse(m); got != ResponseRevenge {
		t.Fatalf("fixture misbuilt: classifyWitnessResponse = %v, want ResponseRevenge", got)
	}

	seedShapesOnlyWitnessResponse(m)

	if hasRevengeGoalForTest(t, m) {
		t.Error("a shapes-only fighter seeded a revenge goal against a player it never saw")
	}
	if !issuedAlarmForTest(t, m) {
		t.Error("a shapes-only fighter issued no alarm reaction")
	}
}

// TestIdentifyingWitness_FighterStillTakesRevenge proves the gate did not
// quietly disarm the identifying tier as well.
func TestIdentifyingWitness_FighterStillTakesRevenge(t *testing.T) {
	m := newFighterWitnessForTest(t)
	seedWitnessResponse(m, 17, aggressiveWitnessRevengePriority)

	if !hasRevengeGoalForTest(t, m) {
		t.Error("an identifying fighter seeded no revenge goal; the identifying tier must be unchanged")
	}
}

// TestShapesOnlyWitness_GuardStillReportsOnly pins the third branch: a guard
// is a no-op in both tiers.
func TestShapesOnlyWitness_GuardStillReportsOnly(t *testing.T) {
	m := newGuardWitnessForTest(t)
	seedShapesOnlyWitnessResponse(m)

	if hasRevengeGoalForTest(t, m) {
		t.Error("a shapes-only guard seeded a revenge goal")
	}
	if issuedAlarmForTest(t, m) {
		t.Error("a shapes-only guard issued an alarm; guards report through the crime record instead")
	}
}
```

**Fixtures and assertion helpers.** `alarmReaction` works by issuing
`m.Command(...)`, and `seedRevengeGoalIfAbsent` writes a goal onto the mob, so
both are observable on the mob itself. Before writing the four helpers
(`newNoncombatantWitnessForTest`, `newFighterWitnessForTest`,
`newGuardWitnessForTest`, `hasRevengeGoalForTest`, `issuedAlarmForTest`), read
how the package's existing tests build a mob and inspect its goals:

Run: `ls internal/seeders/*_test.go && grep -rn "mobs.Mob{" internal/seeders/*_test.go | head`

Build the fixtures on whatever pattern is already there. `newGuardWitnessForTest`
must give the mob a group that `mobs.IsGuardMob` recognises, and
`newNoncombatantWitnessForTest` must satisfy `m.IsNonCombatant()`; check both
predicates' sources before choosing the fixture data, rather than guessing at
a group name.

- [ ] **Step 3: Run it and watch it fail to compile**

Run: `go test ./internal/seeders/ -run TestShapesOnlyWitness`

Expected: build failure, `undefined: seedShapesOnlyWitnessResponse`.

- [ ] **Step 4: Add the shapes-tier responder**

In `internal/seeders/witness_response.go`, directly below `seedWitnessResponse`:

```go
// seedShapesOnlyWitnessResponse is seedWitnessResponse for a witness that made
// out shapes but no faces (crimes.Witnesses.ShapesOnly).
//
// Owner ruling 2026-09-22: the tier is decided by the RESPONSE, not by which
// list the witness came from. ResponseRevenge collapses into the alarm here,
// because seedRevengeGoalIfAbsent targets the player BY ID and this witness
// has no identity to target: it can recoil and run, it cannot hunt a person it
// never saw. ResponseAlarm is already identical in both tiers, since
// alarmReaction names nobody. Guards stay a no-op in both tiers, because a
// personal reaction would derail enforcement.
//
// It stays in this file so alarmReaction does not need exporting for a single
// extra caller.
func seedShapesOnlyWitnessResponse(m *mobs.Mob) {
	switch classifyWitnessResponse(m) {
	case ResponseReportOnly:
		// no-op, exactly as in the identifying tier.
	default:
		// ResponseAlarm and ResponseRevenge both land here.
		alarmReaction(m)
	}
}
```

- [ ] **Step 5: Run the tests and watch them pass**

Run: `go test ./internal/seeders/ -run "TestShapesOnlyWitness|TestIdentifyingWitness" -v 2>&1 | tail -20`

Expected: PASS on all four.

- [ ] **Step 6: Split the caller's loop by tier**

In `internal/seeders/aggressive_action_to_revenge.go`, replace the single
witness loop with:

```go
	// Witnesses who SHARE a faction with the victim (the same rule
	// crimes.WitnessesInRoom applies), at witness priority. Skip AutoAggro
	// witnesses - they already attack on sight, so revenge is redundant noise.
	//
	// The two tiers get different responses, not different eligibility (owner
	// ruling 2026-09-22): a witness that made out only shapes can raise the
	// alarm but cannot seed a goal that targets the player by ID.
	witnesses := crimes.WitnessesInRoom(victimFactions, room, attackedMob.InstanceId)
	for _, witnessInstId := range witnesses.Identifying {
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedWitnessResponse(witness, pa.UserId, aggressiveWitnessRevengePriority)
	}
	for _, witnessInstId := range witnesses.ShapesOnly {
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedShapesOnlyWitnessResponse(witness)
	}
```

- [ ] **Step 7: Build and test**

Run: `go build ./internal/seeders/ && go test ./internal/seeders/ 2>&1 | tail -5`

Expected: no build output, then `ok  github.com/GoMudEngine/GoMud/internal/seeders`.

- [ ] **Step 8: Prove the whole tree builds**

Run: `go build ./... 2>&1 | head -20`

Expected: no output. Every consumer of the changed signature is now updated,
which is the compiler confirming the enumeration is complete.

- [ ] **Step 9: Commit**

```bash
git add internal/seeders/witness_response.go internal/seeders/aggressive_action_to_revenge.go internal/seeders/witness_response_test.go
git commit -F - <<'EOF'
feat(crimes): a witness that saw no face raises the alarm, not a hunt

A mob that cannot see the room no longer responds at all, because it is
no longer a witness.

For the shapes tier the owner ruled the split belongs on the RESPONSE
rather than the witness list, and the existing classification already
draws that line: alarmReaction emotes and runs and names nobody, while
seedRevengeGoalIfAbsent targets the player by ID. So a shapes-only
witness gets the alarm whatever it would otherwise classify as, a
combat-capable one included, and guards stay a no-op in both tiers.

This site is not in the M5 spec. The compiler found it when
WitnessesInRoom changed shape, which is why the return type changed
rather than a second function being added.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 8: Full suite, root gate and boot

**Files:** none modified unless something fails.

- [ ] **Step 1: Full suite**

Run: `go test ./... 2>&1 | grep -v "^ok\|no test files" | head -30`

Expected: no output. Anything printed is a real failure; fix it before
continuing rather than deferring it.

- [ ] **Step 2: Root gate**

Run: `go test . 2>&1 | tail -20`

Expected: `ok  github.com/GoMudEngine/GoMud`. The root holds the guard tests,
including the messaging surface guards.

- [ ] **Step 3: Vet and format**

Run: `go vet ./internal/crimes/ ./internal/actions/ ./internal/hooks/ ./internal/seeders/`

Expected: no output.

Run: `gofmt -l internal/crimes internal/actions internal/hooks internal/seeders`

Expected: no output. If a file is listed, check it is a real formatting issue
and not the Windows CRLF false positive: `git show HEAD:<file> | head -c 16 | od -c`
shows the committed blob's line endings.

- [ ] **Step 4: Boot check in an isolated detached worktree**

Follow `dogmud-shipping`'s detached-worktree boot procedure.

🪤 A failed boot EXITS 0. `main()` recovers, logs `PANIC` and a stack, then
returns normally. Grep the log for `Server Ready` and for a real panic line;
never check `$?`.

🪤 Two `PANIC` hits in a boot log are the documented false positive:
`GamePlay.MapConsistencyEnforce` has the literal value `panic`. Three or more,
or one with a stack trace, is real.

- [ ] **Step 5: Commit nothing, or commit the fix**

If steps 1 to 4 were clean there is nothing to commit. If a fix was needed,
commit it on its own with a message naming what the gate caught.

---

## Task 9: The adversarial playtest gate

This is where the stage ends, per the project content SOP. It is not optional
and it is not satisfied by a green suite.

**Files:**
- Create: an ephemeral goals file (see `dogmud-playtesting`)

- [ ] **Step 1: Write the goals file**

It must exercise all three tiers and must include the case no previous
playtest in this project has ever run: a crime committed in an unlit room.

1. **Clear sight.** Commit an assault in a lit city room with faction guards
   present. Expect: identified, rep drops, bounty logic runs as today.
2. **No sight.** Commit the same assault in an unlit room whose mobs are
   ordinary humans. Expect: the crime is recorded, perp is unknown, no rep
   change, no bounty, no mob knows your name.
3. **Shapes only.** Same unlit room, but an admin first applies condition 85
   to one witness mob. Expect: crime recorded, perp still unknown, and that
   mob still seeds revenge.
4. **Night vision.** Same unlit room with a mob that carries condition 29
   (any canine, serpent, raptor, feline, arachnid, mustelid, goblin or troll).
   Expect: fully identified, exactly as the lit case.
5. **The sleep gate.** A LIT room whose only faction mob is asleep. Expect:
   not a witness, perp unknown. This is the case that proves attention is
   being read and not just lighting.

- [ ] **Step 2: Note the two traps before running**

🪤 The shapes tier is unreachable by shipped content: 0 species, 0 mobs and 0
items grant InfraredVision. Scenario 3 only works by an admin applying
condition 85 by hand, the same way infrared was finally verified on
2026-09-21.

🪤 `setcondition` is ADMIN ONLY. The playtest character must be **Megalomania**
(`role: admin`), not Meirok (`role: user`).

- [ ] **Step 3: Run it**

Per `dogmud-playtesting`. Local runs need an ephemeral goals file and a
`--checkout`.

🪤 `playtestrun stop` exits 0 and does nothing. Tear the container down with
`docker rm -f`.

- [ ] **Step 4: Confirm the accepted play consequence actually holds**

74 of 641 shipped mobs carry NightVision and they are overwhelmingly wildlife,
so the prediction is that an unlit CITY room is a low-risk crime scene while an
unlit forest stays patrolled. Verify that is what happens in play, and record
the result either way. If unlit city rooms turn out to be rarer than the
analysis assumed, say so: it changes how much this ruling actually affects
players.

- [ ] **Step 5: Extract findings to memory**

Playtest reports are gitignored, so anything learned must be written to a
memory file or it is lost.

- [ ] **Step 6: Commit the goals file if it is worth keeping**

---

## Task 10: Documentation

**Files:**
- Modify: `internal/crimes/context.md`
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Update the package context**

`internal/crimes/context.md` must describe `Witnesses`, `All()`, `Any()`, the
changed `WitnessesInRoom` and `IdentifiedPerp` signatures, and the rule that
knowledge writes use `Identifying` only.

Every symbol named must exist. Verify with:

```
Select-String -Path internal\crimes\*.go -Pattern '^(func|type|const|var)\s'
```

- [ ] **Step 2: Run the context audit**

Run: `python tools/context_md_audit.py internal/crimes`

Expected: `All documented symbols resolve.`

- [ ] **Step 3: Write the patch note**

Player framing, no raw numbers, no dashes, wrap at 80. It should say that
crimes committed where nobody can see are harder to pin on you, and that
creatures which see in the dark are not fooled. Do not enumerate which species
those are: let players find out.

- [ ] **Step 4: Commit**

```bash
git add internal/crimes/context.md docs/PATCH_NOTES.md
git commit -F - <<'EOF'
docs(crimes): record the witness split and its player-facing effect

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Proof obligations

1. **The tier test must be capable of failing.** Before trusting
   `TestWitnessesInRoom_SplitsBySight`, sabotage it: make the gate classify
   everything as `Identifying` and confirm the ShapesOnly assertion goes red.
   Restore. A gate that cannot fail is not a gate, and this project has been
   bitten by that three times in the M4 arc alone.
2. **The sleep case must fail for the right reason.** Confirm
   `TestWitnessesInRoom_SleeperInLitRoomIsNotAWitness` is green because of
   `awake()` and not because the room computed as dark: assert the room's
   visibility in the test, which Task 1's test already does.
3. **The compiler must confirm the call-site enumeration.** `go build ./...`
   passing after Task 7 is the evidence that all five sites were found. Do not
   grep for them by hand and call it complete.
4. **The boot check must grep for `Server Ready`**, never `$?`.
5. **The playtest must include an unlit-room crime.** If it does not, the stage
   has not met its gate, whatever the suite says.

---

## Risks

1. **This PR changes faction and bounty consequences, which touch saved
   state.** Crime rows, rep values and bounties all persist. A crime that would
   have been attributed yesterday may be recorded as `PerpUnknown` today. There
   is no migration and none is wanted: existing rows keep whatever they were
   written with.
2. **The blast radius is gated on lighting**, so a world where the relevant
   rooms are lit sees no change at all. That is most of the world, which makes
   this both low risk and easy to under-test. Scenario 2 of the playtest exists
   precisely because the suite cannot tell you how often unlit rooms with
   faction mobs actually occur.
3. **The shapes tier is dead content.** Nothing in the shipped world grants
   InfraredVision, so the `ShapesOnly` branch ships correct and unexercised
   outside tests and an admin command. The owner accepted this. It is a real
   risk only in that a future bug there would be invisible until something
   grants condition 85.
4. **`currentExternal` is now `len(witnesses.Identifying) > 0`, which is
   strictly narrower than today's `len(witnesses) > 0`.** If the gate wrongly
   excludes a witness, or wrongly files one as ShapesOnly, the murder upgrade
   silently takes Case C and REFUNDS the assault rep. That is a quiet,
   player-favouring failure, which is the kind that goes unreported. Task 8's
   full suite plus playtest scenario 1 are what stand between that and
   production.

   🔑 The mirror-image worry, that an unattributed assault could be farmed for
   free rep through Case C, was chased and DISPROVED (fact 24). Do not re-raise
   it: `FindRecentAssault` never matches a row whose perpetrator is not this
   player, so Case C cannot be reached for an unattributed assault at all.
5. **Two existing crimes tests were written against a world with no lighting at
   all.** Task 1 gives that binary real data, which means those tests now
   exercise a code path they never did. If either starts failing for a reason
   unrelated to this change, that is a pre-existing bug surfacing, not
   collateral damage. Investigate rather than pin.

---

## Settled by the owner, 2026-09-22

All three questions this plan opened have been answered. They are recorded here
as rulings and implemented in the tasks above; do not reopen them.

1. **Witness response is split by RESPONSE, not by list.** Task 7. Neither
   `All()` nor `Identifying`: identifying witnesses take the existing
   `seedWitnessResponse`, shapes-only witnesses get `alarmReaction` alone
   whatever they classify as, and guards stay a no-op in both tiers.
2. **`HadExternalWitness` takes the identifying list, minus the victim.**
   Tasks 3, 4, 5, and `currentExternal` in Task 6. Its only consumer asks
   whether the assault was identified by someone other than the victim, and a
   shapes-only bystander identifies nobody.
3. **No new knowledge `Subject` is needed, and none is built.** See the note
   below: the substrate a future system would read already ships.

### The crime-heat substrate ships with this PR; the system that reads it does not

The owner wants "a place with frequent crimes goes on high alert and hires more
guards." That is **future work and explicitly not M5.**

It needs nothing added here. `crimes.Record` already fires unconditionally,
outside the `perp.Type == PerpPlayer` guard, and already stores `RoomId` and
`Zone` (fact 25). So a shapes-only crime already lands in the faction crime log
as a located, unattributed row, which is exactly the shape a heat system would
count. This PR makes that row *more* common rather than less, since crimes in
the dark now record as `PerpUnknown` instead of being fully attributed.

**Build none of it here.** The note exists so that whoever picks the feature up
knows the data is already being written and does not add a parallel log.

### Filed, not built: a sleeping bystander never wakes to a nearby brawl

The owner raised a noise-based wake system as a separate future sidequest, and
it interacts with this PR directly enough to record.

Condition 15 wakes on `cancel-on-action`, `cancel-on-combat` and
`cancel-on-damage` only (fact 26). There is no proximity or noise trigger, so a
sleeper wakes when violence lands **on it**, never because a fight is happening
next to it.

After this PR, therefore, a sleeping bystander witnesses nothing no matter how
long a brawl runs in its room. That is correct under the sleep gate and it is
what the owner asked for. It is also precisely the behaviour a noise-based wake
system would change, and **this PR is where that behaviour is decided**, so the
next person to touch it should start here rather than assuming the sleep gate
was an oversight.

Build none of it in M5.
