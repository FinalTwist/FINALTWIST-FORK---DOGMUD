# Messaging M3 item 6: crafting and progression onto the narration path

Date: 2026-09-15. Branch `feature/messaging-m3-item6-crafting-progression` off
master `42606b10a`. Item 6 of M3 in the messaging unification arc
(`2026-08-31-messaging-unification-design.md`, M3 table). The arc names these
two as "the two that need new behavior": crafting gains audience slots, and
progression comes inside the pipeline and gains a `Category`.

Owner rulings taken during this design (2026-09-15, do not relitigate):

- **Crafting is Actor plus Observer, no Actee today.** A craft has no second
  party. The door must leave an Actee open for a future "enchant another
  player's gear" feature, which it does by construction: `narration.Roles`
  already carries `Actee`, so adding one is one optional key and one line in
  the door.
- **Progression lands on `CategorySkillProgress` and turns gold.** The
  pipeline's color stage wraps the line in `<ansi fg="skill-progress">` (alias
  179 in `ansi-aliases.yaml`). The banner has no tags, so it turns gold whole;
  the untagged words of the crit, fumble and regen lines turn gold. This
  matches the quest skill-up, recipe-learned and spell-learned lines, which
  already use that category. It is the only player-visible change in the slice.
- **Approach A**: a store door for crafting, and a boot-registered notifier for
  progression. A new progression event (B) was rejected because the extra
  queue hop reorders the banner against same-round lines; returning text to
  callers (C) was rejected because `AwardResolved` alone has 58 production
  call sites across 51 files, and making every caller send the text is the
  exact "effect copied, narration dropped" shape M1's audit found behind all
  five of its defects.

## Facts verified against source

Every claim below was read from the tree at `42606b10a` on 2026-09-15.

### Crafting

| Fact | Where |
|---|---|
| `RecipeSpec.SuccessMessage` (`yaml:"success_message"`) and `FailureMessage` (`yaml:"failure_message"`), both `string` | `internal/crafting/crafting.go:46-47` |
| `RecipeSpec.Validate()` checks id, name, skill and output only; nothing about the messages | `crafting.go:60-74` |
| Shipped data: 126 recipe files, dogmud world only (the default world has no `recipes/`); 126 `success_message`, 126 `failure_message`, none empty, zero `{token}` in any of them | grep over `_datafiles/world/*/recipes` |
| Recipe text reaches a player at FOUR sites, all `CategorySystem`, all color-wrapped: success `<ansi fg="green">%s</ansi>`, failure `<ansi fg="red">%s</ansi>` | `internal/usercommands/craft.go:130` (instant complete, via `result.SuccessMsg`), `craft.go:631` (`completeCraft`), `internal/hooks/NewRound_UserRoundTick.go:671` (multi-round success, shared by enchanting), `:701` (multi-round failure) |
| `completeCraft` has one caller | `usercommands/craft.go:306` |
| `CraftResult.SuccessMsg` copies `recipe.SuccessMessage` out of the store; its only reader is `usercommands/craft.go:130` | `internal/actions/craft.go:59,161` |
| A PLAYER craft sends nothing to the room at start, completion or failure | the four sites above; `usercommands/craft.go:131-135` (initiated is actor-only) |
| A MOB craft sends three fixed room lines that ignore the recipe, all `CategoryMobIdle`: instant `works quickly and produces something.`, initiated `begins working on something.`, multi-round win `finishes their work.`; a mob's failed craft is silent | `internal/mobcommands/craft.go:49,61`; `internal/hooks/NewRound_MobRoundTick.go` (win branch of the mob craft completion, `sendVisualRoomText`) |
| `crafting` imports `configs contest fileloader items mudlog skills util`; `textutil` depends only on `copyover mudlog term util narration`, so `crafting` may import `textutil` with no cycle | `go list` |
| The 5b door shape to copy: `Phase` type, `Narration(p) narration.Variants`, `Narrate(p, ctx) narration.Roles` via `textutil.Narrate`, and a `validateNarration` running `narration.ValidateVariants(v, 1)` per phase with text | `internal/spells/narration.go:13-60` |
| `textutil.Narrate(v narration.Variants, ctx TokenContext) narration.Roles`; `textutil.Pool(text)`; `TokenContext{SourceName, SourcePlainName, TargetName, TargetPlainName}` | `internal/textutil/narrate.go:8,25`; `tokens.go:10-14` |
| Quest `room_text` is refused at load when it lacks `{source}`: the in-repo precedent for an Observer line that must name its actor | `internal/quests/roomtext.go` |
| The YAML key registry lists `success_message` and `failure_message` as narration with a "no user/room split exists" reason | `messaging_surface_guard_test.go:86-87` |
| `TestStoreTextFieldsAreReadOnlyByTheirStore` holds a per-store regex table (conditions, spells, quest rewards) | `store_text_fields_guard_test.go:25-32` |
| ⚠️ The spelling `.FailureMessage` also exists on two unrelated plan structs, so a crafting field guard must be qualified | `internal/hooks/NewRound_IdleMobs_patrol.go:25,86,116`; `NewRound_IdleMobs_schedule.go:24,135,178` |
| The snapshot harness loads real stores in `setupRealStores`; ten goldens exist, none for crafting | `internal/narration/snapshot_test.go:131-158`; `internal/narration/testdata/stores/` |

