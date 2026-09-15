# Messaging M6 Content Ledger

**Created:** 2026-09-15, while M3 item 6 (crafting, progression) was being
planned. Not a spec or a plan: a standing collection point.

## Purpose

The messaging unification arc has repeatedly deferred player-facing text, and
the code that would carry it, to a stage called M6 (the content pass), and a
few times to M5 (the quality pass) or to "filed" with no stage named at all.
Each slice's design document records its own deferrals in an "out of scope"
or "deliberately not in this slice" section, but nobody had collected them
across slices in one place before this document, so M6 would otherwise have
to reread every prior spec and every playtest memory file to find its own
work list. This ledger is that collection: every deferred item, where it was
deferred, and what (if anything) blocks it. It also records deferrals that
never named M6 by name but fit the same shape (missing or wrong player text,
filed rather than fixed), because the arc spec's own M6 section describes its
job as "authoring the text for slots that M3 and M4 created but left empty,"
and several of the entries below are exactly that even though the source
document said "M5" or just "filed."

## How to maintain

Every slice that defers content work adds its own rows to this ledger in the
same commit that writes its design or its "out of scope" section, citing
itself as the source, rather than leaving the next audit to rediscover it.
When M6 (or any later slice) closes a row, it marks the row's Status column
`done (PR #___)` and leaves the row in place rather than deleting it, so the
ledger stays a complete record of what was ever deferred, not just what is
still open. A row is never deleted, only marked done.

## Sources read

