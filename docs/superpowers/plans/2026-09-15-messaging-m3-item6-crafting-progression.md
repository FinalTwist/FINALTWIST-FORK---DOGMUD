# Messaging M3 item 6: crafting and progression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Crafting text reaches every site through a store door with an empty Observer slot, and progression's six raw `events.Message` sends go through the messaging pipeline on `CategorySkillProgress`.

**Architecture:** Spec: `docs/superpowers/specs/2026-09-15-messaging-m3-item6-crafting-progression-design.md` (owner-approved 2026-09-15; read its facts table and rulings before starting any task). Crafting copies the 5b Kind B door shape (`internal/spells/narration.go`). Progression gets a boot-registered notifier, the codebase's existing answer to the `messaging` imports `characters` cycle.

**Tech Stack:** Go, the repo's `internal/narration` core, `internal/textutil` adapter, root-package AST/text guard tests.

**Branch:** `feature/messaging-m3-item6-crafting-progression` (already created, spec committed at `3747ba614`).

---

## Rules for every implementer (read first)

- **Edit tool only** for changing files. Never a Python read-modify-write (it truncates, and once converted a file to CRLF).
- **Never `git add -A` or `git add .`**; add named paths only. **Never `git reset`.** One implementer at a time; reviewers may run in parallel.
- `grep -c` that finds zero exits 1 and breaks `&&` chains: run "expect zero" checks standalone.
- `go test .` (repo root) runs the root guards; run it in every task that touches a guarded surface, not just the touched package.
- No em dashes or en dashes in any prose, comment or commit message.
- `-update` on the narration goldens is allowed ONLY in Task 0.
- Commit messages end with `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`. Use a heredoc (`git commit -F - <<'EOF'`), since bash expands backticks inside `-m`.

## File map

| File | Responsibility | Task |
|---|---|---|
| `internal/narration/snapshot_test.go` | `crafting.golden` builder + loader call | 0, 5 |
| `internal/narration/testdata/stores/crafting.golden` | frozen crafter lines | 0 |
| `internal/crafting/crafting.go` | two new fields, `Validate` calls the door check | 1 |
| `internal/crafting/narration.go` (new) | `Phase`, `Narration`, `Narrate`, `validateNarration` | 1 |
| `internal/crafting/narration_test.go` (new) | door and validation tests | 1 |
| `internal/actions/craft.go` | `CraftResult.SuccessMsg` becomes `Recipe` | 2 |
| `internal/usercommands/craft.go` | two instant-complete sites | 2 |
| `internal/hooks/NewRound_UserRoundTick.go` | multi-round success and failure sites | 3 |
| `internal/mobcommands/craft.go`, `internal/hooks/NewRound_MobRoundTick.go` | mob sites | 4 |
| `messaging_surface_guard_test.go` | viewpoint registry entries (2, 3), YAML key registry (5) | 2, 3, 5 |
| `store_text_fields_guard_test.go` | crafting row with exemptions | 5 |
| `internal/characters/progression_notify.go` (new) | notifier seam | 6 |
| `internal/characters/progression.go` | six sends become `notifyProgression` | 6 |
| `internal/characters/progression_notify_test.go` (new) | notifier tests | 6 |
| `internal/hooks/progression_notify.go` (new), `internal/hooks/progression_notify_test.go` (new) | the callback | 7 |
| `main.go` | registration | 7 |
| `progression_notifier_guard_test.go` (new, root) | registration guard | 7 |
| `raw_events_message_guard_test.go` (new, root) | no raw `events.Message{` outside plumbing | 8 |
| `context.md` x4, `internal/banner/banner.go` comment, `docs/README.md`, M6 ledger | docs | 9 |

---

### Task 0: Record `crafting.golden` from pre-migration code

The golden must be built BEFORE any production change, reading the raw recipe fields exactly as the four player sites send them today, so later tasks prove byte identity against real pre-change output.

**Files:**
- Modify: `internal/narration/snapshot_test.go` (imports, `setupRealStores`, `TestSnapshotStores`, new builder at end of file)
- Create: `internal/narration/testdata/stores/crafting.golden` (generated)

- [ ] **Step 1: Add the import and the loader call**

In the import block add, alphabetically after `configs`:

```go
	"github.com/GoMudEngine/GoMud/internal/crafting"
```

In `setupRealStores`, after `quests.LoadDataFiles()`:

```go
	// M3 item 6: recipes load from the same configured data path.
	crafting.LoadRecipeFiles()
```

- [ ] **Step 2: Add the subtest**

In `TestSnapshotStores`, after the `quests` subtest and before `post_pipeline`:

```go
	t.Run("crafting", func(t *testing.T) {
		checkGolden(t, "crafting.golden", buildCraftingGolden(t))
	})
```

- [ ] **Step 3: Add the pre-migration builder at the end of the file**

```go
// Store 11: crafting (internal/crafting: success_message / failure_message)
//
// Built from PRE-migration code: the raw field, color-wrapped exactly as the
// four player sites send it on CategorySystem. Task 5 of the M3 item 6 plan
// switches this builder to RecipeSpec.Narrate; the rows, their order and the
// header must not change, which is the byte-identity proof.
func buildCraftingGolden(t *testing.T) string {
	t.Helper()

	var b strings.Builder
	fmt.Fprintf(&b, "# crafting store snapshot (internal/crafting)\n")
	fmt.Fprintf(&b, "# Built 2026-09-15 from PRE-migration code: what the CRAFTER is sent, color wrap\n")
	fmt.Fprintf(&b, "# included (success green, failure red, CategorySystem). Recipes had no room line\n")
	fmt.Fprintf(&b, "# before M3 item 6; *_room_message rows appear only once one is authored.\n")
	fmt.Fprintf(&b, "# dimensions: recipe id x authored key; source only, a craft has no target\n\n")

	all := crafting.GetAll()
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		t.Fatal("no recipes loaded; setupRealStores must call crafting.LoadRecipeFiles()")
	}
	for _, id := range ids {
		r := all[id]
		fmt.Fprintf(&b, "recipe|%s|success_message => %s\n", id, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, r.SuccessMessage))
		fmt.Fprintf(&b, "recipe|%s|failure_message => %s\n", id, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, r.FailureMessage))
	}
	return b.String()
}
```

- [ ] **Step 4: Record it (the only `-update` in this plan)**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/crafting' -update`
Expected: `ok`.

- [ ] **Step 5: Verify the recording**

Run: `go test ./internal/narration/ -run TestSnapshotStores`
Expected: `ok` (all eleven subtests).

Run (standalone): `grep -c "^recipe|" internal/narration/testdata/stores/crafting.golden`
Expected: `252`.

Run: `git status --short`
Expected: only `internal/narration/snapshot_test.go` modified and `crafting.golden` new. If any other golden changed, stop: `-update` was run too broadly.

- [ ] **Step 6: Commit**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/crafting.golden
git commit -F - <<'EOF'
test(narration): crafting.golden recorded from pre-migration code

Freezes what the crafter is sent for all 126 recipes, color wrap included,
before M3 item 6 moves crafting onto a store door.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 1: The crafting store door

**Files:**
- Create: `internal/crafting/narration.go`
- Create: `internal/crafting/narration_test.go`
- Modify: `internal/crafting/crafting.go:46-47` (fields), `:60-74` (`Validate`)

- [ ] **Step 1: Write the failing tests**

`internal/crafting/narration_test.go`:

