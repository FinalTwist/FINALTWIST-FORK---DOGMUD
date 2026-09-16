# Messaging M3 item 8, PR 2: combat-messages onto the narration core

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render every audience of one swing from a single coordinated index,
delete the superseded `ConsistentAttackMessages` apparatus, and give the attack
store the load-time validator it has never had.

**Architecture:** The store gains `PoolFor` (the tier union as plain strings)
and one `Render` method per split, adapting to `narration.Render` exactly as
the defence sibling does. The two combat call sites collapse from three or four
independent draws into one call. `Validate()` delegates to
`narration.ValidateVariants` per (intensity, split, tier) group.

**Tech Stack:** Go, `internal/items`, `internal/combat`, `internal/narration`.

Spec: `docs/superpowers/specs/2026-09-16-messaging-m3-item8-combat-messages-design.md`
Prior PR: `docs/superpowers/plans/2026-09-16-messaging-m3-item8-content-pad.md`

---

## Read this before touching anything

### Do not start until PR 1 has merged

The validator in Task 3 fails the boot on any group whose role pools differ in
length. Before PR 1 that is 440 of 534 groups. Confirm with
`python tools/combat_message_pool_audit.py`, which must print
`OK: every role pool is equal per tier (20 file(s))` and exit 0.

### The role mapping is the mistake this refactor makes easiest

An attacker **acts** and a defender is **acted upon**, so `toattacker` is the
Actor and `todefender` is the Actee. Three separately named pools become
adjacent fields of one struct literal differing only by role name, which is
precisely how the defence store nearly shipped an inversion
(`internal/items/defensive_messages.go:127-142`).

| Split | Actor | Actee | Observer | ActeeObserver |
|---|---|---|---|---|
| `together` | `ToAttacker` | `ToDefender` | `ToRoom` | *(absent)* |
| `separate` | `ToAttacker` | `ToDefender` | `ToAttackerRoom` | `ToDefenderRoom` |

`together` leaving `ActeeObserver` absent is correct, not missing: when the two
share a room there is only one observer audience. The validator is told so
explicitly so a genuinely lost pool is still caught.

The golden keys rows by the **authored** role name, which is what makes a swap
visible. Do not re-key it to the core's vocabulary.

### The draw count changes, deliberately

Today each role takes its own `util.Rand` draw: three for `together`, four for
`separate`. After this PR there is one draw per narrated message. That shifts
the global random stream, so combat-determinism tests and playtest outcomes
will move. This is correct and expected. Do not debug it as a regression.

### Tripwires

- `_datafiles/config.yaml` carries the git skip-worktree bit and desyncs in
  both directions. Build its one-line change from the `git show HEAD:` blob,
  never from disk.
- Never `git add -A` or `git add .`. Named paths only.
- `grep` exits 1 on zero matches, so any "expect zero" check runs standalone.
- Every sabotage probe must be confirmed to compile **and** to turn the test
  red before the green run means anything.
- Combat tests carry a roughly 2.3% attack-fumble flake. Re-run a single
  combat failure alone before treating it as real.
- No em dashes or en dashes.

---

## File structure

**Modify (production):**
- `internal/items/attack_messages.go`: add `PoolFor` and two `Render` methods, rewrite `Validate`, delete the seeded apparatus
- `internal/combat/combat.go:227-354`: `GetWaitMessages` onto `Render`
- `internal/combat/combat_helpers.go:1538-1815`: `buildAttackMessages` onto `Render`
- `internal/configs/config.balance.go:304`: drop the field
- `internal/configs/config.balance.combat.go:340`: drop the comment
- `internal/configs/config.gameplay.go:59`: drop the ignore line
- `_datafiles/config.yaml:853`: drop the key

**Modify (proof):**
- `internal/narration/snapshot_test.go`: re-key the combat builder to coordinated rows
- `internal/narration/testdata/stores/combat_messages.golden`: re-recorded
- `internal/items/attack_messages_test.go`: new unit and sabotage tests
- a root guard file for the anti-backslide check

**Modify (content and docs):**
- `_datafiles/world/default/combat-messages/*.yaml`: comment banners, 8 files
- `internal/items/context.md`, `internal/narration/context.md`, `internal/configs/context.md`
- `docs/README.md`, `docs/superpowers/audits/messaging-m6-content-ledger.md`

---

### Task 0: Confirm the ground state

- [ ] **Step 1: PR 1 is merged and the store is equal**

Run: `python tools/combat_message_pool_audit.py`

Expected: `OK: every role pool is equal per tier (20 file(s))`, exit 0.

- [ ] **Step 2: The golden is at its PR 1 size**

Run: `grep -c " => " internal/narration/testdata/stores/combat_messages.golden`

Expected: `6978`.

- [ ] **Step 3: Baseline is green**

Run: `go test ./internal/items/... ./internal/combat/... ./internal/narration/...`

Expected: PASS.

---

### Task 1: `PoolFor` and the two `Render` methods