### Progression

| Fact | Where |
|---|---|
| SIX raw `events.AddToQueue(events.Message{UserId: userId, Text: msg + "\n"})` sends, each gated `userId > 0` | `internal/characters/progression.go:206` (skill banner), `:211` (banner for an unknown skill name), `:301` (stat banner), `:610` (regen stat line), `:979` (crit line), `:983` (fumble line) |
| Banners come from `banner.Format(kind, name, tier)`: pure text, no ANSI tags | `internal/banner/banner.go` |
| `banner`'s package comment still says progression "queues the banner directly via events.AddToQueue" | `banner.go:7-9` |
| 🔴 `internal/messaging` imports `internal/characters`, so `characters` cannot import `messaging` or `users` | `go list -f '{{.Imports}}' ./internal/messaging` |
| The codebase's answer to that cycle is a callback registered at boot in `main.go`: `characters.SetUserUntargetableCheck`, `users.SetCanSeeInRoomCheck`, `rooms.SetCompanionTransport` | `internal/characters/engagement_storage.go:60-76`; `main.go:296-330` |
| `UserRecord.SendText(cat, txt)` runs `messaging.RenderForRecipient` on `ChannelAudio` and queues the same `events.Message{UserId, Text: rendered + "\n"}` | `internal/users/userrecord.go:486-500` |
| `RenderForRecipient` stages that apply on audio: normalize, then color. `CategorySkillProgress` skips every normalize stage ("banner has its own formatting"); color wraps `<ansi fg="skill-progress">...</ansi>` for any non-default category | `internal/messaging/pipeline.go:53-116`; `normalize.go:25-38` |
| `skill-progress: 179` | `_datafiles/world/dogmud/ansi-aliases.yaml:256` |
| Quest skill-up, stat-up, recipe-learned and spell-learned lines already use `CategorySkillProgress` | `internal/hooks/Quest_HandleQuestUpdate.go:359,374,388,397`; `internal/questengine/bridge.go:302,310` |
| `events.Message` has one listener, `Message_SendMessage`, which AnsiParses and sends; its `UserId` branch is identical for both paths | `internal/hooks/Message_SendMessages.go:16-40`; `hooks.go:94` |
| Remaining raw `events.Message{` producers outside progression: `rooms/rooms.go` (6, pipeline plumbing), `users/userrecord.go:496` (`SendText` itself), `usercommands/print.go:17` (the `print` command, raw by purpose) | grep |
| Production callers of the progression entry points: `AwardResolved` 58, `OnStatUse` 6, `OnRegenTick` 6, `ApplyProgression` 6, `OnStatUseScaled` 5, `OnSkillUse` 5, `CheckStatProgression` 4, `CheckSkillProgression` 3, `OnSkillUseScaled` 3; 51 files | grep, `_test.go` excluded |
| No test asserts progression's text through the event queue; `consider_no_progression_test.go` guards consider and look structurally, by source text | grep over `*_test.go` for the line text and `banner.Format` |

## What item 6 delivers

After this slice:

- Crafting is a store with a door. Its text reaches every site as
  `narration.Roles` through `textutil.Narrate`; nothing outside
  `internal/crafting` reads a recipe message field.