```go
package crafting

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/textutil"
)

func narrationTestRecipe() *RecipeSpec {
	return &RecipeSpec{
		RecipeId:       "test-stew",
		Name:           "Test Stew",
		Skill:          "cooking",
		Output:         RecipeOutput{ItemId: 30022, Quantity: 1},
		SuccessMessage: "You stir the pot. Stew!",
		FailureMessage: "The stew burns.",
	}
}

var narrationTestCrafter = textutil.TokenContext{
	SourceName:      `<ansi fg="username">Aliceia</ansi>`,
	SourcePlainName: "Aliceia",
}

func TestRecipeNarration_ActorIsTheCrafterLine(t *testing.T) {
	r := narrationTestRecipe()

	success := r.Narration(PhaseSuccess)
	if len(success.Actor) != 1 || success.Actor[0] != r.SuccessMessage {
		t.Errorf("success Actor = %q, want [%q]", success.Actor, r.SuccessMessage)
	}
	if len(success.Observer) != 0 || len(success.Actee) != 0 {
		t.Errorf("success with no room message: Observer %q Actee %q, want both empty", success.Observer, success.Actee)
	}

	failure := r.Narration(PhaseFailure)
	if len(failure.Actor) != 1 || failure.Actor[0] != r.FailureMessage {
		t.Errorf("failure Actor = %q, want [%q]", failure.Actor, r.FailureMessage)
	}
}

func TestRecipeNarrate_ObserverNamesTheCrafter(t *testing.T) {
	r := narrationTestRecipe()
	r.SuccessRoomMessage = "{source} ladles out a steaming stew."
	r.FailureRoomMessage = "Smoke pours from {source}'s pot."

	s := r.Narrate(PhaseSuccess, narrationTestCrafter)
	if s.Actor != r.SuccessMessage {
		t.Errorf("success Actor = %q, want %q", s.Actor, r.SuccessMessage)
	}
	if want := `<ansi fg="username">Aliceia</ansi> ladles out a steaming stew.`; s.Observer != want {
		t.Errorf("success Observer = %q, want %q", s.Observer, want)
	}

	f := r.Narrate(PhaseFailure, narrationTestCrafter)
	if want := `Smoke pours from <ansi fg="username">Aliceia</ansi>'s pot.`; f.Observer != want {
		t.Errorf("failure Observer = %q, want %q", f.Observer, want)
	}
}

func TestRecipeValidate_Narration(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(r *RecipeSpec)
		wantErr string
	}{
		{"shipped shape passes", func(r *RecipeSpec) {}, ""},
		{"room lines naming the crafter pass", func(r *RecipeSpec) {
			r.SuccessRoomMessage = "{source} finishes a stew."
			r.FailureRoomMessage = "{source} burns a stew."
		}, ""},
		{"empty success refused", func(r *RecipeSpec) { r.SuccessMessage = "" }, "success_message cannot be empty"},
		{"empty failure refused", func(r *RecipeSpec) { r.FailureMessage = "" }, "failure_message cannot be empty"},
		{"whitespace failure refused", func(r *RecipeSpec) { r.FailureMessage = "   " }, "is empty"},
		{"whitespace room line refused", func(r *RecipeSpec) { r.SuccessRoomMessage = "  " }, "is empty"},
		{"room line without source refused", func(r *RecipeSpec) { r.FailureRoomMessage = "A pot burns." }, "failure_room_message must name the crafter with {source}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := narrationTestRecipe()
			tc.mutate(r)
			err := r.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/crafting/ -run 'TestRecipeNarration|TestRecipeNarrate|TestRecipeValidate_Narration'`
Expected: build FAIL (`r.Narration undefined`, `PhaseSuccess undefined`, `SuccessRoomMessage undefined`).

- [ ] **Step 3: Add the fields**

In `internal/crafting/crafting.go`, replace lines 46-47:

```go
	SuccessMessage       string             `yaml:"success_message"`
	FailureMessage       string             `yaml:"failure_message"`
```

with:

```go
	SuccessMessage       string             `yaml:"success_message"`
	FailureMessage       string             `yaml:"failure_message"`
	// Observer slot for the room watching the crafter (M3 item 6). Empty in
	// every shipped recipe until M6 authors them. Read through Narrate only.
	SuccessRoomMessage string `yaml:"success_room_message,omitempty"`
	FailureRoomMessage string `yaml:"failure_room_message,omitempty"`
```

Run `gofmt -w internal/crafting/crafting.go` afterwards so the struct tags realign.

- [ ] **Step 4: Call the door check from `Validate`**

In `RecipeSpec.Validate`, replace the final `return nil` with:

```go
	return r.validateNarration()
```

- [ ] **Step 5: Write the door**

`internal/crafting/narration.go`:

```go
package crafting

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/textutil"
)

// Phase selects which of a recipe's two narrated outcomes to render. One line
// per outcome and audience, so the phase IS the selector.
type Phase uint8

const (
	PhaseSuccess Phase = iota
	PhaseFailure
)

// Narration assembles the variants for one outcome: the crafter's line is the
// Actor, the room's line the Observer. There is no Actee: a craft has no second
// party today. Enchanting another player's gear would add one here, as one
// optional key and one more field in this literal.
func (r *RecipeSpec) Narration(p Phase) narration.Variants {
	var crafter, room string
	switch p {
	case PhaseSuccess:
		crafter, room = r.SuccessMessage, r.SuccessRoomMessage
	case PhaseFailure:
		crafter, room = r.FailureMessage, r.FailureRoomMessage
	}
	return narration.Variants{Actor: textutil.Pool(crafter), Observer: textutil.Pool(room)}
}

// Narrate renders one outcome with the crafter as {source}. A craft has no
// target, so {target} renders empty.
func (r *RecipeSpec) Narrate(p Phase, ctx textutil.TokenContext) narration.Roles {
	return textutil.Narrate(r.Narration(p), ctx)
}

// validateNarration is called from Validate, so a violation fails the load.
//
// Both crafter lines are required: every site sends the Actor line
// unconditionally, as it did before the door existed, and all 126 shipped
// recipes set both. A room line must name the crafter with {source}, the same
// rule quests.RoomTextProblems applies to quest room_text: the room is watching
// someone work, and a subjectless line cannot say who.
func (r *RecipeSpec) validateNarration() error {
	if r.SuccessMessage == "" {
		return fmt.Errorf("recipe %q: success_message cannot be empty", r.RecipeId)
	}
	if r.FailureMessage == "" {
		return fmt.Errorf("recipe %q: failure_message cannot be empty", r.RecipeId)
	}
	phases := []struct {
		name string
		p    Phase
		room string
	}{
		{"success", PhaseSuccess, r.SuccessRoomMessage},
		{"failure", PhaseFailure, r.FailureRoomMessage},
	}
	for _, ph := range phases {
		// No expected role set: the Observer is deliberately optional until M6
		// authors it, so no fixed shape exists to declare. The blank-variant
		// check is what this call is for.
		if err := narration.ValidateVariants(r.Narration(ph.p), 1); err != nil {
			return fmt.Errorf("recipe %q %s text: %w", r.RecipeId, ph.name, err)
		}
		if ph.room != "" && !strings.Contains(ph.room, "{source}") {
			return fmt.Errorf("recipe %q: %s_room_message must name the crafter with {source} (the room is watching them work)", r.RecipeId, ph.name)
		}
	}
	return nil
}
```

- [ ] **Step 6: Run the tests and the shipped-data load**

Run: `go test ./internal/crafting/`
Expected: `ok`.

Run: `go test ./internal/narration/ -run TestSnapshotStores`
Expected: `ok` (this loads all 126 shipped recipes through `Validate`, proving the new rules refuse nothing shipped).

- [ ] **Step 7: Commit**

```bash
git add internal/crafting/crafting.go internal/crafting/narration.go internal/crafting/narration_test.go
git commit -F - <<'EOF'
feat(crafting): recipe store door with an empty Observer slot

RecipeSpec gains Phase / Narration / Narrate over textutil.Narrate, and two
optional room-message keys for M6 to author. Validate now requires both
crafter lines and refuses a room line that does not name the crafter.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 2: `CraftResult.Recipe` and the two instant-complete player sites

No shipped recipe has `time_rounds <= 0`, so both sites are unreachable in play today; the existing craft tests and the guard keep them honest.

**Files:**
- Modify: `internal/actions/craft.go:59`, `:161`
- Modify: `internal/usercommands/craft.go:130`, `:306`, `:627-632`
- Modify: `messaging_surface_guard_test.go` (`narrationViewpointRegistry`)

- [ ] **Step 1: Replace the text field with the recipe**

In `internal/actions/craft.go`, replace line 59:

```go
	SuccessMsg           string // recipe.SuccessMessage
```

with:

```go
	Recipe               *crafting.RecipeSpec // resolved recipe; render its text through Recipe.Narrate
```

and in `InitiateCraft`, replace `SuccessMsg:   recipe.SuccessMessage,` with `Recipe:       recipe,`. Run `gofmt -w internal/actions/craft.go`.

- [ ] **Step 2: Find every reader the compiler now rejects**

Run: `go build ./... && go vet ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/`
Expected: build FAIL naming `usercommands/craft.go:130` (`result.SuccessMsg undefined`). If any OTHER file is named, including a `_test.go` found by `go vet`, migrate it the same way in this task.

- [ ] **Step 3: Rewrite the instant-complete site in `Craft`**

Replace `internal/usercommands/craft.go:130`:

```go
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, result.SuccessMsg))
```

with:

```go
		roles := result.Recipe.Narrate(crafting.PhaseSuccess, textutil.TokenContext{
			SourceName:      user.Character.GetCharacterName(true),
			SourcePlainName: user.Character.GetCharacterName(false),
		})
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, roles.Actor))
		if roles.Observer != "" {
			room.SendTextVisual(messaging.CategoryEmote, roles.Observer, user.UserId)
		}
```

Add `"github.com/GoMudEngine/GoMud/internal/textutil"` to the file's imports (`crafting` is already imported).

- [ ] **Step 4: Rewrite `completeCraft` and its caller**

Replace the whole function at `internal/usercommands/craft.go:627-632`:

```go
// completeCraft resolves a craft instantly (used when time_rounds <= 0).
func completeCraft(user *users.UserRecord, recipe *crafting.RecipeSpec) {
	user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	newItem := items.New(recipe.Output.ItemId)
	user.Character.StoreItem(newItem)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))
}
```

with:

```go
// completeCraft resolves a craft instantly (used when time_rounds <= 0).
func completeCraft(user *users.UserRecord, room *rooms.Room, recipe *crafting.RecipeSpec) {
	user.Character.Items, user.Character.ComponentItems = crafting.ConsumeIngredients(user.Character.Items, user.Character.ComponentItems, recipe)
	newItem := items.New(recipe.Output.ItemId)
	user.Character.StoreItem(newItem)
	roles := recipe.Narrate(crafting.PhaseSuccess, textutil.TokenContext{
		SourceName:      user.Character.GetCharacterName(true),
		SourcePlainName: user.Character.GetCharacterName(false),
	})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, roles.Actor))
	if roles.Observer != "" {
		room.SendTextVisual(messaging.CategoryEmote, roles.Observer, user.UserId)
	}
}
```

and change the caller at `craft.go:306` (inside `craftEnchanting`, which has `room`) from `completeCraft(user, recipe)` to `completeCraft(user, room, recipe)`.

- [ ] **Step 5: Build and test the packages**

Run: `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/`
Expected: `ok` for all three.

- [ ] **Step 6: Run the root viewpoint guard and register what it surfaces**

Run: `go test . -run TestNarrationSitesMatchViewpointAudit`
Expected: FAIL listing newly visible candidate event(s) in `usercommands/craft.go` (actor + observer). The key is the file joined to the first string literal of the event's first send, so both sites most likely share the key `usercommands/craft.go|<ansi fg="green">%s</ansi>`; register exactly the key(s) printed, never a guessed one.

For each printed key add to `narrationViewpointRegistry`, keeping the map's alphabetical order:

```go
	"usercommands/craft.go|<ansi fg=\"green\">%s</ansi>": {verdictCorrect, true, false, true, "M3 item 6: instant craft completes; the crafter gets the recipe's success line and the room its Observer slot, empty until M6 authors it. A craft has no second party, so no actee. Read against source for this guard."},
```

If the guard ALSO lists a key as stale, read it: a changed literal elsewhere in `craft.go` means an existing entry's fingerprint moved; update that entry's key and keep its verdict and reason.

Run: `go test . -run TestNarrationSitesMatchViewpointAudit`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add internal/actions/craft.go internal/usercommands/craft.go messaging_surface_guard_test.go
git commit -F - <<'EOF'
refactor(crafting): instant-complete sites render through the recipe door

CraftResult carries the resolved recipe instead of a copy of its success text,
and both instant paths send the Actor line as before plus the Observer slot
when one is authored.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 3: The multi-round player sites

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:671` (success), `:701` (failure)
- Modify: `messaging_surface_guard_test.go` (`narrationViewpointRegistry`)

Both sites sit inside `if room := rooms.LoadRoom(roomId); room != nil {` (line 153, closed at 723), so `room` is in scope. `textutil` is already imported.

- [ ] **Step 1: Rewrite the success send**

Replace line 671:

```go
									user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, recipe.SuccessMessage))