**Files:**
- Modify: `internal/items/attack_messages.go`
- Test: `internal/items/attack_messages_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/items/attack_messages_test.go`:

```go
func TestPoolForUnionsTiersCumulatively(t *testing.T) {
	stm := SkillTieredMessages{
		Beginner: MessageOptions{"b1", "b2"},
		Expert:   MessageOptions{"e1"},
		Master:   MessageOptions{"m1"},
	}
	cases := []struct {
		skill int
		want  []string
	}{
		{10, []string{"b1", "b2"}},
		{34, []string{"b1", "b2", "e1"}},
		{66, []string{"b1", "b2", "e1"}},
		{67, []string{"b1", "b2", "e1", "m1"}},
		{100, []string{"b1", "b2", "e1", "m1"}},
	}
	for _, c := range cases {
		got := stm.PoolFor(c.skill)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("PoolFor(%d) = %v, want %v", c.skill, got, c.want)
		}
	}
}

func TestTogetherRenderCoordinatesOneIndexAcrossRoles(t *testing.T) {
	m := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a0", "a1", "a2"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d0", "d1", "d2"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r0", "r1", "r2"}},
	}
	for idx := 0; idx < 3; idx++ {
		pick := func(n int) int { return idx }
		roles := m.Render(10, nil, pick)
		wantA := fmt.Sprintf("a%d", idx)
		wantD := fmt.Sprintf("d%d", idx)
		wantR := fmt.Sprintf("r%d", idx)
		if roles.Actor != wantA || roles.Actee != wantD || roles.Observer != wantR {
			t.Errorf("index %d: got (%q,%q,%q), want (%q,%q,%q); all roles must come from ONE index",
				idx, roles.Actor, roles.Actee, roles.Observer, wantA, wantD, wantR)
		}
		if roles.ActeeObserver != "" {
			t.Errorf("index %d: together must leave ActeeObserver empty, got %q", idx, roles.ActeeObserver)
		}
	}
}

func TestSeparateRenderMapsFourRoles(t *testing.T) {
	m := SeparateMessages{
		ToAttacker:     SkillTieredMessages{Beginner: MessageOptions{"atk"}},
		ToDefender:     SkillTieredMessages{Beginner: MessageOptions{"def"}},
		ToAttackerRoom: SkillTieredMessages{Beginner: MessageOptions{"atkroom"}},
		ToDefenderRoom: SkillTieredMessages{Beginner: MessageOptions{"defroom"}},
	}
	roles := m.Render(10, nil, FirstPickerForTest)
	if roles.Actor != "atk" || roles.Actee != "def" ||
		roles.Observer != "atkroom" || roles.ActeeObserver != "defroom" {
		t.Errorf("separate role mapping wrong: %+v", roles)
	}
}
```

Add at the top of the test file if not already present:

```go
// FirstPickerForTest always selects index 0.
func FirstPickerForTest(n int) int { return 0 }
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/... -run 'TestPoolFor|TestTogetherRender|TestSeparateRender' -v`

Expected: compile failure, `stm.PoolFor undefined` and `m.Render undefined`.

- [ ] **Step 3: Implement**

Add to `internal/items/attack_messages.go`:

```go
// PoolFor returns the tier union for a skill level as the core's plain-string
// form. The union is cumulative and matches what GetForSkillLevelWith built:
// beginner always, plus expert at 34, plus master at 67.
//
// Assembly stays in the store. The core coordinates the index and substitutes
// tokens; it knows nothing about tiers (internal/narration/context.md).
func (stm SkillTieredMessages) PoolFor(skillLevel int) []string {
	out := make([]string, 0, len(stm.Beginner)+len(stm.Expert)+len(stm.Master))
	for _, m := range stm.Beginner {
		out = append(out, string(m))
	}
	if skillLevel >= 34 {
		for _, m := range stm.Expert {
			out = append(out, string(m))
		}
	}
	if skillLevel >= 67 {
		for _, m := range stm.Master {
			out = append(out, string(m))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Render renders one coordinated attacker/defender/room triad for a swing the
// two participants share a room for.
//
// ALL ROLES COME FROM ONE VARIANT INDEX. The authored pools pair up by index,
// so picking per role narrates three different events to three audiences,
// which is the defect this migration exists to remove and which shipped twice
// before (melee defence PR #112, taunt PR #115).
//
// Role mapping is the one thing here worth reading slowly: an attacker ACTS
// and a defender is ACTED UPON, so toattacker is the Actor and todefender is
// the Actee. Swapping those two lines inverts every combat message in the
// game. combat_messages.golden keys its rows by the AUTHORED name, which is
// what catches it.
//
// ActeeObserver is deliberately left empty: when the participants share a
// room there is only one observer audience.
//
// A nil picker means production behaviour (narration.DefaultPicker).
func (m TogetherMessages) Render(skillLevel int, tokenReplacements map[TokenName]string, pick narration.Picker) narration.Roles {
	return narration.Render(
		narration.Variants{
			Actor:    m.ToAttacker.PoolFor(skillLevel),
			Actee:    m.ToDefender.PoolFor(skillLevel),
			Observer: m.ToRoom.PoolFor(skillLevel),
		},
		tokenStrings(tokenReplacements),
		pick,
	)
}

// Render renders one coordinated quartet for a ranged swing where attacker and
// defender are in different rooms, so there are genuinely two observer
// audiences. ActeeObserver is the observers where the DEFENDER is; this is the
// case narration.Roles grew its fourth field for
// (internal/narration/render.go:35-38).
//
// A nil picker means production behaviour (narration.DefaultPicker).
func (m SeparateMessages) Render(skillLevel int, tokenReplacements map[TokenName]string, pick narration.Picker) narration.Roles {
	return narration.Render(
		narration.Variants{
			Actor:         m.ToAttacker.PoolFor(skillLevel),
			Actee:         m.ToDefender.PoolFor(skillLevel),
			Observer:      m.ToAttackerRoom.PoolFor(skillLevel),
			ActeeObserver: m.ToDefenderRoom.PoolFor(skillLevel),
		},
		tokenStrings(tokenReplacements),
		pick,
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/items/... -run 'TestPoolFor|TestTogetherRender|TestSeparateRender' -v`