Docs:
- `docs/superpowers/specs/2026-08-31-messaging-unification-design.md` (full)
- `docs/superpowers/specs/completed/2026-09-08-messaging-m2-shared-narration-seam-design.md` (full)
- `docs/superpowers/specs/completed/2026-09-09-messaging-m3-store-core-extraction-design.md` (full)
- `docs/superpowers/specs/completed/2026-09-11-messaging-m3-item5a-narration-defects-design.md` (full)
- `docs/superpowers/specs/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration-design.md` (full)
- `docs/superpowers/specs/completed/2026-09-15-messaging-m3-item6-crafting-progression-design.md` (full)
- `docs/README.md` (full, for the arc's row-by-row doc index)
- `docs/superpowers/plans/completed/2026-09-09-messaging-m3-store-core-extraction.md` (grepped for `\bM6\b`, matches read in context)
- `docs/superpowers/plans/completed/2026-09-11-messaging-m3-item5a-narration-defects.md` (grepped for `\bM6\b`, matches read in context)
- `docs/superpowers/plans/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration.md` (grepped for `\bM6\b`, matches read in context)
- `docs/superpowers/audits/2026-08-31-messaging-surface-sweep.md` (grepped for deferral keywords, matches read in context)
- `docs/superpowers/audits/2026-09-07-narration-viewpoint-audit.md` (grepped for deferral keywords, matches read in context)
- `docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md` (grepped for deferral keywords)
- `docs/superpowers/specs/completed/2026-09-14-conditions-unification-slice-2-rename-design.md` (grepped for deferral keywords)
- `docs/superpowers/specs/completed/2026-09-14-conditions-unification-slice-1b-mechanics-design.md` (grepped for deferral keywords)
- `docs/superpowers/specs/completed/2026-09-12-conditions-unification-slice-1-model-design.md` (grepped for deferral keywords)

Memory (`C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`):
- `MEMORY.md` (grepped)
- `project-session-handoff-2026-09-15.md` (full)
- `project-session-handoff-2026-09-12.md` (full)
- `project-conditions-unification-arc.md` (full)
- `project-messaging-m3-item5a-design.md` (full)
- `project-messaging-m3-item5b-design.md` (full)
- `project-messaging-m3-store-core-design.md` (full)
- `project-messaging-m3-item5a-followups.md` (full)
- `project-followup-slice-a-names-in-dark.md` (full)
- `project-followup-slice-b-red-names-and-combat-name-leaks.md` (full)
- `project-followup-slice-c-buff-notices.md` (full)
- `project-attack-the-darkness-messaging.md` (full)
- `project-shield-heal-buff-identity.md` (full)
- `project_world_service_coverage.md`, `feedback-owner-does-the-deploys.md`, `reference_prod_perf_baseline.md`, `project_stillwater_bank.md` (grepped; confirmed unrelated, "M6" and "content pass" there are coincidental matches to an unrelated deploy log line and an unrelated world-building backlog, no new ledger items)

Git:
- `git log --oneline --all -i --grep="M6"` (8 commits, all documentation commits for specs already read above, no additional deferred item found)
- `git log --oneline -i --grep="wording" --all` and `git log --oneline -i --grep="wording|copy" --since=2026-08-31` (checked for a deferral recorded only in a commit message; none found beyond what the specs and memory above already record)

**Not read:** the remaining 26 of the 37 memory files the broad grep for
`\bM6\b|content pass|filed` matched were not opened, because a second, tighter
grep for `\bM6\b|content pass` (dropping the bare word "filed," which matches
constantly on unrelated backlog items) narrowed the true candidate set to 11
files, all of which are listed as read above.

## Ledger

| # | Item | Size | Prerequisite | Deferred by | Owner ruling / notes | Status |
|---|---|---|---|---|---|---|
| **Conditions (buffs)** | | | | | | |
| 1 | One merged "who did what" line per audience for a condition applied by a spell (today the spell's own line and the condition's start line both fire, and the condition's line cannot name the caster at all) | 17 buffs, per the owner ruling; the defects table separately states "17 of 19 spell-to-buff links, 5 of them also double the room line" | A caster id on `events.Buff` (renamed `events.Condition` as of slice 2 of the conditions arc) plus a `{caster}` token in `textutil` | `docs/superpowers/specs/completed/2026-09-11-messaging-m3-item5a-narration-defects-design.md`, "Owner rulings" and "Deliberately not in this slice"; restated in `docs/superpowers/specs/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration-design.md`, "Out of scope, filed" ("Actee text for buffs, spells and quests, and a caster on `events.Buff`: M6"); memory `project-messaging-m3-item5a-design.md` | Owner (2026-09-11): "the right end state is one merged line per audience, with the code supplying who and the data supplying what." 5a fixed only the self-cast wording in the meantime. See the Open Question below: this reframes the arc spec's original 427-field Actee count, since a condition's holder line already occupies the Actee role and was never the actual gap | open |
| 2 | Per-spell shield/ward identity: distinct display names, expiry lines and web Status-tab entries for each shield spell. Today both shield spells (Conviction Ward, Chrysalis Cocoon) apply the same condition, whose display name and expiry text are hardcoded "Minor Shield" / "Your Minor Shield dissipates." Whether the heal spells share a single condition the same way is unverified | not counted (2 shield spells confirmed sharing one name; heal-spell overlap "unverified", per source) | none named; owner tied it to a balance pass and a possible pruning of shield spells, not yet scoped | Memory `project-shield-heal-buff-identity.md` | Owner (2026-09-11): "We really need different names for the buff than just 'minor shield' that is reused for all of the shield type spells. We could also balance the spells, prune the number of them, and/or add some other functionality to them like reflect protection." Explicitly "NOT YET SCOPED... Brainstorm before planning" | open |
| 3 | Reskin commands (`howl`, a wolf reskin of `taunt`; `charge`, a boar reskin of trip/`ExecuteTrip`) hand-roll one fixed line each with no variety, because routing them through the shared taunt or combat-messages pool would erase the reskin's own voice | 2 reskins (`howl`, `charge`), each one hand-rolled `fmt.Sprintf` line today with no pool at all | none; needs its own new pool per reskin, not a slot | Memory `project-messaging-m3-store-core-design.md`, "M6 CONTENT FOLLOWUP, OWNER-APPROVED 2026-09-09" | Owner (2026-09-09): "we may want to apply different text lines to howl (it is a reskin of taunt after all and could have its own messages)... record the howl issue (and probably all the other reskins) as a followup for the content part of this arc." | open |
| 4 | The shared taunt pool (`rhetoric.yaml`) is written for people trading barbs and reads oddly from a beast (example seen in play: "Alpha Pack-Leader trips over their own words trying to mock you," said of a hound with no words). Possible fix named but not chosen: species- or command-flavored pools | not counted | none stated | Memory `project-messaging-m3-store-core-design.md`, "CONTENT FINDING FOR THE OWNER" | "Whether a beast drawing 'questions your courage with venomous words' reads acceptably is OPEN, and is an M6 content question, not a code one." Same root problem as row 3 | open |
| **Defence / taunt / grapple / casting** | | | | | | |
| 5 | `defense-messages/counter-defy.yaml` calls every countered conviction attack a "taunt," which is misleading when the exchange was never a taunt command | 11 occurrences of the literal word "taunt" in the file, per source | none | Memory `project-messaging-m3-store-core-design.md`, "FILED FOR M5" | Not an M6 item; filed for M5's quality pass. Kept here because it is the same class of miscategorized authored text the ledger exists to track, and M5/M6 boundary items are explicitly in scope per this document's purpose | open |
| 6 | Casting's `cast_started` pool has only 3 variants and fires on every cast, so a caster sees a repeat roughly every third spell, against defence's 10 to 14 variants per pool | 3 variants shipped; no target count set (source frames it only as a comparison against defence's 10 to 14, not a requirement) | none; the validator (minimum 3) already accepts more if authored | `docs/superpowers/specs/completed/2026-09-09-messaging-m3-store-core-extraction-design.md`, "Casting's validator"; `docs/superpowers/plans/completed/2026-09-09-messaging-m3-store-core-extraction.md` line ~1178 | "The real content problem here... is filed as M6 content work." | open |
| **Crafting** | | | | | | |
| 7 | Recipe Observer (room) lines: item 6 gives `RecipeSpec` two optional keys, `success_room_message` and `failure_room_message` (shipped in item 6 on `feature/messaging-m3-item6-crafting-progression`). No recipe file sets them, so the authoring commit must also register both keys in `textSurfaceRegistry`. Also filed: a generic room-visible fallback line for a player craft when the recipe authors no room line at all (today a player craft with no authored room line stays silent to onlookers) | 252 fields (2 keys x 126 recipes), 0 authored today | none; the slots and validation already exist as of item 6 | `docs/superpowers/specs/completed/2026-09-15-messaging-m3-item6-crafting-progression-design.md`, "What item 6 delivers" and "Out of scope, filed"; `docs/superpowers/plans/completed/2026-09-15-messaging-m3-item6-crafting-progression.md`, Task 9 | "Authoring `success_room_message` / `failure_room_message` for 126 recipes, and any generic player-craft room fallback: M6." | open |
| 8 | Crafting's Actee role (a third party affected by the craft, e.g. enchanting another player's gear) | not counted (feature does not exist yet, so no field count applies) | an "enchant another player's gear" feature, not yet scoped or planned | `docs/superpowers/specs/completed/2026-09-15-messaging-m3-item6-crafting-progression-design.md`, owner ruling and "Out of scope, filed"; `docs/superpowers/plans/completed/2026-09-15-messaging-m3-item6-crafting-progression.md`, Task 9 | Owner ruling (2026-09-15): "Crafting is Actor plus Observer, no Actee today... The door must leave an Actee open for a future 'enchant another player's gear' feature." Deferred beyond M6, to whenever that feature is built | open (not scheduled) |
| **Spells** | | | | | | |
| 9 | Spell Actee text: the target's own personal line when a spell is cast on someone other than the caster does not exist as its own authored field; today only the caster's Actor line and the room's Observer line are authored | not counted (arc spec's original 135-field spell count covers only the Actor/Observer fields that exist today, not a new Actee field) | none; `narration.Roles` already has an `Actee` field, so the mechanism exists, only the store door and the data do not | `docs/superpowers/plans/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration.md` line ~848, code comment: "Actee is empty; the target's own line is authored in M6"; `docs/superpowers/specs/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration-design.md`, "Out of scope, filed" | Same "Out of scope, filed" line as row 1 covers spells and quests together: "Actee text for buffs, spells and quests... M6." | open |
| 10 | Cast/windup line ordering: the observer's line (`skill.cast.go:389`) is inverted relative to the caster's own line on both sides of a cast | not counted | none named | `docs/superpowers/audits/2026-09-07-narration-viewpoint-audit.md` line 566 | "Left alone. Ordering spans the casting system and the per-spell text, so it affects every spell, not this slice." No stage was assigned; recorded here because it is unresolved player-facing wording/ordering the M1 audit chose not to touch | open (no stage assigned) |
| **Quests** | | | | | | |
| 11 | Quest Actee text, and possibly the Actee mechanism itself. The "Out of scope, filed" line groups quests with buffs and spells for M6 authored text, but as shipped in item 5b, `ActionDef`/`QuestReward`'s `Narration()` methods carry only Actor (`send_text`/`playermessage`) and Observer (`room_text`/`roommessage`); no Actee field or `ActionContext` method was added | not counted | Possibly the Actee seam itself (a new `ActionContext` method plus a new quest YAML key), not only the text. See Open Question 2 below | `docs/superpowers/specs/completed/2026-09-08-messaging-m2-shared-narration-seam-design.md`, "The quest bridge's missing actee seam" ruling ("M3's own migration table already schedules 'the Actee slot appears here'... and M6 authors the text"), versus `docs/superpowers/specs/completed/2026-09-12-messaging-m3-item5b-kind-b-store-migration-design.md` design, which shipped no Actee field for quests | Conflict between the M2 ruling's expectation and 5b's shipped design; flagged as an open question rather than resolved here | open, seam status unclear |
| 12 | Quest 14's strongbox fires a second, contradictory line ("The strongbox is open and empty. You have already taken the ledger.") in the same command that grants the item, because the grant lands mid-event and the has-evidence trigger matches too | 1 site | none stated | Memory `project-messaging-m3-item5a-followups.md`, "Parked until the quest arc" | "Fix during or after the quest mechanisms arc." Not M6; the future quest mechanisms arc is itself only named, not yet scoped, per `MEMORY.md`'s roadmap table | open |
| 13 | Three different things are all called "hints" in the codebase (the global broadcast tips, per-dialogue hints, and the per-quest `hint` command); the quest `hint:` text is explicitly carved out of the messaging arc's M3 item 7 and left for the future quest mechanisms arc instead | not counted | none stated | Memory `project-messaging-m3-store-core-design.md`, "THREE different things are called 'hints'" | "This one belongs to the QUEST arc, not the messaging arc." Included here because it is a deferred content ownership question, not because M6 owns it. Update 2026-09-15: M3 item 7 renames the broadcast to "tips" (`tips.yaml`, `set tips`, saved flag migrated in 0.18.0) and leaves dialogue hints unchanged, so after item 7 only the quest `hint` command and dialogue `hints` keep the word | open, not messaging arc's |
| **Combat-messages / hand-rolled Group C sites** | | | | | | |
| 14 | The defended-swing line "dodges your swing, but you still connect" fired three times identically in one fight (observed in play); the pool it comes from has no variety judged | not counted | none | `docs/superpowers/audits/2026-09-07-narration-viewpoint-audit.md` line 568 | "Left alone deliberately: this is core combat narration that M3 and M4 rewrite, and changing the wording now would conflict with that work." Deferred to whenever M3 item 8 (combat-messages) and M4 land; likely an M6 candidate once that store is migrated, but the audit does not name M6 | open |
| 15 | No line announces a corpse appearing after a kill | 0 lines exist today (new content, not a fix) | none | `docs/superpowers/audits/2026-09-07-narration-viewpoint-audit.md` line 569 | "Left alone. Content addition, not a defect in existing text." No stage assigned | open, unscheduled |
| 16 | "You prepare to enter into mortal combat" is sent to a party member who never attacked, because party auto-assist engaged them; the line is accurate but gives no hint that the party's assist, not the player's own action, caused it | 1 site | none | `docs/superpowers/audits/2026-09-07-narration-viewpoint-audit.md` line 565 | "Design question, left for the owner." | open |
| 17 | "You attack the darkness!" is sent for two different failures (no name typed and nothing to inherit, versus a typed name that resolves to nothing), and should read differently for each ("There is nothing here to attack." vs "You do not see \<name\> here.") | 2 failure cases sharing 1 line today | none | Memory `project-attack-the-darkness-messaging.md` | Owner (2026-08-15): "I don't like that messaging either, it should say 'Nothing by that name is in this room' or something close to that." Filed, not fixed. A 2026-09-11 correction in the same memory casts doubt on the file's original secondary claim (that a hidden mob was the cause of a specific 2026-08-15 playtest failure), but the primary defect, one message covering two distinct failures, is unaffected by that correction and remains open | open |
| **Other / prerequisite** | | | | | | |
| 18 | The arc spec (2026-08-31) found `buffs.NightVision` granted by nothing, which would make the M5 crime-in-the-dark ruling turn every unlit room into a free-crime zone. ⚠️ **Partly stale, re-verified 2026-09-15:** the flag is now `conditions.NightVision`, and condition 65 (`conditions/65-cats_eye_draught.yaml`) carries `nightvision` and is granted by item 30047 (`items/consumables-30000/30047-cats_eye_draught.yaml`), so a consumable grants it. Condition 29 (`29-night_vision.yaml`) also carries the flag; no grant source for it was verified. The mutation copy ("You see clearly in the dark.", `internal/mutations/describe.go:128`) still has no mutation granting it. Open part: whether a potion is enough of an escape hatch for the ruling | not counted | An owner decision: grant `NightVision` via a mutation (the copy already exists for that), or restate the M5 crime ruling in terms of room lighting alone | `docs/superpowers/specs/2026-08-31-messaging-unification-design.md`, "BLOCKING FINDING" | "Do not implement the crime gate until that is decided, shipping it as-is would make every unlit room a free-crime zone." This is a prerequisite for M5's ruling, and the unused copy line means part of the content already exists and only needs a grant source | open, blocks M5 |
| 19 | Infrared sight has no in-game way to obtain it (buff 85 exists but nothing grants it outside a synthetic playtest profile); owner suggested making it a real spell | not counted | none named beyond "add it as a spell" | Memory `project-followup-slice-a-names-in-dark.md`, "Owner rulings on the T17 findings" | Owner (2026-09-11): "yeah we probably need to add that as a spell." Would give the shapes-only sight path (and the infrared aiming syntax slice A built) its first real content grant and its first live playtest coverage; today only the synthetic playtest profile can reach it. Not yet scoped | open, not yet scoped |
| **Crafting (continued)** | | | | | | |
| 20 | Crafting: the mob-initiated craft room line (`begins working on something.`, `internal/mobcommands/craft.go:64`) is fixed text with no recipe phase, unlike the instant, multi-round-success and (once authored) recipe-driven mob lines | 1 site | none | `docs/superpowers/plans/completed/2026-09-15-messaging-m3-item6-crafting-progression.md`, Task 9 (plan item 5); plan line 627, "The initiated line (`begins working on something.`) has no recipe phase and is not touched." | Filed, not scoped further; whether it should read from a recipe phase like the other mob craft lines is an open question | open |
| **Progression** | | | | | | |
| 21 | Progression: the `skill-progress` alias (179, `_datafiles/world/dogmud/ansi-aliases.yaml:256`) colors every `CategorySkillProgress` line gold, including progression banners, quest skill-up, recipe-learned and spell-learned lines; the owner may want to retune it | 1 alias | none | `docs/superpowers/plans/completed/2026-09-15-messaging-m3-item6-crafting-progression.md`, Task 9 (plan item 6) | Filed; M3 item 6 shipped the gold coloring as an owner-approved default (owner ruling 2026-09-15), so retuning it is a follow-up, not a defect | open |
| **Tips (the periodic broadcast, formerly "hints")** | | | | | | |
| 22 | Tips are sent as single long lines with no server-side wrap. Each tip is a folded YAML scalar (`>-`), which joins its authored line breaks with spaces, so 56 of the 74 tips exceed 80 characters once tags are stripped (longest 211, plus the 6-character `[Tip] ` prefix). `shouldWrap` returns false for every category (`internal/messaging/pipeline.go`), unlike mob speech, which `internal/mobcommands/say.go` breaks at 80 with `util.SplitStringNL`. Most MUD clients and the web client word-wrap at the window edge; a bare telnet or terminal client may break mid-word. Expected behaviour from source, not observed in a client. The file's header claim that each hint "should fit within 80 characters" was false and is corrected in M3 item 7 | 56 of 74 tips over 80 plain characters | M5's wrap decision (the arc spec lists "the wrap decision" under M5) | `docs/superpowers/specs/2026-09-15-messaging-m3-item7-gossip-tips-design.md`, "Out of scope, filed"; owner question 2026-09-15 ("does this mean the current tips broadcast doesn't wrap properly?") | Owner (2026-09-15) asked for this row. Wrapping tips alone would make them the only server-wrapped long narration, so the fix belongs to the arc-wide wrap decision, not to item 7 | open, M5 |
| **Gossip (the templated NPC broadcast)** | | | | | | |
| 23 | The five `IronwindEvent-Global` / `IronwindEvent-Regional` gossip lines in `_datafiles/world/dogmud/gossip_templates.yaml` are unreachable: no `worldevents.WorldEventType` maps to the key prefix `IronwindEvent` in `eventTypeKey` (`internal/hooks/MobIdle_HandleIdleMobs.go`), so no NPC can ever say them | 5 lines across 2 keys | An Ironwind world event type, or rewriting the lines under an existing key | Added by commit `bea1b3431` (Stage 42.6.6 content); deferred by this plan's Task 7, found while reviewing `gossip.golden` | `gossip.Validate` cannot catch this: the event-type list lives in `internal/hooks`, not the store, so a key can be well-formed and still be dead | open |

## Seen and excluded

Recorded so the reader can check the judgment calls above rather than trust
them blind. None of these are rows because none are missing or wrong player
text waiting to be authored; they are either code/delivery defects, or
explicitly out of the whole arc's scope, or already shipped.

- **The `{source_plain}` anonymizer leak.** An untagged name token that
  `Anonymize` cannot strip, so an infrared observer reads a real name where
  a tagged token would have hidden it. A code/token defect, not missing text;
  held for M5 by the owner in the original arc spec and restated in the 5a
  and 5b specs.
- **M5's other code/doc cleanup items**: the wrap pipeline stage that never
  runs, the `world/default` template-shadowing bug, and
  `internal/messaging/context.md`'s phantom wrap-stage documentation. No
  player-facing text involved.
- **Group A, roughly 2,000 refusal and admin/status strings** ("Zone not
  found," "Usage: caravan reset"). Ruled entirely out of this arc by the
  owner on 2026-09-07 and handed to its own future arc, because they have no
  band, no viewpoint split and no pool for a narration core to help with.
- **Help templates, room descriptions, dialogue trees, quest `description`
  fields, `localize/`.** Deliberately out of scope for the entire arc from
  its first design (2026-08-31): "authored content a player reads on
  request, not events narrated at them."
- **Weather-driven mechanics** (wet conditions affecting items or players).
  A different arc entirely, per the original arc spec.
- **The crime-in-the-dark ruling's delivery mechanism itself** (routing
  crime witnessing through the shared perception verdict). A behavior
  change belonging to M5, not authored text; only its NightVision
  prerequisite (row 18) belongs on this content ledger.
- **Renaming `hints.yaml`'s broadcast to "tips."** A scope/naming decision
  for M3 item 7, not text authoring for M6.
- **40 authored condition start/end notice lines, and the seven previously
  inert consumables.** Both were content gaps once, but both already
  shipped (slice C, PR #124; the consumable wiring, PR #125), so neither
  is a ledger row.
- **Red/aggro name coloring, pet-name leaks on the participant path, GMCP
  bypassing darkness, and a buff trigger line naming a hostile mob to a
  blind reader.** All are code/delivery defects already filed elsewhere in
  project memory, not missing or wrong authored text.
- **`rally` silently spending `warcry`'s cooldown.** A mechanics/design
  question, not a text question.

## Open questions for the owner

1. **Does the arc spec's original 427-field Actee gap (buffs 170 + spells
   135 + quests 122) still hold?** The 2026-08-31 arc spec counted all 427
   as missing Actee text. The 2026-09-12 item 5b design then ruled that a
   buff's holder line already occupies the Actee role, and that what buffs
   actually lack is the Actor (caster) line, addressed by row 1 above (17
   buffs, not 170 fields). The two sources disagree on what buffs'
   contribution to the 427 figure means; the newer source (5b design) should
   likely govern, which would make the real remaining Actee gap
   spells' 135 plus quests' 122, not 427.
2. **Does the quest Actee seam still need to be built before M6 can author
   quest Actee text?** The 2026-09-08 M2 design said M3's migration table
   "schedules the Actee slot appears here" for quests, implying the
   mechanism (an `ActionContext` method, a new YAML key) would exist by the
   time M6 runs. The 2026-09-12 item 5b design, which is that migration
   step, shipped `ActionDef`/`QuestReward` with only Actor and Observer, no
   Actee field or method. It is unclear whether this was a deliberate
   narrowing (row 11) or an oversight.
3. **Do the heal spells share one hardcoded condition the way the two shield
   spells do?** `project-shield-heal-buff-identity.md` verified the shield
   case (Conviction Ward and Chrysalis Cocoon both apply the same hardcoded
   "Minor Shield" condition) but explicitly left the heal-spell version of
   the same question unverified: "Same question applies to the healing
   spells." Row 2 above cannot be sized until this is checked.