```

with:

```go
									successRoles := recipe.Narrate(crafting.PhaseSuccess, textutil.TokenContext{
										SourceName:      user.Character.GetCharacterName(true),
										SourcePlainName: user.Character.GetCharacterName(false),
									})
									user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, successRoles.Actor))
									if successRoles.Observer != "" {
										sendVisualRoomText(room, messaging.CategoryEmote, successRoles.Observer, user.UserId)
									}
```

- [ ] **Step 2: Rewrite the failure send**

Replace (line numbers have shifted by 7):

```go
									user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, recipe.FailureMessage))
```

with:

```go
									failureRoles := recipe.Narrate(crafting.PhaseFailure, textutil.TokenContext{
										SourceName:      user.Character.GetCharacterName(true),
										SourcePlainName: user.Character.GetCharacterName(false),
									})
									user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, failureRoles.Actor))
									if failureRoles.Observer != "" {
										sendVisualRoomText(room, messaging.CategoryEmote, failureRoles.Observer, user.UserId)
									}
```

- [ ] **Step 3: Confirm no production reader of the old fields remains outside crafting**

Run (standalone): `grep -rn "recipe\.SuccessMessage\|recipe\.FailureMessage" --include=*.go internal modules`
Expected: no output (exit 1 is correct here).

- [ ] **Step 4: Build and test**

Run: `go build ./... && go test ./internal/hooks/`
Expected: `ok`. If `internal/hooks` shows `TestWaitRound_*` or `TestBuildGossipLine_*` failing, rerun once: those are known pre-existing shuffle-order flakes (they reproduce on master).

- [ ] **Step 5: Register the newly visible sites**

Run: `go test . -run TestNarrationSitesMatchViewpointAudit`
Expected: FAIL listing candidate event(s) in `hooks/NewRound_UserRoundTick.go`. Register exactly the printed keys, with these reasons (success key first literal is the green wrap, failure key the red wrap):

```go
	"hooks/NewRound_UserRoundTick.go|<ansi fg=\"green\">%s</ansi>": {verdictCorrect, true, false, true, "M3 item 6: multi-round craft succeeds (enchanting included); crafter gets the recipe's success line, the room its Observer slot, empty until M6. No second party, so no actee. Read against source for this guard."},
	"hooks/NewRound_UserRoundTick.go|<ansi fg=\"red\">%s</ansi>":   {verdictCorrect, true, false, true, "M3 item 6: multi-round craft fails; crafter gets the recipe's failure line, the room its Observer slot, empty until M6. No second party, so no actee. Read against source for this guard."},