Expected: PASS, all three.

- [ ] **Step 5: Sabotage the coordination probe and confirm it goes red**

A coordination test that cannot fail is worthless. Temporarily change
`TogetherMessages.Render` to give the observer its own draw:

```go
			Observer: m.ToRoom.PoolFor(skillLevel)[:1],
```

Run: `go test ./internal/items/... -run TestTogetherRenderCoordinates -v`

Expected: FAIL. Because the pools then differ in length, `Variants.Len()`
returns 0 and every role renders empty, so the test reports
`got ("","",""), want ("a0","d0","r0")`.

Revert the sabotage and confirm PASS again.

- [ ] **Step 6: Commit**

```bash
git add internal/items/attack_messages.go internal/items/attack_messages_test.go
git commit -m "feat(items): coordinated renderer for the combat-message store

PoolFor exposes the cumulative tier union as plain strings; TogetherMessages
and SeparateMessages adapt to narration.Render so every audience of one swing
comes from a single index. Separate maps its four authored roles onto the
core's fourth role, which render.go grew for exactly this case.

Coordination probe verified red against a per-role sabotage.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: The draw-count probe

**Files:**
- Test: `internal/items/attack_messages_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRenderTakesExactlyOneDrawPerMessage(t *testing.T) {
	draws := 0
	counting := func(n int) int { draws++; return 0 }

	together := TogetherMessages{
		ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a0", "a1"}},
		ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d0", "d1"}},
		ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r0", "r1"}},
	}
	together.Render(10, nil, counting)
	if draws != 1 {
		t.Errorf("together took %d draws, want exactly 1; per-role draws are the defect this migration removes", draws)
	}

	draws = 0
	separate := SeparateMessages{
		ToAttacker:     SkillTieredMessages{Beginner: MessageOptions{"a0", "a1"}},
		ToDefender:     SkillTieredMessages{Beginner: MessageOptions{"d0", "d1"}},
		ToAttackerRoom: SkillTieredMessages{Beginner: MessageOptions{"ar0", "ar1"}},
		ToDefenderRoom: SkillTieredMessages{Beginner: MessageOptions{"dr0", "dr1"}},
	}
	separate.Render(10, nil, counting)
	if draws != 1 {
		t.Errorf("separate took %d draws, want exactly 1", draws)
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/items/... -run TestRenderTakesExactlyOneDraw -v`

Expected: PASS. It passes immediately because Task 1 already made it true.

- [ ] **Step 3: Prove it can fail**

Temporarily add a second `pick` call inside `TogetherMessages.Render`, before
the `return`:

```go
	_ = pick(2)
```

Run: `go test ./internal/items/... -run TestRenderTakesExactlyOneDraw -v`

Expected: FAIL with `together took 2 draws, want exactly 1`.

Remove the sabotage and confirm PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/items/attack_messages_test.go
git commit -m "test(items): pin combat narration to one random draw per message

Was three draws for together and four for separate, one per audience. The
count is load-bearing: it is both the coordination guarantee and the thing
that shifts the global random stream. Verified red against an extra draw.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: The validator

**Files:**
- Modify: `internal/items/attack_messages.go:153-164`
- Test: `internal/items/attack_messages_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestValidateRejectsUnequalRolePools(t *testing.T) {
	g := &WeaponAttackMessageGroup{
		OptionId: "testweapon",
		Options:  AttackTypes{},
	}
	for _, i := range []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical, Fumble} {
		g.Options[i] = AttackOptions{Together: TogetherMessages{
			ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a"}},
			ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d"}},
			ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r"}},
		}}
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("balanced group should validate, got %v", err)
	}

	// One role one line short is the exact shape the pad fixed.
	short := g.Options[Critical]
	short.Together.ToRoom = SkillTieredMessages{Beginner: MessageOptions{}}
	g.Options[Critical] = short
	if err := g.Validate(); err == nil {
		t.Fatal("Validate accepted a group with a missing toroom pool")
	}
}

func TestValidateRejectsBlankVariant(t *testing.T) {
	g := &WeaponAttackMessageGroup{OptionId: "testweapon", Options: AttackTypes{}}
	for _, i := range []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical, Fumble} {
		g.Options[i] = AttackOptions{Together: TogetherMessages{
			ToAttacker: SkillTieredMessages{Beginner: MessageOptions{"a"}},
			ToDefender: SkillTieredMessages{Beginner: MessageOptions{"d"}},
			ToRoom:     SkillTieredMessages{Beginner: MessageOptions{"r"}},
		}}
	}
	blank := g.Options[Weak]
	blank.Together.ToDefender = SkillTieredMessages{Beginner: MessageOptions{"   "}}
	g.Options[Weak] = blank
	if err := g.Validate(); err == nil {
		t.Fatal("Validate accepted a blank message variant")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/... -run TestValidateRejects -v`

Expected: both FAIL. Today's `Validate` checks intensity presence only, so it
accepts the missing pool and the blank line.

- [ ] **Step 3: Implement**

Replace `Validate` in `internal/items/attack_messages.go` with:

```go
// Validate checks that every intensity is present and that every authored
// group can be rendered from ONE coordinated index.
//
// Equality is checked PER TIER, not on the union totals. The runtime union is
// cumulative (beginner, plus expert at 34, plus master at 67), so equal totals
// with unequal tiers would still pair an expert line against a master one at
// the same index. Per-tier equality is exactly equivalent to union equality at
// all three skill levels.
//
// minVariants is 1, not the defence store's 5: the smallest authored group in
// the shipped store is generic/coupdegrace/separate/beginner at one line, and
// 414 of 534 groups hold fewer than five.
func (w *WeaponAttackMessageGroup) Validate() error {

	// Make sure all important options are present.
	optionsToCheck := []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical, Fumble}
	for _, option := range optionsToCheck {
		if _, ok := w.Options[option]; !ok {
			return fmt.Errorf("missing option[`%s`] for %s", option, w.OptionId)
		}
	}

	tiers := []struct {
		name string
		get  func(SkillTieredMessages) MessageOptions
	}{
		{"beginner", func(s SkillTieredMessages) MessageOptions { return s.Beginner }},
		{"expert", func(s SkillTieredMessages) MessageOptions { return s.Expert }},
		{"master", func(s SkillTieredMessages) MessageOptions { return s.Master }},
	}

	// Sorted so a file with several faults always reports the same one first.
	intensities := make([]string, 0, len(w.Options))
	for intensity := range w.Options {
		intensities = append(intensities, string(intensity))
	}
	sort.Strings(intensities)

	for _, name := range intensities {
		intensity := Intensity(name)
		opts := w.Options[intensity]
		for _, tier := range tiers {
			together := narration.Variants{
				Actor:    messageStrings(tier.get(opts.Together.ToAttacker)),
				Actee:    messageStrings(tier.get(opts.Together.ToDefender)),
				Observer: messageStrings(tier.get(opts.Together.ToRoom)),
			}
			if together.Len() > 0 || anyPool(together) {
				if err := narration.ValidateVariants(together, 1,
					narration.RoleActor, narration.RoleActee, narration.RoleObserver); err != nil {
					return fmt.Errorf("%s option[`%s`].together.%s: %w", w.OptionId, intensity, tier.name, err)
				}
			}

			separate := narration.Variants{
				Actor:         messageStrings(tier.get(opts.Separate.ToAttacker)),
				Actee:         messageStrings(tier.get(opts.Separate.ToDefender)),
				Observer:      messageStrings(tier.get(opts.Separate.ToAttackerRoom)),
				ActeeObserver: messageStrings(tier.get(opts.Separate.ToDefenderRoom)),
			}
			if anyPool(separate) {
				if err := narration.ValidateVariants(separate, 1,
					narration.RoleActor, narration.RoleActee,
					narration.RoleObserver, narration.RoleActeeObserver); err != nil {
					return fmt.Errorf("%s option[`%s`].separate.%s: %w", w.OptionId, intensity, tier.name, err)
				}
			}
		}
	}

	return nil
}

// anyPool reports whether a group was authored at all, so a subtype with no
// separate block is skipped rather than reported as four missing roles.
func anyPool(v narration.Variants) bool {
	return len(v.Actor) > 0 || len(v.Actee) > 0 ||
		len(v.Observer) > 0 || len(v.ActeeObserver) > 0
}
```

Add `"sort"` to the file's imports.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/items/... -run TestValidateRejects -v`

Expected: both PASS.

- [ ] **Step 5: Verify the real store still loads**

Run: `go test ./internal/items/...`

Expected: PASS. If a real weapon file fails here, PR 1's pad is incomplete for
that group; fix the content, do not relax the validator.

- [ ] **Step 6: Prove the validator fails on the pre-pad shape**

A validator that cannot fail is not a validator, and this one must be shown to
reject exactly what the pad fixed.

Temporarily delete the last line of `toroom`'s `beginner` block in
`_datafiles/world/dogmud/combat-messages/throttle.yaml`.

Run: `go test ./internal/items/...`

Expected: FAIL, naming `throttle`, the intensity, `together` and `beginner`,
with the sibling-length message from `ValidateVariants`.

Restore with `git checkout -- _datafiles/world/dogmud/combat-messages/throttle.yaml`,
then check `git status`: that command **stages** the revert, so unstage if
needed. Re-run and confirm PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/items/attack_messages.go internal/items/attack_messages_test.go
git commit -m "feat(items): real load-time validation for the combat-message store

Validate checked intensity presence and nothing else: not pool emptiness,
not blank variants, not role-pool equality, which is what the coordinated
index requires. The defence sibling has had all three for as long as it has
existed.

Delegates to narration.ValidateVariants per (intensity, split, tier) with
the expected roles named, so a together block's deliberately absent
ActeeObserver stays distinguishable from a toroom pool that went missing.
minVariants is 1, not defence's 5: 414 of 534 shipped groups hold fewer.

Verified red by removing one authored line from throttle.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Migrate `GetWaitMessages`

**Files:**
- Modify: `internal/combat/combat.go:227-354`

- [ ] **Step 1: Replace the selection block**

Replace `internal/combat/combat.go:268-279` (the `if sourceChar.RoomId ==
targetChar.RoomId` selection, four `GetForSkillLevel` calls per branch) with:

```go
	var roles narration.Roles
	if sourceChar.RoomId == targetChar.RoomId {
		roles = msgs.Together.Render(skillLevel, tokenReplacements, nil)
	} else {
		roles = msgs.Separate.Render(skillLevel, tokenReplacements, nil)
	}
	toAttackerMsg = items.ItemMessage(roles.Actor)
	toDefenderMsg = items.ItemMessage(roles.Actee)
	toAttackerRoomMsg = items.ItemMessage(roles.Observer)
	toDefenderRoomMsg = items.ItemMessage(roles.ActeeObserver)
```

`Render` substitutes tokens itself, so `tokenReplacements` moves from being
applied afterwards to being passed in. Delete any subsequent
`SetTokenValue` loop over these four messages, but **keep** the
`{exitname}`/`{entrancename}` resolution at lines 282-300: those two tokens are
computed from the two rooms and must still be applied to the rendered text.

- [ ] **Step 2: Delete the now-unused seed**

Delete `internal/combat/combat.go:236-239`:

```go
	// zero means randomly selected, otherwise use the ItemId to consistently choose a message
	msgSeed := 0
	if configs.GetBalanceConfig().ConsistentAttackMessages {
		msgSeed = sourceChar.Equipment.Weapon.ItemId
	}
```

Remove the `configs` import if nothing else in the file uses it.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./internal/combat/...`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/combat/combat.go
git commit -m "refactor(combat): GetWaitMessages renders one coordinated event

Was three or four independent GetForSkillLevel draws, one per audience, so
the prepare and wait lines narrated different moments to the attacker, the
defender and each room. Now one Render call per branch.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Migrate `buildAttackMessages`

**Files:**
- Modify: `internal/combat/combat_helpers.go:1538-1815`

- [ ] **Step 1: Replace the selection block**

Replace `internal/combat/combat_helpers.go:1678-1687` with the same shape as
Task 4, reading the skill level and tokens already in scope in that function:

```go
	} else if sourceChar.RoomId == targetChar.RoomId {
		roles := msgs.Together.Render(skillLevel, tokenReplacements, nil)
		toAttackerMsg = items.ItemMessage(roles.Actor)
		toDefenderMsg = items.ItemMessage(roles.Actee)
		toAttackerRoomMsg = items.ItemMessage(roles.Observer)
		toDefenderRoomMsg = items.ItemMessage(roles.ActeeObserver)
	} else {
		roles := msgs.Separate.Render(skillLevel, tokenReplacements, nil)
		toAttackerMsg = items.ItemMessage(roles.Actor)
		toDefenderMsg = items.ItemMessage(roles.Actee)
		toAttackerRoomMsg = items.ItemMessage(roles.Observer)
		toDefenderRoomMsg = items.ItemMessage(roles.ActeeObserver)
	}