- Recipes have an Observer slot: two new optional keys, empty in all 126 files.
  M6 authors them.
- Progression never builds an `events.Message`. Its six lines go through the
  pipeline on `CategorySkillProgress`.
- A root guard makes a raw `events.Message{` outside the pipeline plumbing a
  test failure, which is one of the arc's end-state guards.

Crafting output is byte-identical. Progression output is identical up to the
outer color wrap, which is the ruled change.

## Design

### Crafting store door (`internal/crafting/narration.go`)

```go
type Phase uint8

const (
	PhaseSuccess Phase = iota
	PhaseFailure
)

// Narration: the crafter's line is the Actor, the room line the Observer.
// No Actee today; enchanting another player's gear would add one here.
func (r *RecipeSpec) Narration(p Phase) narration.Variants
func (r *RecipeSpec) Narrate(p Phase, ctx textutil.TokenContext) narration.Roles
func (r *RecipeSpec) validateNarration() error
```

`RecipeSpec` gains:

```go
SuccessRoomMessage string `yaml:"success_room_message,omitempty"`
FailureRoomMessage string `yaml:"failure_room_message,omitempty"`
```

Empty fields are omitted from `Variants` (via `textutil.Pool`), so an absent
room line renders an empty Observer.

**Validation**, called from `RecipeSpec.Validate`, three rules:

1. `success_message` and `failure_message` must be non-empty. All 126 shipped
   recipes set both, so this refuses nothing, and it means the Actor line
   always exists at every site, so no site needs an empty-Actor branch.
2. For each phase, `narration.ValidateVariants(v, 1)` refuses a
   whitespace-only line.
3. A non-empty room message that lacks `{source}` is refused, matching quest
   `room_text`. Zero shipped recipes carry a room message.

Recipes registered through `RegisterRecipeForTest` bypass `Validate`, as today.

### Crafting sites

The four player sites become:

```go
roles := recipe.Narrate(crafting.PhaseSuccess, tCtx)
user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">%s</ansi>`, roles.Actor))
if roles.Observer != "" {
	room.SendTextVisual(messaging.CategoryEmote, roles.Observer, user.UserId)
}
```

with the crafter's tagged and plain names in `tCtx`, failure red, and today's
category and wrap on the Actor send, unconditionally, as today. The Observer
send is unreachable on shipped data; it uses `CategoryEmote`, the category
quest reward room lines already use, and M4 settles categories for good.

`CraftResult.SuccessMsg string` is replaced by `Recipe *crafting.RecipeSpec`,
set at the same point, so the instant-complete site renders through the door
on the recipe `InitiateCraft` already resolved. No second lookup.

**Mob sites.** Instant complete and the multi-round win render the recipe's
`PhaseSuccess` Observer line when one is authored and fall back to today's
fixed line when not. Initiated has no recipe phase and keeps its fixed line. A
mob's failed craft stays silent to the room unless the recipe authors a
failure room line. On shipped data every mob line is unchanged.

A player craft with no authored room line stays silent to the room. A generic
player fallback would be new player-facing text, which is M6's.

### Progression notifier (`internal/characters`)

```go
// progressionNotifyFn delivers a progression line to a player. Registered from
// main at boot, because characters cannot import messaging or users
// (messaging imports characters). nil = no delivery (safe default for tests).
var progressionNotifyFn func(userId int, text string)

func SetProgressionNotifier(fn func(userId int, text string))

// notifyProgression is the only way progression text leaves this package.
func notifyProgression(userId int, text string)
```

The six sends become `notifyProgression(userId, msg)` with the text as built
today, without the trailing `"\n"` (`SendText` adds it). The callback is a
named function in `internal/hooks`, beside `hooks.CompanionTransportCallback`,
and `main.go` registers it next to `SetUserUntargetableCheck`:

```go
// internal/hooks/progression_notify.go
func ProgressionNotifyCallback(userId int, text string) {
	if user := users.GetByUserId(userId); user != nil {
		user.SendText(messaging.CategorySkillProgress, text)
	}
}