```

If the printed viewpoints differ from actor + observer (for example the walk groups the discovery line into the success event), register the viewpoints the guard reports and say so in the reason; do not restructure the code to suit the walk.

Run: `go test . -run TestNarrationSitesMatchViewpointAudit`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go messaging_surface_guard_test.go
git commit -F - <<'EOF'
refactor(crafting): multi-round craft outcomes render through the recipe door

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 4: Mob craft sites use an authored room line when one exists

On shipped data no recipe authors a room line, so every mob line is unchanged.

**Files:**
- Modify: `internal/mobcommands/craft.go:48-51` (instant complete)
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (`tickMobCrafting`, win and loss branches)
- Create: `internal/crafting/mob_room_line.go`, test in `internal/crafting/narration_test.go`

The initiated line (`begins working on something.`) has no recipe phase and is not touched.

- [ ] **Step 1: Write the failing test for the shared helper**

Both mob sites need "authored Observer line, else the fixed fallback", so it lives once in the store. Append to `internal/crafting/narration_test.go`:

```go
func TestMobRoomLine_AuthoredLineElseFallback(t *testing.T) {
	r := narrationTestRecipe()
	const fallback = `<ansi fg="mobname">Smith</ansi> finishes their work.`

	if got := r.MobRoomLine(PhaseSuccess, "Smith", fallback); got != fallback {
		t.Errorf("no authored room line: got %q, want the fallback %q", got, fallback)
	}
	if got := r.MobRoomLine(PhaseFailure, "Smith", ""); got != "" {
		t.Errorf("no authored failure line and no fallback: got %q, want empty", got)
	}

	r.SuccessRoomMessage = "{source} sets down a finished stew."
	if want := `<ansi fg="mobname">Smith</ansi> sets down a finished stew.`; r.MobRoomLine(PhaseSuccess, "Smith", fallback) != want {
		t.Errorf("authored room line: got %q, want %q", r.MobRoomLine(PhaseSuccess, "Smith", fallback), want)
	}
}
```

Run: `go test ./internal/crafting/ -run TestMobRoomLine`
Expected: build FAIL (`r.MobRoomLine undefined`).

- [ ] **Step 2: Write the helper**

`internal/crafting/mob_room_line.go`:

```go
package crafting

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/textutil"
)

// MobRoomLine is what the room sees when a mob reaches outcome p on this
// recipe: the recipe's authored Observer line with the mob as {source}, or
// fallback when the recipe authors none. Mobs have no client, so only the
// room line matters. fallback may be empty, which keeps the outcome silent.
func (r *RecipeSpec) MobRoomLine(p Phase, mobName, fallback string) string {
	roles := r.Narrate(p, textutil.TokenContext{
		SourceName:      fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mobName),
		SourcePlainName: mobName,
	})
	if roles.Observer != "" {
		return roles.Observer
	}
	return fallback
}
```

Run: `go test ./internal/crafting/`
Expected: `ok`.

- [ ] **Step 3: Mob instant complete**

In `internal/mobcommands/craft.go`, replace:

```go
		room.SendTextVisual(messaging.CategoryMobIdle, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> works quickly and produces something.`,
			mob.Character.Name))
```

with:

```go
		room.SendTextVisual(messaging.CategoryMobIdle, result.Recipe.MobRoomLine(
			crafting.PhaseSuccess, mob.Character.Name, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> works quickly and produces something.`,
				mob.Character.Name)))
```

Add the `crafting` import if the file lacks it.

- [ ] **Step 4: Mob multi-round win and loss**

In `tickMobCrafting` (`internal/hooks/NewRound_MobRoundTick.go`), replace the win branch's send:

```go
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			sendVisualRoomText(room, messaging.CategoryMobIdle, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> finishes their work.`,
				mob.Character.Name))
		}
```

with:

```go
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			sendVisualRoomText(room, messaging.CategoryMobIdle, recipe.MobRoomLine(
				crafting.PhaseSuccess, mob.Character.Name, fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> finishes their work.`,
					mob.Character.Name)))
		}
```

and in the `else` (loss) branch, after the `ConsumeIngredients` assignment, add:

```go
		// A failed mob craft is silent unless the recipe authors a failure room line.
		if line := recipe.MobRoomLine(crafting.PhaseFailure, mob.Character.Name, ""); line != "" {
			if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
				sendVisualRoomText(room, messaging.CategoryMobIdle, line)
			}
		}
```

- [ ] **Step 5: Build and test**

Run: `go build ./... && go test ./internal/mobcommands/ ./internal/hooks/ ./internal/crafting/ && go test . -run TestNarrationSitesMatchViewpointAudit`
Expected: `ok` for all. (The mob sends are observer-only, so the viewpoint walk does not list them; if it does, register as in Task 3 with a reason naming the mob branch.)

- [ ] **Step 6: Commit**

```bash
git add internal/crafting/mob_room_line.go internal/crafting/narration_test.go internal/mobcommands/craft.go internal/hooks/NewRound_MobRoundTick.go
git commit -F - <<'EOF'
feat(crafting): mob crafts use a recipe's authored room line when one exists

Falls back to today's fixed lines, so shipped output is unchanged; a mob's
failed craft stays silent unless a failure room line is authored.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 5: Golden through the door, store guard, key registry, sabotage probes 1 and 3

**Files:**
- Modify: `internal/narration/snapshot_test.go` (`buildCraftingGolden`)
- Modify: `store_text_fields_guard_test.go`
- Modify: `messaging_surface_guard_test.go:83-87` (YAML key registry)

- [ ] **Step 1: Switch the builder to the door**

Replace the body of the `for _, id := range ids {` loop in `buildCraftingGolden` with:

```go
		r := all[id]
		success := r.Narrate(crafting.PhaseSuccess, kindBNoTarget)
		failure := r.Narrate(crafting.PhaseFailure, kindBNoTarget)
		fmt.Fprintf(&b, "recipe|%s|success_message => %s\n", id, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, success.Actor))
		if success.Observer != "" {
			fmt.Fprintf(&b, "recipe|%s|success_room_message => %s\n", id, success.Observer)
		}
		fmt.Fprintf(&b, "recipe|%s|failure_message => %s\n", id, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, failure.Actor))
		if failure.Observer != "" {
			fmt.Fprintf(&b, "recipe|%s|failure_room_message => %s\n", id, failure.Observer)
		}
```

and change the builder's doc comment to:

```go
// Store 11: crafting (internal/crafting: success_message / failure_message and
// the optional *_room_message Observer slot)
//
// Recorded 2026-09-15 from PRE-migration code (the raw field, color-wrapped as
// the four player sites sent it). Since M3 item 6 this builder reads through
// RecipeSpec.Narrate; the rows, their order and the header are unchanged, which
// is the byte-identity proof. Rows are keyed by the AUTHORED key, so a swapped
// Actor and Observer shows as a changed row.
```

- [ ] **Step 2: All eleven goldens byte-identical (no `-update`)**

Run: `go test ./internal/narration/`
Expected: `ok`.

- [ ] **Step 3: Sabotage probe 1, proven red**

Temporarily edit `internal/crafting/narration.go` `Narration` return to `narration.Variants{Actor: textutil.Pool(room), Observer: textutil.Pool(crafter)}`.

Run: `go test ./internal/narration/ -run TestSnapshotStores/crafting`
Expected: FAIL with a golden mismatch whose diff shows `success_message => <ansi fg="green"></ansi>` rows. Record the first diff line for the task report.

Revert the edit exactly (`git diff internal/crafting/narration.go` must print nothing), then rerun and expect `ok`.