```

Keep the branch above this one (whatever `} else if` it chains from) and keep
the `{exitname}`/`{entrancename}` resolution at lines 1690-1706.

- [ ] **Step 2: Delete the seed from the swing params**

Delete `internal/combat/combat_helpers.go:498-501` (the `msgSeed` assignment
and its knob check), the `msgSeed: msgSeed,` entry at line 507, and the
`msgSeed int` field at line 88.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./internal/combat/...`

Expected: PASS. Re-run any single combat failure alone before believing it,
because of the fumble flake.

- [ ] **Step 4: Commit**

```bash
git add internal/combat/combat_helpers.go
git commit -m "refactor(combat): buildAttackMessages renders one coordinated event

The per-swing hit and miss narration took an independent draw per audience,
so one sword blow could be a laceration to the attacker, a devastating hit
to the defender and a critical strike to the room. Now one Render call.

Also drops msgSeed from the swing params: nothing seeds it any more.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Delete the knob and the seeded apparatus

**Files:**
- Modify: `internal/items/attack_messages.go`
- Modify: `internal/configs/config.balance.go:304`, `config.balance.combat.go:340`, `config.gameplay.go:59`
- Modify: `_datafiles/config.yaml:853`
- Test: a root guard file

- [ ] **Step 1: Delete the seeded branches in the store**

In `internal/items/attack_messages.go`:

Replace `GetWith` (lines 65-90) with a seedless version, and delete
`GetForSkillLevel` and `GetForSkillLevelWith` (lines 92-145) outright, since
Task 4 and Task 5 removed their last callers:

```go
// GetWith chooses one message from a flat pool. A nil picker means production
// behaviour: narration.DefaultPicker, i.e. util.Rand.
func (mo MessageOptions) GetWith(pick narration.Picker) ItemMessage {
	if pick == nil {
		pick = narration.DefaultPicker
	}
	if ct := len(mo); ct > 0 {
		return mo[pick(ct)]
	}
	return ItemMessage("")
}
```

This removes the unreachable `if seedNum[0] == 0 { return mo[0] }`, which
line 78's guard made dead on every input.

Update `Get` to match:

```go
// Get chooses a message using the default picker.
func (mo MessageOptions) Get() ItemMessage {
	return mo.GetWith(nil)
}
```

- [ ] **Step 2: Let the compiler find the rest**

Run: `go build ./...`

Fix each reported call site by dropping the seed argument. Do not add a
compatibility shim: deleting the field first and letting the compiler
enumerate consumers is the point (`dogmud-refactoring`).

- [ ] **Step 3: Delete the config field and its yaml key**

Delete `internal/configs/config.balance.go:304`, the comment at
`config.balance.combat.go:340`, and the ignore line at
`config.gameplay.go:59`.

For `_datafiles/config.yaml`, which carries skip-worktree, build the change
from the committed blob rather than from disk:

```bash
git show HEAD:_datafiles/config.yaml > "$SCRATCH/config.yaml"
```

where `$SCRATCH` is this session's scratchpad directory, outside the repo.

Remove the single `  ConsistentAttackMessages: false` line from that copy with
the Edit tool, then write it back over `_datafiles/config.yaml` and confirm
with `git diff -- _datafiles/config.yaml` that exactly one line is removed and
nothing else moved.

- [ ] **Step 4: Write the anti-backslide guard**

Create a root guard test:

```go
func TestConsistentAttackMessagesIsGone(t *testing.T) {
	banned := []string{"ConsistentAttackMessages", "msgSeed", "seedNum"}
	roots := []string{"internal", "modules", "_datafiles/config.yaml", "main.go"}
	for _, want := range banned {
		for _, root := range roots {
			hits := grepTree(t, root, want)
			// This plan's own docs may name it; code and config may not.
			if len(hits) > 0 {
				t.Errorf("%q still appears in %s: %v\n"+
					"The knob was an upstream mechanism superseded by Stage 9.2 and "+
					"deleted in M3 item 8. Do not reintroduce it; see the spec.",
					want, root, hits)
			}
		}
	}
}
```

Use whichever tree-walking helper the existing root guards already use rather
than writing a new one. Check `narration_render_callers_guard_test.go` for the
established pattern and reuse it.

- [ ] **Step 5: Prove the guard can fail**

Reintroduce the identifier in a comment in `internal/items/attack_messages.go`,
run the guard, confirm FAIL, then remove it and confirm PASS. A commented-out
occurrence is the right probe here: a `strings.Contains` guard once passed
while the call it guarded was merely commented.

- [ ] **Step 6: Full build and test**

Run: `go build ./... && go test ./...`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/items/attack_messages.go internal/configs/config.balance.go \
  internal/configs/config.balance.combat.go internal/configs/config.gameplay.go \
  _datafiles/config.yaml <the guard file>
git commit -m "refactor: delete ConsistentAttackMessages and the seeded pick

Upstream GoMud's mechanism (2a1c51087, 2024-11-21), sound where it came
from: pools were equal there, so one ItemId seed gave a coordinated triad
and a per-weapon voice at once. Stage 9.2 (bcd08700b) expanded the pools
and switched it off in the same commit, correctly, since a fixed index
would have shown 1 line of 15. That switch is also what replaced the one
shared index with a draw per role, which is where the uncoordinated triad
came from. Stage 9.5 then added tiers and pools drifted unequal, so
flipping it back would no longer have coordinated anything.

Not broken, superseded. Its only remaining effect was to reverse a
deliberate decision. Removes the field, the yaml key, both call-site
branches, msgSeed, both seeded getters and the unreachable
`if seedNum[0] == 0` branch that line 78's guard made dead on every input.

config.yaml edited from the HEAD blob, since it carries skip-worktree.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Re-key the golden to coordinated rows

**Files:**
- Modify: `internal/narration/snapshot_test.go`
- Modify: `internal/narration/testdata/stores/combat_messages.golden`

- [ ] **Step 1: Capture the current strings for the multiset check**

Write scratch files outside the repo, in this session's scratchpad directory.
`$SCRATCH` below stands for that path; set it once.

```bash
grep " => " internal/narration/testdata/stores/combat_messages.golden \
  | sed 's/^.* => //' | sort > "$SCRATCH/golden_before.txt"