// main.go
characters.SetProgressionNotifier(hooks.ProgressionNotifyCallback)
```

🔴 **A missing registration silences all progression text with every unit
test green**, because the nil default is deliberately silent. So a root guard
asserts `main.go` registers it (below), and the playtest checks it live.

The call is synchronous, so the line is queued at the same point in the round
as today. A missing user drops the line, as `Message_SendMessage` does today.

`banner`'s package comment and `characters/context.md` are corrected to name
the notifier.

## The net

### Golden first

`internal/narration/testdata/stores/crafting.golden`, built from PRE-migration
code and committed before any production change. One row per recipe per
authored key, sorted by id, recording what the crafter is SENT including the
color wrap: `recipe|anti-corrosion-quench|success_message => <ansi fg="green">...</ansi>`.
`setupRealStores` additionally calls `crafting.LoadRecipeFiles()`.

After the migration the builder switches to the door and all eleven goldens
(ten existing plus this one) must be byte-identical. `-update` is used once,
in the task that records `crafting.golden`, and nowhere else.

### Tests

- Door: Actor/Observer assembly per phase; a room message without `{source}`
  refused; a whitespace-only line refused; an empty room line renders an empty
  Observer.
- Progression: with a recording notifier installed, each of the six paths
  delivers the same text as today minus the newline; with no notifier, nothing
  is queued.
- `hooks.ProgressionNotifyCallback`: with a registered test user, the queued
  `events.Message` carries the text wrapped in `<ansi fg="skill-progress">`
  and a trailing newline; for an unknown user id, nothing is queued.

### Sabotage probes, each proven red before it is trusted

1. Swap Actor and Observer in `RecipeSpec.Narration`: `crafting.golden` goes red.
2. Put one raw `events.Message{` back in `progression.go`: the root guard goes red.
3. Delete the `{source}` rule: the refusal test goes red.
4. Delete the `SetProgressionNotifier` line from `main.go`: the registration
   guard goes red.

### Root guards

- `TestNoRawEventsMessageOutsidePipeline`: `events.Message{` may appear in
  production code only in `internal/rooms/rooms.go`,
  `internal/users/userrecord.go` and `internal/usercommands/print.go`, each
  registered with a reason, plus the listener registration in
  `internal/hooks/hooks.go` and the comment in `Message_SendMessages.go`.
- `TestProgressionNotifierRegisteredAtBoot`: `main.go` contains
  `characters.SetProgressionNotifier(hooks.ProgressionNotifyCallback)`.
- `TestStoreTextFieldsAreReadOnlyByTheirStore` gains a crafting row covering
  `SuccessMessage`, `FailureMessage`, `SuccessRoomMessage`,
  `FailureRoomMessage`, qualified so the patrol and schedule plan structs'
  `FailureMessage` does not match.
- The YAML key registry gains `success_room_message` and
  `failure_room_message`, and the reasons on the two existing keys stop saying
  no audience split exists.
- The viewpoint registry gains an entry for each site the rewrite makes newly
  visible to `TestNarrationSitesMatchViewpointAudit`.

### Gate

Full suite green (`internal/playtestrun` standalone if it reds under load),
`go test .` for the root guards, gofmt, vet, the boot check, and one playtest
lane in a lit room with two players: player one crafts an instant recipe and
a multi-round recipe; player two, present, must see no crafting line; player
one's success lines must match `crafting.golden` verbatim. Then drive a skill
or stat gain on player one and confirm the banner arrives once, whole, in
color 179, and not duplicated. The report records the raw bytes of the banner
line so the gold wrap is checked, not judged.

## Out of scope, filed

- Authoring `success_room_message` / `failure_room_message` for 126 recipes,
  and any generic player-craft room fallback: M6.
- The crafting Actee: when enchanting another player's gear exists.
- The hand-rolled craft lines (`You begin crafting...`, discovery, storage
  pulls, refusals) and mob craft fixed lines: Group C and A, not store text.
- Moving the recipe keys to one role vocabulary: M4.
- `print.go`'s raw send: a debug command, registered, not migrated.

## Documentation

`context.md` for `crafting` (the door, the two keys, the validation rules),
`characters` (the notifier), `banner` (package comment), and `narration` (the
new golden and consumer). Rows for this spec and its plan in `docs/README.md`.
Patch note: none for crafting; the PR description records the progression
color change for the owner.