- [ ] **Step 4: Sabotage probe 3, proven red**

Temporarily delete the `if ph.room != "" && !strings.Contains(...)` block in `validateNarration` (and the now-unused `strings` import).

Run: `go test ./internal/crafting/ -run TestRecipeValidate_Narration`
Expected: FAIL on `room_line_without_source_refused`.

Revert exactly (`git diff internal/crafting/narration.go` prints nothing); rerun, expect `ok`.

- [ ] **Step 5: Store text field guard gains a crafting row with exemptions**

In `store_text_fields_guard_test.go`, replace the `storeTextFieldOwners` declaration with:

```go
var storeTextFieldOwners = []struct {
	owner   string
	pattern *regexp.Regexp
	what    string
	// exempt maps a production file to why a match there is not a store read:
	// the same field spelling on an unrelated struct. Every exempt file must
	// still match, or the entry is stale.
	exempt map[string]string
}{
	{"internal/conditions", regexp.MustCompile(`\.(StartUserText|StartRoomText|TriggerUserText|TriggerRoomText|EndUserText|EndRoomText)\b`), "condition text fields", nil},
	{"internal/spells", regexp.MustCompile(`\.(CastUserText|CastRoomText|WaitUserText|WaitRoomText|MagicUserText|MagicRoomText)\b`), "spell text fields", nil},
	{"internal/quests", regexp.MustCompile(`Rewards\.(PlayerMessage|RoomMessage)\b`), "quest reward messages", nil},
	{"internal/crafting", regexp.MustCompile(`\.(SuccessMessage|FailureMessage|SuccessRoomMessage|FailureRoomMessage)\b`), "recipe message fields", map[string]string{
		"internal/hooks/NewRound_IdleMobs_patrol.go":   "FailureMessage on the patrol plan struct, a log string unrelated to recipes",
		"internal/hooks/NewRound_IdleMobs_schedule.go": "FailureMessage on the schedule plan struct, a log string unrelated to recipes",
	}},
}
```

Update the comment above it: replace "Every condition, spell and quest-reward line" with "Every condition, spell, quest-reward and recipe line".

In `TestStoreTextFieldsAreReadOnlyByTheirStore`, declare `exemptSeen := map[string]bool{}` next to `inside := 0`, and replace:

```go
					if strings.HasPrefix(rel, owner.owner+"/") {
```

with:

```go
					if _, ok := owner.exempt[rel]; ok {
						exemptSeen[rel] = true
						continue
					}
					if strings.HasPrefix(rel, owner.owner+"/") {
```

and after the `if inside == 0 {` block add:

```go
		for file, why := range owner.exempt {
			if !exemptSeen[file] {
				t.Errorf("%s: exemption for %s (%s) matched nothing; remove the stale entry", owner.what, file, why)
			}
		}
```

- [ ] **Step 6: YAML key registry**

`TestEveryTextSurfaceIsRegistered` builds its schema from keys found in world YAML (`messagingSurfaceSplitSchema` iterates only `keyFiles`), so a registry entry for a key no file sets yet is reported STALE. The two new room keys therefore must NOT be registered until M6 authors them in 2+ files; the comment records that instead.

In `messaging_surface_guard_test.go`, replace the crafting comment and its two entries (lines 83-87; confirm by reading first) with:

```go
	// -- Crafting narration: internal/crafting/crafting.go RecipeSpec, 126
	// recipe files. Since M3 item 6 the recipe is a store with a door
	// (internal/crafting/narration.go): the *_message keys are the crafter's
	// Actor line. The Observer slot keys, success_room_message and
	// failure_room_message, are deliberately NOT registered: no shipped file
	// sets them, so this guard would report them stale. Register them in the
	// M6 commit that authors them. --
	"success_message": {narration, "internal/crafting/crafting.go RecipeSpec.SuccessMessage -- the crafter's (Actor) line on a successful craft, 126 recipe files; rendered through RecipeSpec.Narrate."},
	"failure_message": {narration, "internal/crafting/crafting.go RecipeSpec.FailureMessage -- the crafter's (Actor) line on a failed craft, paired with success_message; rendered through RecipeSpec.Narrate."},
```

- [ ] **Step 7: Root guards**

Run: `go test . -run 'TestStoreTextFieldsAreReadOnlyByTheirStore|TestEveryTextSurfaceIsRegistered|TestNarrationSitesMatchViewpointAudit'`
Expected: `ok`.

Prove the new row capable of failing: temporarily add `_ = result.Recipe.SuccessMessage` inside `Craft` in `internal/usercommands/craft.go`, rerun `go test . -run TestStoreTextFieldsAreReadOnlyByTheirStore`, expect FAIL naming `internal/usercommands/craft.go`; revert exactly and rerun, expect `ok`.

- [ ] **Step 8: Commit**

```bash
git add internal/narration/snapshot_test.go store_text_fields_guard_test.go messaging_surface_guard_test.go
git commit -F - <<'EOF'
test(crafting): golden reads through the door; store guard covers recipes

crafting.golden is byte-identical through RecipeSpec.Narrate. The store field
guard gains a recipe row with per-file exemptions for the unrelated patrol and
schedule FailureMessage. The key registry reasons name the Actor role, and the
unset Observer keys are recorded as deliberately unregistered until M6.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 6: The progression notifier in `internal/characters`

**Files:**
- Create: `internal/characters/progression_notify.go`
- Create: `internal/characters/progression_notify_test.go`
- Modify: `internal/characters/progression.go:206`, `:211`, `:301`, `:610`, `:979-985`

- [ ] **Step 1: Write the failing tests**

`internal/characters/progression_notify_test.go`:

```go
package characters

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/progression"
)

type recordedNotice struct {
	userId int
	text   string
}

// recordProgressionNotices installs a recording notifier for one test.
func recordProgressionNotices(t *testing.T) *[]recordedNotice {
	t.Helper()
	var got []recordedNotice
	SetProgressionNotifier(func(userId int, text string) {
		got = append(got, recordedNotice{userId, text})
	})
	t.Cleanup(func() { SetProgressionNotifier(nil) })
	return &got
}

// A rank-0 skill roll at a huge multiplier clamps to chance 1.0 under the
// pinned base of 1.0, so the gain is certain; the Fatal below proves it.
const certainSkillMultiplier = 1000.0

const notifyTestUser = 7

func assertNoRawProgressionMessages(t *testing.T) {
	t.Helper()
	if queued := events.DrainQueuedMessagesForTest(notifyTestUser); len(queued) != 0 {
		t.Errorf("progression queued %d raw events.Message %q; it must go through the notifier", len(queued), queued)
	}
}

func TestProgressionNotice_SkillBanner(t *testing.T) {
	pinCertainStatProgressionForTest(t)
	events.DrainQueuedMessagesForTest(notifyTestUser)
	got := recordProgressionNotices(t)

	c := newProgressionTestCharacter(t)
	if !c.CheckSkillProgression("weapon-combat", notifyTestUser, certainSkillMultiplier) {
		t.Fatal("pinned skill roll did not progress; the test proves nothing")
	}
	if len(*got) != 1 {
		t.Fatalf("got %d notices, want 1: %q", len(*got), *got)
	}
	n := (*got)[0]
	if n.userId != notifyTestUser || !strings.Contains(n.text, "SKILL ADVANCEMENT") {
		t.Errorf("notice = %+v, want user %d and a SKILL ADVANCEMENT banner", n, notifyTestUser)
	}
	if strings.HasSuffix(n.text, "\n") {
		t.Errorf("notice ends in a newline; UserRecord.SendText adds it")
	}
	assertNoRawProgressionMessages(t)
}