wc -l < "$SCRATCH/golden_before.txt"
```

Expected: `6978`.

- [ ] **Step 2: Re-key the builder**

Rewrite the two loops from PR 1 so each row is one coordinated variant with all
roles on it, keyed `subtype|intensity|split|tier|index`:

```go
			for _, tier := range tiers {
				pools := map[string][]string{}
				widest := 0
				for _, role := range togetherRoles {
					pool := messageStringsForTest(tier.get(role.get(opts.Together)))
					pools[role.name] = pool
					if len(pool) > widest {
						widest = len(pool)
					}
				}
				for idx := 0; idx < widest; idx++ {
					fmt.Fprintf(&b, "%s|%s|together|%s|%d =>", subtype, intensity, tier.name, idx)
					for _, role := range togetherRoles {
						fmt.Fprintf(&b, " %s=%s", role.name, substituteTokens(pools[role.name][idx]))
					}
					fmt.Fprintf(&b, "\n")
				}
			}
```

and the same shape for `separate` over `separateRoles`. Rows stay keyed by the
**authored** role name inside the row, which is what makes a role swap visible.

- [ ] **Step 3: Re-record**

Run: `go test ./internal/narration/... -run TestSnapshotStores -update`

Expected: PASS.

- [ ] **Step 4: Confirm the row count**

Run: `grep -c " => " internal/narration/testdata/stores/combat_messages.golden`

Expected: `2283`, which is 2,280 coordinated store rows plus the 3
derived-selection rows at the foot.

- [ ] **Step 5: Confirm no string changed, only the grouping**

This is the property that proves a re-key rather than a rewrite.

The row format changed, so compare the multiset of message texts rather than
whole lines. A row now looks like:

```
slashing|critical|together|beginner|0 => toattacker=<text> todefender=<text> toroom=<text>
```

Each role value is separated from the next by ` <rolename>=`, and the texts
themselves contain spaces, so split on the role names rather than on spaces:

```bash
python - "$SCRATCH" <<'PY'
import re, sys, collections
scratch = sys.argv[1]
path = "internal/narration/testdata/stores/combat_messages.golden"
roles = ("toattacker", "todefender", "toroom", "toattackerroom", "todefenderroom")
splitter = re.compile(r"\s(?=(?:%s)=)" % "|".join(roles))
after = collections.Counter()
for line in open(path, encoding="utf-8"):
    if " => " not in line or line.startswith("#"):
        continue
    key, _, payload = line.partition(" => ")
    if key.startswith("derived|"):
        continue
    for field in splitter.split(payload.rstrip("\n")):
        _, _, text = field.partition("=")
        after[text] += 1