func TestProgressionNotice_StatBanner(t *testing.T) {
	pinCertainStatProgressionForTest(t)
	events.DrainQueuedMessagesForTest(notifyTestUser)
	got := recordProgressionNotices(t)

	c := newProgressionTestCharacter(t)
	if !c.CheckStatProgression("willpower", notifyTestUser, 1.0) {
		t.Fatal("pinned stat roll did not progress; the test proves nothing")
	}
	if len(*got) != 1 || !strings.Contains((*got)[0].text, "STATISTIC INCREASED") {
		t.Fatalf("notices = %q, want one STATISTIC INCREASED banner", *got)
	}
	assertNoRawProgressionMessages(t)
}

func TestProgressionNotice_RegenLine(t *testing.T) {
	pinCertainStatProgressionForTest(t)
	events.DrainQueuedMessagesForTest(notifyTestUser)
	got := recordProgressionNotices(t)

	c := newProgressionTestCharacter(t)
	before := c.GetStatTraining("willpower")
	c.CheckRegenProgression("willpower", notifyTestUser, 1.0)
	if c.GetStatTraining("willpower") <= before {
		t.Fatal("pinned regen roll did not progress; the test proves nothing")
	}
	want := `<ansi fg="magenta">***</ansi> Your <ansi fg="yellow">willpower</ansi> grows stronger! <ansi fg="magenta">***</ansi>`
	if len(*got) != 1 || (*got)[0].text != want {
		t.Fatalf("notices = %q, want exactly [%q]", *got, want)
	}
	assertNoRawProgressionMessages(t)
}

func TestProgressionNotice_CritAndFumbleLines(t *testing.T) {
	pinCertainStatProgressionForTest(t)
	cases := []struct {
		class progression.Class
		line  string
	}{
		{progression.ClassCrit, fmt.Sprintf(`<ansi fg="magenta">***</ansi> A moment of brilliance! Your <ansi fg="yellow">%s</ansi> technique improves! <ansi fg="magenta">***</ansi>`, "weapon-combat")},
		{progression.ClassFumble, fmt.Sprintf(`<ansi fg="red">!!!</ansi> You learn from your mistake! Your <ansi fg="yellow">%s</ansi> understanding deepens. <ansi fg="red">!!!</ansi>`, "weapon-combat")},
	}
	for _, tc := range cases {
		events.DrainQueuedMessagesForTest(notifyTestUser)
		got := recordProgressionNotices(t)
		c := newProgressionTestCharacter(t)
		c.applyBonusProgression(progression.Event{Skill: "weapon-combat", Class: tc.class, Multiplier: certainSkillMultiplier}, notifyTestUser)
		// The banner first (CheckSkillProgression), then the class line.
		if len(*got) != 2 || !strings.Contains((*got)[0].text, "SKILL ADVANCEMENT") || (*got)[1].text != tc.line {
			t.Errorf("class %v notices = %q, want [banner, %q]", tc.class, *got, tc.line)
		}
		assertNoRawProgressionMessages(t)
	}
}

func TestProgressionNotice_MobAndNilNotifierSendNothing(t *testing.T) {
	pinCertainStatProgressionForTest(t)
	events.DrainQueuedMessagesForTest(notifyTestUser)

	got := recordProgressionNotices(t)
	mobLike := newProgressionTestCharacter(t)
	if !mobLike.CheckStatProgression("willpower", 0, 1.0) {
		t.Fatal("pinned stat roll did not progress; the test proves nothing")
	}
	if len(*got) != 0 {
		t.Errorf("userId 0 produced notices %q; only players are notified", *got)
	}

	SetProgressionNotifier(nil)
	c := newProgressionTestCharacter(t)
	if !c.CheckStatProgression("willpower", notifyTestUser, 1.0) {
		t.Fatal("pinned stat roll did not progress; the test proves nothing")
	}
	assertNoRawProgressionMessages(t)
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/characters/ -run TestProgressionNotice`
Expected: build FAIL (`SetProgressionNotifier undefined`).

- [ ] **Step 3: Write the seam**

`internal/characters/progression_notify.go`:

```go
package characters

// progressionNotifyFn delivers a progression line (banner, regen gain, crit or
// fumble lesson) to a player. It is registered from main at boot, because this
// package cannot import messaging or users: messaging imports characters.
// Follows SetUserUntargetableCheck. nil = no delivery, the safe default for
// tests.
//
// 🔴 A missing registration silences ALL progression text while every unit
// test stays green. The root guard progression_notifier_guard_test.go asserts
// main.go registers it.
var progressionNotifyFn func(userId int, text string)

// SetProgressionNotifier registers the progression delivery callback.
// Repeated registrations overwrite; pass nil to disable.
func SetProgressionNotifier(fn func(userId int, text string)) {
	progressionNotifyFn = fn
}

// notifyProgression is the ONLY way progression text leaves this package.
// text carries no trailing newline; the pipeline's send adds one. Mobs
// (userId <= 0) have no client and are never notified.
func notifyProgression(userId int, text string) {
	if userId <= 0 || progressionNotifyFn == nil {
		return
	}
	progressionNotifyFn(userId, text)
}
```

- [ ] **Step 4: Replace the six sends**

In `internal/characters/progression.go`:

Line 206 and line 211, each `events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})` becomes:

```go
				notifyProgression(userId, msg)
```

Line 301, same replacement.

Line 610, same replacement.

Lines 979-985, replace:

```go
			events.AddToQueue(events.Message{UserId: userId, Text: fmt.Sprintf(
				`<ansi fg="magenta">***</ansi> A moment of brilliance! Your <ansi fg="yellow">%s</ansi> technique improves! <ansi fg="magenta">***</ansi>`,
				ev.Skill) + "\n"})
		case progression.ClassFumble:
			events.AddToQueue(events.Message{UserId: userId, Text: fmt.Sprintf(
				`<ansi fg="red">!!!</ansi> You learn from your mistake! Your <ansi fg="yellow">%s</ansi> understanding deepens. <ansi fg="red">!!!</ansi>`,
				ev.Skill) + "\n"})
```

with:

```go
			notifyProgression(userId, fmt.Sprintf(
				`<ansi fg="magenta">***</ansi> A moment of brilliance! Your <ansi fg="yellow">%s</ansi> technique improves! <ansi fg="magenta">***</ansi>`,
				ev.Skill))
		case progression.ClassFumble:
			notifyProgression(userId, fmt.Sprintf(
				`<ansi fg="red">!!!</ansi> You learn from your mistake! Your <ansi fg="yellow">%s</ansi> understanding deepens. <ansi fg="red">!!!</ansi>`,
				ev.Skill))
```

The existing `userId > 0` gates around each send stay; `notifyProgression` checks again so a future caller cannot skip it.

Run (standalone): `grep -n "events.Message" internal/characters/progression.go`
Expected: no output. `events.SkillUsed` at line 428 must remain (it is not a message).

- [ ] **Step 5: Run tests**

Run: `go build ./... && go test ./internal/characters/`
Expected: `ok`. If `events` became unused in `progression.go`, the build says so; it should not, because `events.SkillUsed` still uses it.

- [ ] **Step 6: Commit**

```bash
git add internal/characters/progression_notify.go internal/characters/progression_notify_test.go internal/characters/progression.go
git commit -F - <<'EOF'
refactor(progression): send progression text through a registered notifier

The six raw events.Message sends become notifyProgression, a boot-registered
callback, because characters cannot import messaging. Nothing is registered
yet; the next commit wires it.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 7: The callback, its registration, and the registration guard

Task 6 left progression silent in a booted server until this task lands, so do not open a PR or run a playtest between the two.

**Files:**
- Create: `internal/hooks/progression_notify.go`, `internal/hooks/progression_notify_test.go`
- Modify: `main.go` (after the `characters.SetUserUntargetableCheck(...)` block, around line 313)
- Create: `progression_notifier_guard_test.go` (repo root, `package main`)

- [ ] **Step 1: Write the failing callback tests**

`internal/hooks/progression_notify_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestProgressionNotifyCallback_SkillProgressCategory(t *testing.T) {
	u := users.NewTestUser(901, "prognotify", "Prognotify", 1901)
	restore := users.SeedUsersForTest(map[int]*users.UserRecord{901: u})
	defer restore()
	events.DrainQueuedMessagesForTest(901)

	ProgressionNotifyCallback(901, "BANNER")

	got := events.DrainQueuedMessagesForTest(901)
	want := `<ansi fg="skill-progress">BANNER</ansi>` + "\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("queued %q, want exactly [%q]", got, want)
	}
}

func TestProgressionNotifyCallback_UnknownUserSendsNothing(t *testing.T) {
	restore := users.SeedUsersForTest(map[int]*users.UserRecord{})
	defer restore()
	events.DrainQueuedMessagesForTest(902)

	ProgressionNotifyCallback(902, "BANNER")

	if got := events.DrainQueuedMessagesForTest(902); len(got) != 0 {
		t.Fatalf("queued %q for a user who is not online, want nothing", got)
	}
}
```

Run: `go test ./internal/hooks/ -run TestProgressionNotifyCallback`
Expected: build FAIL (`ProgressionNotifyCallback undefined`).

- [ ] **Step 2: Write the callback**

`internal/hooks/progression_notify.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ProgressionNotifyCallback delivers a progression line through the messaging
// pipeline on CategorySkillProgress, the category quest skill-up lines already
// use. Registered from main via characters.SetProgressionNotifier, because
// characters cannot import messaging or users.
//
// The category's color stage wraps the whole line in skill-progress (alias 179),
// so an untagged banner renders gold: an owner-approved change (M3 item 6 spec).
// A user who is not online gets nothing, as the old raw Message listener did.
func ProgressionNotifyCallback(userId int, text string) {
	if user := users.GetByUserId(userId); user != nil {
		user.SendText(messaging.CategorySkillProgress, text)
	}
}
```

Run: `go test ./internal/hooks/ -run TestProgressionNotifyCallback`
Expected: `ok`.

- [ ] **Step 3: Write the failing registration guard**

`progression_notifier_guard_test.go`:

```go
package main

import (
	"os"
	"strings"
	"testing"
)

// characters.SetProgressionNotifier defaults to nil, which is silent on
// purpose so tests need no wiring. The cost of that default: deleting the
// registration from main.go silences every skill and stat banner in the game
// while every unit test stays green. This guard is the only thing that sees it.
func TestProgressionNotifierRegisteredAtBoot(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go (test must run from the repo root): %v", err)
	}
	const want = "characters.SetProgressionNotifier(hooks.ProgressionNotifyCallback)"
	if !strings.Contains(string(src), want) {
		t.Errorf("main.go does not register the progression notifier (%s); every progression line would be silent", want)
	}
}
```

Run: `go test . -run TestProgressionNotifierRegisteredAtBoot`
Expected: FAIL (`main.go does not register`).

- [ ] **Step 4: Register in `main.go`**

Immediately after the closing `})` of the `characters.SetUserUntargetableCheck(...)` call, add:

```go

	// Register progression delivery so skill and stat banners reach the
	// messaging pipeline without characters importing messaging or users
	// (messaging imports characters, so that would be a cycle).
	characters.SetProgressionNotifier(hooks.ProgressionNotifyCallback)
```

Run: `go build ./... && go test . -run TestProgressionNotifierRegisteredAtBoot`
Expected: `ok`.

- [ ] **Step 5: Sabotage probe 4, proven red**