before = collections.Counter()
for line in open(f"{scratch}/golden_before.txt", encoding="utf-8"):
    before[line.rstrip("\n")] += 1
missing = before - after
added = after - before
print("texts only in PR 1 golden:", sum(missing.values()))
for t, n in list(missing.items())[:10]:
    print("   -", n, repr(t[:70]))
print("texts only in PR 2 golden:", sum(added.values()))
for t, n in list(added.items())[:10]:
    print("   +", n, repr(t[:70]))
PY
```

Expected: `texts only in PR 1 golden: 3` (the three derived-selection rows,
which this comparison strips from the new side but not the old) and
`texts only in PR 2 golden: 0`. Anything beyond those three means the re-key
dropped, duplicated or altered a message line, which is the one failure this
task can introduce.

- [ ] **Step 6: Commit**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/combat_messages.golden
git commit -m "test(narration): re-key combat-messages golden to coordinated rows

From one row per (role, index) to one row per coordinated variant with all
roles on it: 2,280 store rows covering the same 6,975 lines. The grouping
is now the thing under test, so a role swap or a lost coordination shows
as a moved string rather than as nothing.

Multiset of message texts verified identical to PR 1's golden.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Comment the default tree

**Files:**
- Modify: `_datafiles/world/default/combat-messages/*.yaml`, 8 files

- [ ] **Step 1: Add the banner to each of the 8 files**

Insert at the very top of each file, above the existing token comment header:

```yaml
# UPSTREAM GoMud HOLDOVER. NOT LOADED BY DOGMud.
#
# The live combat-message store is _datafiles/world/dogmud/combat-messages.
# This tree is upstream content kept for merge tracking only: main.go passes
# it to util.ValidateWorldFiles, which compares top-level DIRECTORY NAMES and
# never opens a file.
#
# It could not load if it were pointed at. These files use flat lists under
# each role, while the schema expects beginner/expert/master tiers, and none
# of them defines the `fumble` intensity that Validate() requires. The wider
# tree is missing 30+ subsystems the engine loads unconditionally.
#
# Do not edit these to match dogmud. See
# docs/superpowers/specs/2026-09-16-messaging-m3-item8-combat-messages-design.md
```

- [ ] **Step 2: Confirm nothing tries to load them**

Run: `go test ./...`

Expected: PASS. YAML comments cannot change behaviour; this is a regression
check that the files were not otherwise disturbed.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/default/combat-messages
git commit -m "docs(content): mark the default combat-messages tree as an upstream holdover

Owner ruling: the default world has nothing to do with us now, comment it
and move on. Measured while planning item 8: ValidateWorldFiles compares
directory names only and never opens a file, these 8 files use the
pre-tier flat schema, none defines fumble which Validate() requires, and
the tree lacks 30+ subsystems the engine loads unconditionally.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Documentation

**Files:**
- Modify: `internal/items/context.md`, `internal/narration/context.md`, `internal/configs/context.md`
- Modify: `docs/README.md`, `docs/superpowers/audits/messaging-m6-content-ledger.md`

- [ ] **Step 1: `internal/items/context.md`**

Document `PoolFor` and the two `Render` methods in the public API section, the
validator's per-tier equality rule with the reason it is exactly equivalent to
union equality, and the removal of the seeded getters. Verify every symbol you
name exists, with
`Select-String -Path internal\items\*.go -Pattern '^(func|type|const|var)\s'`.

- [ ] **Step 2: `internal/narration/context.md`**

Add combat-messages to the consumer list. Rewrite the `combat_messages.golden`
paragraph for the coordinated-row shape and its authored-role keying.

- [ ] **Step 3: `internal/configs/context.md`**

Remove the `ConsistentAttackMessages: true` line at 108. It documented the knob
as `true` while the shipped value was `false`; the drift dies with the knob.

- [ ] **Step 4: Run the context audit**

Run: `python tools/context_md_audit.py`

Expected: no findings for `internal/items`, `internal/narration` or
`internal/configs`.

- [ ] **Step 5: Check the `docs/README.md` rows still describe what shipped**

The spec row and both plan rows already exist. Re-read them against what
actually landed and correct any claim the implementation changed. A row that
describes the plan rather than the result is the thing to fix here.

- [ ] **Step 6: Commit**

```bash
git add internal/items/context.md internal/narration/context.md \
  internal/configs/context.md docs/README.md \
  docs/superpowers/audits/messaging-m6-content-ledger.md
git commit -m "docs: item 8 context.md updates and plan index rows

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Gate, playtest and PR

- [ ] **Step 1: Full suite**

Run: `go test ./...`

Expected: PASS. Expect combat-determinism tests to have moved, because the
draw count per message went from three or four to one. That is the documented
consequence of coordination, not a regression. Any test that moved should be
inspected and re-pinned deliberately, never re-recorded blindly.

- [ ] **Step 2: Boot check**

Follow `dogmud-shipping`'s detached-worktree boot check. The new validator runs
at load, so a content gap anywhere in the store fails the boot here.

- [ ] **Step 3: Adversarial playtest**

Per the content SOP. Two scenarios are required and neither is optional:

1. **Melee, two players in one room, several rounds.** Capture all three
   viewpoints of the same swings together and read them for coherence. The
   whole point of the slice is that the attacker, the defender and the room
   now describe one event; this is the only place that gets verified.
2. **Ranged, attacker and defender in different rooms.** Exercises the
   `separate` split and its fourth audience, including `shooting`'s newly
   authored `todefenderroom` lines for `prepare` and `wait`.

Use the Sable arena at Rift Chamber 5000 with gold set for a long fight: a
fixture that dies in one round returns a partial report.

Extract findings to memory afterwards. Playtest reports are gitignored.

- [ ] **Step 4: Pre-push gate**

Follow `dogmud-shipping`'s gate in its stated order. Do not skip hooks.

- [ ] **Step 5: Open the PR**

```bash
gh pr create --repo pruuk/DOGMud \
  --base master \
  --head <branch> \
  --title "refactor(combat): combat-messages onto the narration core (M3 item 8, PR 2)" \
  --body-file <path to a written body>
```

`--repo pruuk/DOGMud` is mandatory. This repo is a fork of
`GoMudEngine/GoMud` and `gh` defaults to the parent.

The body must state: that this closes the third and last live instance of the
arc's coordination defect; that the draw count per message drops from three or
four to one and therefore shifts the global random stream; that
`ConsistentAttackMessages` is deleted with the history that justifies it; and
that the golden was re-keyed with the message multiset proven unchanged.

End the body with:

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

- [ ] **Step 6: Do not deploy**

The owner runs all deploys. Note in the PR that this merges into the
undeployed stack, which already carries migrations 0.17.0 and 0.18.0.