Temporarily comment out the registration line in `main.go`.
Run: `go test . -run TestProgressionNotifierRegisteredAtBoot`
Expected: FAIL. Restore the line exactly (`git diff main.go` shows only the Step 4 addition); rerun, expect `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/progression_notify.go internal/hooks/progression_notify_test.go main.go progression_notifier_guard_test.go
git commit -F - <<'EOF'
feat(progression): progression lines go through the pipeline on CategorySkillProgress

main registers hooks.ProgressionNotifyCallback. Banners and the untagged words
of the regen, crit and fumble lines now render in skill-progress gold, matching
quest skill-up lines (owner-approved). A root guard asserts the registration.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 8: No raw `events.Message{` outside the pipeline plumbing

**Files:**
- Create: `raw_events_message_guard_test.go` (repo root, `package main`)

- [ ] **Step 1: Write the guard**

```go
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// rawEventsMessageAllowed lists the only production files that may construct
// events.Message directly. Everything else sends through UserRecord.SendText or
// the Room.SendText* family, which run messaging.RenderForRecipient (category,
// normalize, sight gate, color). Progression was the last known bypass until
// M3 item 6; code in internal/characters uses notifyProgression.
var rawEventsMessageAllowed = map[string]string{
	"internal/rooms/rooms.go":                "the pipeline's own fan-out: Room sends queue per-recipient Messages after rendering",
	"internal/users/userrecord.go":           "UserRecord.SendText itself, the end of the per-user pipeline",
	"internal/usercommands/print.go":         "the print debug command, which echoes its raw argument by purpose",
	"internal/hooks/hooks.go":                "listener registration: events.Message{} is a type key, not a send",
	"internal/hooks/Message_SendMessages.go": "the listener; the match is its comment naming direct events.Message{RoomId} constructions",
}

func TestNoRawEventsMessageOutsidePipeline(t *testing.T) {
	pattern := regexp.MustCompile(`events\.Message\{`)
	seen := map[string]bool{}
	var outside []string

	for _, root := range messagingSurfaceGoRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel := filepath.ToSlash(path)
			for i, line := range strings.Split(string(src), "\n") {
				if !pattern.MatchString(line) {
					continue
				}
				if _, ok := rawEventsMessageAllowed[rel]; ok {
					seen[rel] = true
					continue
				}
				outside = append(outside, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	// Each allowlist entry must still match, which also proves the pattern can
	// match at all: an empty "outside" list from a blind pattern proves nothing.
	for file, why := range rawEventsMessageAllowed {
		if !seen[file] {
			t.Errorf("allowlist entry %s (%s) no longer constructs events.Message; remove it", file, why)
		}
	}
	sort.Strings(outside)
	for _, o := range outside {
		t.Errorf("raw events.Message outside the pipeline: %s\n  send through UserRecord.SendText or Room.SendText*; from internal/characters use notifyProgression", o)
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test . -run TestNoRawEventsMessageOutsidePipeline`
Expected: `ok`. If it lists a file not in the allowlist, read the site: a genuine new bypass gets migrated in this task; a plumbing site gets an allowlist entry with a reason read from its source.

- [ ] **Step 3: Sabotage probe 2, proven red**

Temporarily add to the top of `notifyProgression` in `internal/characters/progression_notify.go`:

```go
	_ = events.Message{UserId: userId}
```

(with the `events` import). Run: `go test . -run TestNoRawEventsMessageOutsidePipeline`
Expected: FAIL naming `internal/characters/progression_notify.go`. Revert exactly (`git diff internal/characters/` prints nothing); rerun, expect `ok`.

- [ ] **Step 4: Commit**

```bash
git add raw_events_message_guard_test.go
git commit -F - <<'EOF'
test(messaging): root guard forbids raw events.Message outside the pipeline

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 9: Documentation and the M6 ledger

**Files:**
- Modify: `internal/crafting/context.md`, `internal/characters/context.md`, `internal/narration/context.md`, `internal/banner/context.md`
- Modify: `internal/banner/banner.go:1-16` (package comment)
- Modify: `docs/superpowers/audits/messaging-m6-content-ledger.md` (created before execution; if it is absent, stop and report)
- Modify: `docs/README.md`

Verify every symbol named in docs exists: `Select-String -Path internal\crafting\*.go -Pattern '^(func|type|const|var)\s'` (PowerShell) or `codegraph_search`.

- [ ] **Step 1: `internal/crafting/context.md`**

Read it, then add a section (adapt headings to the file's existing style):

```markdown
## Narration (M3 item 6)

A recipe is a narration store. Read its text only through the door in
`narration.go`; `store_text_fields_guard_test.go` fails on a read of
`SuccessMessage`, `FailureMessage`, `SuccessRoomMessage` or
`FailureRoomMessage` anywhere else.

- `Phase`: `PhaseSuccess`, `PhaseFailure`.
- `(*RecipeSpec).Narration(p) narration.Variants`: `success_message` /
  `failure_message` are the Actor (the crafter); `success_room_message` /
  `failure_room_message` are the Observer (the room). No Actee: a craft has no
  second party. Enchanting another player's gear would add one.
- `(*RecipeSpec).Narrate(p, textutil.TokenContext) narration.Roles`: renders
  through `textutil.Narrate`, crafter as `{source}`.
- `(*RecipeSpec).MobRoomLine(p, mobName, fallback) string`: a mob crafter's
  room line, the authored Observer line or the fallback.
- `Validate` refuses: an empty `success_message` or `failure_message`; a
  whitespace-only line; a room message without `{source}`.

Room messages are empty in all 126 shipped recipes until M6 authors them; a
player craft with none sends nothing to the room.
```

- [ ] **Step 2: `internal/characters/context.md`**

Add under the progression section:

```markdown
### Progression text delivery

Progression never builds an `events.Message`. Banners (`banner.Format`), the
regen gain line and the crit and fumble lines go through `notifyProgression`,
which calls the callback registered with `SetProgressionNotifier`. `main.go`
registers `hooks.ProgressionNotifyCallback`, which sends on
`messaging.CategorySkillProgress`. The callback exists because `messaging`
imports `characters`. With no callback registered (unit tests) nothing is
sent; `progression_notifier_guard_test.go` asserts the boot registration and
`raw_events_message_guard_test.go` forbids a raw send here.
```

- [ ] **Step 3: `internal/banner/banner.go` package comment and `context.md`**

Replace the paragraph in the package comment that says progression "calls Format() and queues the banner directly via events.AddToQueue" with:

```go
// characters/progression.go calls Format() and hands the banner to
// notifyProgression, which delivers it through the messaging pipeline on
// CategorySkillProgress via a callback registered in main.go.
```

Keep the paragraph about the phantom `SendProgression` helper. Make the same correction wherever `internal/banner/context.md` describes delivery.

- [ ] **Step 4: `internal/narration/context.md`**

Add `crafting` to the list of Kind B stores reaching `Render` through `textutil.Narrate`, and `crafting.golden` to the goldens list, in the file's existing format.

- [ ] **Step 5: M6 ledger rows**

Read `docs/superpowers/audits/messaging-m6-content-ledger.md` and add, in its crafting and progression groups using its column format, rows for (skip any the ledger already carries, adding this plan as a source instead):

1. Crafting: author `success_room_message` for 126 recipes. Size 126. Prerequisite none. Deferred by this spec, "Out of scope, filed". Note: the authoring commit must also register both room keys in `textSurfaceRegistry` (they are unregistered today because no file sets them).
2. Crafting: author `failure_room_message` for 126 recipes. Size 126. Same source.
3. Crafting: decide whether a player craft with no authored room line should get a generic room fallback (players are silent today, mobs get fixed lines). Size not counted. Owner call.
4. Crafting: an Actee line once enchanting another player's gear exists. Prerequisite that feature. Owner ruling 2026-09-15.
5. Crafting: mob initiated line (`begins working on something.`) is fixed text with no recipe phase. Size 1 site. Filed.
6. Progression: owner may retune the gold (`skill-progress` alias 179), which also recolors quest skill-up, recipe-learned and spell-learned lines. Size 1 alias. Filed.

- [ ] **Step 6: README rows**

The spec, this plan and the ledger are already indexed in `docs/README.md`. If any task changed what one of them says in a way its row describes (for example the room keys not being registered), update that row's text.

- [ ] **Step 7: Audit and commit**

Run: `python tools/context_md_audit.py`
Expected: no phantom symbols reported for `crafting`, `characters`, `narration` or `banner`.

```bash
git add internal/crafting/context.md internal/characters/context.md internal/narration/context.md internal/banner/context.md internal/banner/banner.go docs/superpowers/audits/messaging-m6-content-ledger.md docs/README.md
git commit -F - <<'EOF'
docs(messaging): M3 item 6 context.md updates and M6 ledger rows

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 10: Gate, playtest, PR

Load the `dogmud-shipping` and `dogmud-playtesting` skills before this task and follow them where they are more specific than the steps below.

- [ ] **Step 1: Full suite and static checks**

Run: `gofmt -l internal modules *.go`
Expected: no output.

Run: `go vet ./...`
Expected: no output.

Run: `go test ./... 2>&1 | Select-String -Pattern '^(FAIL|ok|---)' ` (PowerShell) or the bash equivalent
Expected: every package `ok`. If `internal/playtestrun` fails under load, rerun it standalone and record both runs. If `internal/hooks` shuffle-order flakes appear (`TestWaitRound_*`, `TestBuildGossipLine_*`), rerun once and record.

Run: `go test .`
Expected: `ok` (root guards).

- [ ] **Step 2: Boot check**

Follow `dogmud-shipping`'s detached-worktree boot check. Expected: `Server Ready`, zero panics, the recipe load line shows `loadedCount=126`.

- [ ] **Step 3: Playtest lane**

Per `dogmud-playtesting` (ephemeral goals file, `--checkout` of this branch's HEAD, extract findings to memory afterwards). One lane, two players in a lit room with a `cooking_fire` station (candidates: `hartcharn/6459`, `pothole_coulee/5290`, `pothole_coulee/6463`; confirm the room is lit before relying on it):

1. Player one (crafter) and player two (observer) both in the room. Give player one `raw-meat` (item 40014) and `salt-pouch` (item 40017) and the `grilled-meat` recipe.
2. Player one: `craft grilled-meat`. After the rounds, player one's outcome line must equal the `recipe|grilled-meat|success_message` or `failure_message` row of `crafting.golden` (as rendered, color tags interpreted), exactly once.
3. Player two must see NO crafting outcome line.
4. Drive a skill or stat gain on player one (repeat the craft, or use an admin progression grant if the harness exposes one). The banner must arrive once, whole, and the report must record the raw bytes of the banner's first rule line so the `skill-progress` (color 179) wrap is checked, not judged.
5. Nothing to player two from player one's progression.

Record findings to memory per the playtesting skill (reports are gitignored).

- [ ] **Step 4: Whole-branch review**

Dispatch a reviewer over `git diff master...HEAD` with the spec. The reviewer must run `go test .` and `go test ./internal/narration/ ./internal/crafting/ ./internal/characters/ ./internal/hooks/` itself.

- [ ] **Step 5: Push and PR**

Per `dogmud-shipping`: push the branch, then

```bash
gh pr create --repo pruuk/DOGMud --base master --head feature/messaging-m3-item6-crafting-progression --title "Messaging M3 item 6: crafting and progression onto the narration path" --body-file <body file>
```

The body lists: what changed, the one player-visible change (progression gold, owner-approved, with the banner bytes from the playtest), byte-identity evidence (`crafting.golden` unchanged through the door, sabotage probes 1 to 4 red), and ends with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`. Merge on green CI. Do NOT deploy (the owner defers deploys until the messaging arc is done).
