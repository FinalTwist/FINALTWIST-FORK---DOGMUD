# M4e PR 1a — mob special-move narration and sight-surface inventory

Scope: the thirteen mob special-move command files in `internal/mobcommands`
(`attack`, `bash`, `charge`, `drain`, `gore`, `grapple`, `hamstring`, `kick`,
`maul`, `pounce`, `rake`, `shoot`, `throttle`, `trip`) plus the `canSeeInDark`
sight surface this slice retires. This document is the frozen source of truth
for the YAML-authoring task and the byte-identity net test that follow it.
Every format string below was copied verbatim from the file at the stated
line; do not normalize whitespace, punctuation, or wording when consuming it.

## Facts verified against source

| Fact | Verified as |
|---|---|
| `canSeeInDark` definition (mobcommands) | `internal/mobcommands/darkness.go:44`, `func canSeeInDark(u *users.UserRecord, room *rooms.Room) bool` |
| `canSeeInDark` unexported twin (usercommands) | `internal/usercommands/skill_move_defence.go:93`, same signature |
| `sendAudioRoomText` definition | `internal/mobcommands/darkness.go:18` |
| Total `canSeeInDark` text matches, repo-wide | 30 (`grep -rn "canSeeInDark" --include=*.go .`) |
| `messaging.Anonymize(` call sites in mobcommands/usercommands | 12 (howl.go x2, shoot.go x2 mob + x1 user, skill_move_defence.go x2, taunt.go x5) — see Step 2 |
| `sendAudioRoomText(` call sites in mobcommands | 15 (darkness.go 1, howl.go 4, rally.go 1, say.go 1, shout.go 1, taunt.go 6, warcry.go 1) |
| `moveDefenceLines` / `sendMoveDefenceShortage` / `acteeDefenceLine` definitions | `internal/mobcommands/skill_move_defence.go:26,61,74` (a 14th file, NOT one of the 13, shared infrastructure) |
| `GetDamageDescription` definition | `internal/combat/descriptions.go:18` |
| 13-file literal/SendTrio/canSeeInDark counts | reproduced exactly below, from the Step 1 command |

## Step 1 — per-file counts

Command run (repo root):

```bash
for f in attack bash charge drain gore grapple hamstring kick maul pounce rake shoot throttle trip; do
  p="internal/mobcommands/$f.go"
  [ -f "$p" ] || continue
  printf '=== %s: %s backtick literals, %s SendTrio calls, %s canSeeInDark refs\n' \
    "$p" \
    "$(grep -oE '`[^`]*`' "$p" | wc -l)" \
    "$(grep -c 'messaging.SendTrio' "$p")" \
    "$(grep -c 'canSeeInDark' "$p")"
done
```

| File | Backtick literals (raw grep) | SendTrio calls | canSeeInDark refs | Narration rows in this doc | Non-narration artifacts |
|---|---:|---:|---:|---:|---|
| attack.go | 6 | 0 | 1 | 5 | 1 (empty-string literal, code) |
| bash.go | 12 | 5 | 1 | 12 | 0 |
| charge.go | 16 | 6 | 2 | 15 | 1 (backtick-quoted word inside a `//` comment) |
| drain.go | 9 | 4 | 1 | 9 | 0 |
| gore.go | 12 | 5 | 1 | 12 | 0 |
| grapple.go | 6 | 4 | 1 | 6 | 0 |
| hamstring.go | 9 | 4 | 1 | 9 | 0 |
| kick.go | 30 | 11 | 1 | 30 | 0 |
| maul.go | 9 | 4 | 1 | 9 | 0 |
| pounce.go | 12 | 5 | 1 | 12 | 0 |
| rake.go | 9 | 4 | 1 | 9 | 0 |
| shoot.go | 16 | 3 | 1 | 16 | 0 |
| throttle.go | 11 | 5 | 1 | 11 | 0 |
| trip.go | 27 | 10 | 2 | 27 | 0 |
| **Total** | **184** | **70** | **16** | **182** | **2** |

`attack.go` uses `u.SendText` / `room.SendTextVisual` directly and makes zero
`messaging.SendTrio` calls — it does not follow the Actor/Actee/Observer trio
shape the other twelve files use. Flagged for the YAML-authoring task: this
file's "engagement announcement" narration is architecturally distinct from
the special-move hit/miss narration in the other twelve.

The two non-narration artifacts:
- `attack.go:21` — `` `` `` (empty raw string), the comparison target in
  `if rest == \`\` {`. Ordinary Go code, not a narration literal.
- `charge.go:30` — `` `charge` `` inside the comment `// See
  narrateTripWhiffOnProne in trip.go. \`charge\` is the verb most exposed to
  this: ...`. This is markdown-style backtick-quoting of a word inside a `//`
  comment, not a Go string literal. The raw grep count (16) includes it; the
  table below does not.

## Step 2 — sight surface, independently

```
grep -rn "canSeeInDark" --include=*.go . | wc -l          → 30
grep -rn "messaging.Anonymize(" --include=*.go internal/mobcommands internal/usercommands → 12 matches (listed below)
grep -rn "sendAudioRoomText(" --include=*.go internal/mobcommands | wc -l → 15
```

**EXPECTED 30 held.** All 30 `canSeeInDark` text matches, classified:

| Kind | Count | Locations |
|---|---:|---|
| Definitions | 2 | `mobcommands/darkness.go:44`, `usercommands/skill_move_defence.go:93` |
| Comments mentioning the symbol | 4 | `mobcommands/darkness.go:43`, `mobcommands/go.go:97`, `usercommands/skill_move_defence.go:86`, `usercommands/skill_move_defence.go:89` |
| Call sites | 24 | `attack.go:94`, `bash.go:45`, `charge.go:56`, `charge.go:186`, `drain.go:50`, `gore.go:50`, `grapple.go:41`, `hamstring.go:42`, `howl.go:54`, `howl.go:77`, `kick.go:45`, `maul.go:49`, `pounce.go:54`, `rake.go:49`, `shoot.go:85`, `mobcommands/skill_move_defence.go:78`, `taunt.go:71`, `taunt.go:97`, `taunt.go:151`, `taunt.go:184`, `throttle.go:49`, `trip.go:56`, `trip.go:280`, `usercommands/skill_move_defence.go:114` |

**Discrepancy to flag:** the task brief's expected breakdown was "2
definitions, 3 comments, 25 call sites." The total (30) matches and the gate
condition ("if the number differs from 30, STOP") did not fire, so this is
reported rather than blocking. The actual breakdown is 2/4/24: the brief's
comment count appears to have missed `usercommands/skill_move_defence.go:89`
(`// An unexported twin of mobcommands.canSeeInDark. ...`), which is a second
comment line in that file alongside its own doc comment at line 86. This is a
sub-total classification difference only; the governing total of 30 is exact.

`messaging.Anonymize(` call sites (12, none inside the 13 mob files except
`shoot.go`):
`howl.go:61`, `howl.go:84`, `shoot.go:97`, `skill_move_defence.go:79`
(mobcommands), `taunt.go:78`, `taunt.go:104`, `taunt.go:152`, `taunt.go:158`,
`taunt.go:185`, `taunt.go:191`, `usercommands/shoot.go:429`,
`usercommands/skill_move_defence.go:115`, plus two occurrences in
`usercommands/taunt_anonymize_test.go` (test file, not counted in the 12
production call sites above but present in the raw grep).

`sendAudioRoomText(` (15): `darkness.go` (the definition, 1), `howl.go` (4),
`rally.go` (1), `say.go` (1), `shout.go` (1), `taunt.go` (6), `warcry.go` (1).
None of these five files (`howl`, `rally`, `say`, `shout`, `taunt`, `warcry`)
are in the 13-file scope of this slice.

## Step 3 — negative verification

```
grep -l 'util.Rand' internal/mobcommands/{attack,bash,charge,drain,gore,grapple,hamstring,kick,maul,pounce,rake,shoot,throttle,trip}.go
  → (nothing printed, checked across all 13 files individually)
grep -l 'util.Rand' internal/mobcommands/kick.go internal/mobcommands/gore.go ; echo "exit=$?"
  → exit=1
grep -l 'util.Rand' internal/usercommands/kick.go ; echo "exit=$?"
  → internal/usercommands/kick.go
  → exit=0
```

Result: **no mob special-move file in the 13-file set pools text via
`util.Rand`** (every file was checked individually and in the two-file
group; all returned nothing / exit 1). The second command proves the search
was capable of finding a pooling call — `internal/usercommands/kick.go` (the
player-facing kick command, not part of this slice) does pool via
`util.Rand` and the grep found it (exit 0). The negative is proven capable of
failing.

## Per-literal inventory

Columns: line, outcome branch (variant where relevant), proposed event key,
role, dark_twin, the Go format string verbatim, and the ordered `fmt.Sprintf`
argument list (or "—" when the literal is passed directly with no
`Sprintf`). All literals use `messaging.CategoryX` category constants noted
inline only where they differ from the file's dominant category.

### attack.go (5 narration rows; 0 SendTrio calls — SendText/SendTextVisual only)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 95 | player-target engage, seen | `engage` | actee | no | `` `<ansi fg="mobname">%s</ansi> prepares to fight you!` `` | `mob.Character.Name` |
| 97 | player-target engage, unseen | `engage` | actee | yes | `` `Something prepares to fight you!` `` | — |
| 101 | player-target engage, room broadcast (excludes `u`) | `engage` | observer | no | `` `<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="username">%s</ansi>` `` | `mob.Character.Name, u.Character.Name` |
| 151 | mob-vs-mob engage, room broadcast (no player, no darkness branch) | `engage` | observer | no | `` `<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>` `` | `mob.Character.Name, m.Character.Name` |
| 161 | no resolvable target, fallback idle broadcast | `idle_confused` | observer | no | `` `<ansi fg="mobname">%s</ansi> looks confused and upset.` `` | `mob.Character.Name` |

Category: 95/97/101 use `messaging.CategoryHitMelee`; 151 uses
`messaging.CategoryHitMelee`; 161 uses `messaging.CategoryMobEmote`.

### bash.go (12 rows)

Label substitution variables (Go string vars, not backtick literals):
`bashLabel` = `"shield bash"` default / `"crushing slam"` if
`species.NaturalBash`; `bashVerb` = `"bashes"` / `"slams into"`; `bashWith` =
`"with their shield"` / `"with tremendous force"`.

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 77 | knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, bashLabel, dmgDesc` |
| 79 | knockdown, unseen | `knockdown` | actee | yes | `` `Something's <ansi fg="yellow-bold">%s</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `bashLabel, dmgDesc` |
| 86 | knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> knocks <ansi fg="username">%s</ansi> to the ground!` `` | `mobName, bashLabel, target.Name` |
| 92 | hit (no kd), seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, bashLabel, dmgDesc` |
| 94 | hit (no kd), unseen | `hit` | actee | yes | `` `Something's <ansi fg="yellow-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)` `` | `bashLabel, dmgDesc` |
| 101 | hit (no kd), observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> %s <ansi fg="username">%s</ansi> %s!` `` | `mobName, bashVerb, target.Name, bashWith` |
| 108 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">%s</ansi> fails to floor you, but still crashes into you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, bashLabel, dmgDesc` |
| 110 | partial, unseen | `partial` | actee | yes | `` `Something's <ansi fg="yellow-bold">%s</ansi> fails to floor you, but still crashes into you! (<ansi fg="damage">%s</ansi>)` `` | `bashLabel, dmgDesc` |
| 115 | partial, observer (default; overridden by `defence.ToRoom` when defended) | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> %s <ansi fg="username">%s</ansi> %s, who staggers but stays up!` `` | `mobName, bashVerb, target.Name, bashWith` |
| 138 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> attempts a %s, but misses!` `` | `mobName, bashLabel` |
| 140 | miss, unseen | `miss` | actee | yes | `` `Something attempts a %s, but misses!` `` | `bashLabel` |
| 147 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> attempts to bash <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Note: line 147's observer-miss line hardcodes the word "bash" rather than
using `bashVerb`, unlike lines 101/115 which do substitute it. Pre-existing
asymmetry in the source; reproduce as-is, do not "fix" during migration.

Sourced elsewhere (no table row): line 117 `defence.ToRoom` (partial-observer
override when defended), lines 131–132 `defence.ToDefender` /
`defence.ToRoom` (defended-outright branch). Origin: `combat.RenderChannelDefenceMessages`
via `moveDefenceLines` in `internal/mobcommands/skill_move_defence.go`.

### charge.go (15 rows; 1 non-narration comment artifact at line 30)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 79 | knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 81 | knockdown, unseen | `knockdown` | actee | yes | `` `Something charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 88 | knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges and slams into <ansi fg="username">%s</ansi>, sending them sprawling!` `` | `mobName, targetName` |
| 94 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 96 | hit, unseen | `hit` | actee | yes | `` `Something charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 103 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges at <ansi fg="username">%s</ansi>, but they keep their footing!` `` | `mobName, targetName` |
| 110 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges and you dodge the worst of it, but the impact still clips you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 112 | partial, unseen | `partial` | actee | yes | `` `Something charges and you dodge the worst of it, but the impact still clips you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 117 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges at <ansi fg="username">%s</ansi>, who mostly dodges but still gets clipped!` `` | `mobName, targetName` |
| 138 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges past you, missing entirely!` `` | `mobName` |
| 140 | miss, unseen | `miss` | actee | yes | `` `Something charges past you, missing entirely!` `` | — |
| 147 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges past <ansi fg="username">%s</ansi>, missing entirely!` `` | `mobName, targetName` |
| 188 | whiff-on-prone (target already down), seen — `narrateChargeWhiffOnProne` | `whiff_on_prone` | actee | no | `` `<ansi fg="mobname">%s</ansi> thunders in, but you are already down, and it overruns you.` `` | `mobName` |
| 191 | whiff-on-prone, unseen | `whiff_on_prone` | actee | yes | `` `Something thunders in, but you are already down, and it overruns you.` `` | — |
| 199 | whiff-on-prone, observer | `whiff_on_prone` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges <ansi fg="username">%s</ansi>, who is already down, and overruns them.` `` | `mobName, target.Name` |

`whiff_on_prone` is shared with `trip.go`'s `narrateTripWhiffOnProne` — the
source comment states charge's version is explicitly "the charge-flavoured
sibling," so this key applies across both verbs even though the wording
differs per verb (that per-verb wording difference is expected and correct;
the event key names the semantic slot, not the sentence).

Sourced elsewhere: line 119 `defence.ToRoom`, lines 131–132
`defence.ToDefender` / `defence.ToRoom` (defended-outright, attack label fed
to the renderer is the plain string `"charge"`, not a backtick literal).

### drain.go (9 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 71 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> plunges into you, sapping your vitality and leeching your life-force! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 73 | hit, unseen | `hit` | actee | yes | `` `Something plunges into you, sapping your vitality and leeching your life-force! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 80 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> plunges into <ansi fg="username">%s</ansi> and leeches their vitality!` `` | `mobName, target.Name` |
| 89 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> reaches for you and you slip most of its grip, but it still catches a sliver of you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 91 | partial, unseen | `partial` | actee | yes | `` `Something reaches for you and you slip most of its grip, but it still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 96 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> reaches for <ansi fg="username">%s</ansi> hungrily, who slips mostly free but still gets caught!` `` | `mobName, target.Name` |
| 121 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> reaches for you hungrily, but misses!` `` | `mobName` |
| 123 | miss, unseen | `miss` | actee | yes | `` `Something reaches for you hungrily, but misses!` `` | — |
| 130 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> reaches for <ansi fg="username">%s</ansi> hungrily, but misses!` `` | `mobName, target.Name` |

No knockdown branch exists in this file (`result.KnockedDown` is never
checked). Note the dark-twin at line 91 drops "a sliver of" versus the seen
line 89 ("catches a sliver of you" → "catches you"); reproduce verbatim.

Sourced elsewhere: line 98 `defence.ToRoom`, lines 114–115
`defence.ToDefender` / `defence.ToRoom` (attack label `"draining grasp"`).

### gore.go (12 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 73 | knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi> lowers its head and charges into you, driving you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 75 | knockdown, unseen | `knockdown` | actee | yes | `` `Something charges into you with bone-jarring force, hurling you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 82 | knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges into <ansi fg="username">%s</ansi> and hurls them to the ground!` `` | `mobName, target.Name` |
| 89 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> drives its horns into you with a powerful charge! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 91 | hit, unseen | `hit` | actee | yes | `` `Something drives into you with a powerful charge! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 98 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> drives its horns into <ansi fg="username">%s</ansi>!` `` | `mobName, target.Name` |
| 105 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges you and you sidestep most of it, but the horns still catch you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 107 | partial, unseen | `partial` | actee | yes | `` `Something charges you and you sidestep most of it, but it still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 112 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges <ansi fg="username">%s</ansi>, who mostly dodges but still gets grazed by the horns!` `` | `mobName, target.Name` |
| 135 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> charges at you, but you sidestep the gore!` `` | `mobName` |
| 137 | miss, unseen | `miss` | actee | yes | `` `Something charges at you, but you sidestep!` `` | — |
| 144 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> charges at <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Note: dark-twin at line 137 drops "the gore" (generic "you sidestep!" vs
"you sidestep the gore!"); verbatim, not to fix.

Sourced elsewhere: line 114 `defence.ToRoom`, lines 128–129
`defence.ToDefender` / `defence.ToRoom` (attack label `"goring charge"`).

### grapple.go (6 rows — success/fail model, no damage, no knockdown)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 63 | success, seen | `success` | actee | no | `` `<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!` `` | `mobName, result.PositionDesc` |
| 65 | success, unseen | `success` | actee | yes | `` `Something <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!` `` | `result.PositionDesc` |
| 72 | success, observer | `success` | observer | no | `` `<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="username">%s</ansi> into <ansi fg="cyan">%s</ansi> position!` `` | `mobName, targetName, result.PositionDesc` |
| 91 | fail, seen | `fail` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to grapple you, but you slip away!` `` | `mobName` |
| 93 | fail, unseen | `fail` | actee | yes | `` `Something tries to grapple you, but you slip away!` `` | — |
| 100 | fail, observer | `fail` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to grapple <ansi fg="username">%s</ansi>, but fails!` `` | `mobName, targetName` |

**Judgment call to flag:** grapple has no damage roll, so it never enters the
hit/knockdown/partial/miss shape the other twelve files share. I keyed its
two outcomes `success` / `fail` (mirroring the Go field `result.Success`)
rather than forcing them into `hit`/`miss`. This is the one verb whose
outcome vocabulary does not overlay the shared key set — confirm this
reading before the YAML author copies it.

Sourced elsewhere: line 79 `result.DisarmResult.TargetMsg` (actee, a world
event carried on grapple success), line 84 `result.DisarmResult.RoomMessage`
(observer), line 107 `result.CritFailure.TargetMessage` (actee, world event
on a grapple critical failure), line 112 `result.CritFailure.RoomMessage`
(observer). Origin: `internal/actions` (`ExecuteGrapple`'s result struct),
not this file. Do not invent wording for these.

### hamstring.go (9 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 65 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 67 | hit, unseen | `hit` | actee | yes | `` `Something rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 74 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> lunges low and rakes its fangs across <ansi fg="username">%s</ansi>'s legs!` `` | `mobName, target.Name` |
| 80 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> lunges at your legs and you dodge most of it, but the fangs still catch you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 82 | partial, unseen | `partial` | actee | yes | `` `Something lunges at your legs and you dodge most of it, but it still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 87 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> lunges at <ansi fg="username">%s</ansi>'s legs, who mostly dodges but still gets caught!` `` | `mobName, target.Name` |
| 110 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> lunges at your legs, but you sidestep the attack!` `` | `mobName` |
| 112 | miss, unseen | `miss` | actee | yes | `` `Something lunges at your legs, but you sidestep the attack!` `` | — |
| 119 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> lunges at <ansi fg="username">%s</ansi>'s legs, but misses!` `` | `mobName, target.Name` |

No knockdown branch. Sourced elsewhere: line 89 `defence.ToRoom`, lines
103–104 `defence.ToDefender` / `defence.ToRoom` (attack label `"hamstring slash"`).

### kick.go (30 rows — three variants: standard, stomp, knee)

Standard variant (only variant with a knockdown branch):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 110 | standard knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi>'s powerful <ansi fg="yellow-bold">kick</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 112 | standard knockdown, unseen | `knockdown` | actee | yes | `` `Something's powerful <ansi fg="yellow-bold">kick</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 119 | standard knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> kicks <ansi fg="username">%s</ansi>, knocking them to the ground!` `` | `mobName, target.Name` |
| 125 | standard hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> kicks you hard! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 127 | standard hit, unseen | `hit` | actee | yes | `` `Something kicks you hard! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 134 | standard hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> kicks <ansi fg="username">%s</ansi>!` `` | `mobName, target.Name` |
| 190 | standard partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> attempts to kick you, and you slip most of it, but the boot still clips you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 192 | standard partial, unseen | `partial` | actee | yes | `` `Something attempts to kick you, and you slip most of it, but it still clips you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 197 | standard partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> swings a kick at <ansi fg="username">%s</ansi>, who mostly dodges but still gets clipped!` `` | `mobName, target.Name` |
| 255 | standard miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> attempts to kick you, but misses!` `` | `mobName` |
| 257 | standard miss, unseen | `miss` | actee | yes | `` `Something attempts to kick you, but misses!` `` | — |
| 264 | standard miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> attempts to kick <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Stomp variant (`actions.KickStomp`; hit/partial/miss only):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 77 | stomp hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> stomps on you while you're down! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 79 | stomp hit, unseen | `hit` | actee | yes | `` `Something stomps on you while you're down! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 86 | stomp hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> stomps on the downed <ansi fg="username">%s</ansi>!` `` | `mobName, target.Name` |
| 146 | stomp partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to stomp you, and you roll aside, but the heel still catches you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 148 | stomp partial, unseen | `partial` | actee | yes | `` `Something tries to stomp you, and you roll aside, but the heel still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 153 | stomp partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to stomp <ansi fg="username">%s</ansi>, who rolls mostly clear but still gets caught!` `` | `mobName, target.Name` |
| 223 | stomp miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to stomp you, but you roll aside!` `` | `mobName` |
| 225 | stomp miss, unseen | `miss` | actee | yes | `` `Something tries to stomp you, but you roll aside!` `` | — |
| 232 | stomp miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to stomp <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Knee variant (`actions.KickKnee`; hit/partial/miss only):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 93 | knee hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> drives a knee into you in the grapple! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 95 | knee hit, unseen | `hit` | actee | yes | `` `Something drives a knee into you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 102 | knee hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> drives a knee into <ansi fg="username">%s</ansi>!` `` | `mobName, target.Name` |
| 168 | knee partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to knee you, and you block most of it, but it still lands! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 170 | knee partial, unseen | `partial` | actee | yes | `` `Something tries to knee you, and you block most of it, but it still lands! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 175 | knee partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to knee <ansi fg="username">%s</ansi> in the grapple, who blocks most of it but still takes the hit!` `` | `mobName, target.Name` |
| 239 | knee miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to knee you, but you block it!` `` | `mobName` |
| 241 | knee miss, unseen | `miss` | actee | yes | `` `Something tries to knee you, but you block it!` `` | — |
| 248 | knee miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to knee <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

`attackName` (used only as the label fed to `moveDefenceLines`, not
narration) is `"kick"` / `"stomp"` / `"knee strike"` per variant.

Sourced elsewhere: lines 155, 177, 199 `defence.ToRoom` (per-variant partial
observer overrides), lines 214–215 `defence.ToDefender` / `defence.ToRoom`
(one shared defended-outright branch for all three variants, keyed by
`attackName`).

### maul.go (9 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 70 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> savages you with vicious fangs, tearing bleeding wounds! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 72 | hit, unseen | `hit` | actee | yes | `` `Something savages you with vicious fangs, tearing bleeding wounds! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 79 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> savages <ansi fg="username">%s</ansi> with vicious fangs!` `` | `mobName, target.Name` |
| 85 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> snaps at you and you dodge most of it, but the fangs still tear you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 87 | partial, unseen | `partial` | actee | yes | `` `Something snaps at you and you dodge most of it, but the fangs still tear you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 92 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, who twists mostly free but still gets torn!` `` | `mobName, target.Name` |
| 115 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> snaps its fangs at you, but misses!` `` | `mobName` |
| 117 | miss, unseen | `miss` | actee | yes | `` `Something snaps its fangs at you, but misses!` `` | — |
| 124 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

No knockdown branch. Sourced elsewhere: line 94 `defence.ToRoom`, lines
108–109 `defence.ToDefender` / `defence.ToRoom` (attack label `"savage bite"`).

### pounce.go (12 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 77 | knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi> leaps at you and slams you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 79 | knockdown, unseen | `knockdown` | actee | yes | `` `Something leaps at you and slams you to the ground! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 86 | knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> leaps at <ansi fg="username">%s</ansi> and slams them to the ground!` `` | `mobName, target.Name` |
| 93 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> springs at you and crashes into your body! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 95 | hit, unseen | `hit` | actee | yes | `` `Something springs at you and crashes into your body! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 102 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> springs at <ansi fg="username">%s</ansi> and crashes into them!` `` | `mobName, target.Name` |
| 109 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> leaps at you and you sidestep most of it, but it still clips you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 111 | partial, unseen | `partial` | actee | yes | `` `Something leaps at you and you sidestep most of it, but it still clips you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 116 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> leaps at <ansi fg="username">%s</ansi>, who mostly dodges but still gets clipped!` `` | `mobName, target.Name` |
| 139 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> leaps at you, but you sidestep the pounce!` `` | `mobName` |
| 141 | miss, unseen | `miss` | actee | yes | `` `Something leaps at you, but you sidestep!` `` | — |
| 148 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> leaps at <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Sourced elsewhere: line 118 `defence.ToRoom`, lines 132–133
`defence.ToDefender` / `defence.ToRoom` (attack label `"pounce"`).

### rake.go (9 rows)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 70 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> rakes its claws across you, opening bleeding wounds! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 72 | hit, unseen | `hit` | actee | yes | `` `Something rakes its claws across you, opening bleeding wounds! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 79 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> rakes its claws across <ansi fg="username">%s</ansi>!` `` | `mobName, target.Name` |
| 85 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> swipes its claws and you dodge most of it, but they still scratch you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 87 | partial, unseen | `partial` | actee | yes | `` `Something swipes at you and you dodge most of it, but it still scratches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 92 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> swipes its claws at <ansi fg="username">%s</ansi>, who mostly dodges but still gets scratched!` `` | `mobName, target.Name` |
| 115 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> swipes its claws at you, but misses!` `` | `mobName` |
| 117 | miss, unseen | `miss` | actee | yes | `` `Something swipes its claws at you, but misses!` `` | — |
| 124 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> swipes its claws at <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, target.Name` |

Note: line 87's dark-twin says "swipes at you" where the seen line 85 says
"swipes its claws"; verbatim, not to fix. No knockdown branch.

Sourced elsewhere: line 94 `defence.ToRoom`, lines 108–109
`defence.ToDefender` / `defence.ToRoom` (attack label `"claw rake"`).

### shoot.go (16 rows — structurally distinct from the melee twelve)

`shoot.go` does not use the `canSee`-branched two-full-sentence pattern the
melee files use. It builds one shared sentence template and substitutes a
`shooter` name token (`mobName`-colored fragment, or the literal `Someone`
when `anonymous` — `result.IsSneaking || !canSeeInDark(u, room)`). It also
performs a genuine cross-room send: the target's room gets a *second*,
separate `SendTrio` whose only populated line is `Observer`, sent against a
distinct `messaging.Audience{Room: tr}` — this is the concrete case of the
`remote_observer` role named in the task brief.

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 49 | name-color fragment (mob) | `name_fragment` | n/a | no | `` `<ansi fg="mobname">%s</ansi>` `` | `mob.Character.Name` |
| 50 | name-color fragment (weapon) | `name_fragment` | n/a | no | `` `<ansi fg="itemname">%s</ansi>` `` | `result.WeaponName` |
| 52 | name-color fragment (target, mob-colored default) | `name_fragment` | n/a | no | `` `<ansi fg="mobname">%s</ansi>` `` | `result.TargetName` |
| 54 | name-color fragment (target, player-colored override) | `name_fragment` | n/a | no | `` `<ansi fg="username">%s</ansi>` `` | `result.TargetName` |
| 87 | anonymous shooter-name substitution | `anon_shooter_name` | actee (fragment) | n/a* | `` `Someone` `` | — |
| 91 | hit, actee (same-room and cross-room share this line) | `hit` | actee | n/a* | `` `%s's shot strikes you!` `` | `shooter` |
| 93 | partial, actee | `partial` | actee | n/a* | `` `%s's shot goes wide, but the edge of it still clips you!` `` | `shooter` |
| 101 | miss (default case), actee | `miss` | actee | n/a* | `` `%s's shot narrowly misses you!` `` | `shooter` |
| 127 | same-room fire announce, observer (default; overridden by `triadRoom` when defended) | `fire_announce` | observer | no | `` `%s fires their %s at %s!` `` | `mobName, weapon, targetColored` |
| 142 | cross-room depart, observer (shooter's own room) | `fire_depart` | observer | no | `` `%s fires their %s %sward.` `` | `mobName, weapon, result.ExitName` |
| 155 | cross-room arrival origin fragment, unknown exit | `origin_unknown` | remote_observer (fragment) | no | `` `from somewhere nearby` `` | — |
| 157 | cross-room arrival origin fragment, known exit | `origin_known` | remote_observer (fragment) | no | `` `from beyond the <ansi fg="exit">%s</ansi>` `` | `fromDir` |
| 162 | cross-room arrival verb fragment, hit | `hit` | remote_observer (fragment) | no | `` `and strikes` `` | — |
| 164 | cross-room arrival verb fragment, partial | `partial` | remote_observer (fragment) | no | `` `and clips` `` | — |
| 166 | cross-room arrival verb fragment, miss (default) | `miss` | remote_observer (fragment) | no | `` `and narrowly misses` `` | — |
| 169 | cross-room arrival template, sent to the TARGET's room as a separate `SendTrio`/`Audience{Room: tr}` | `arrival` | remote_observer | no | `` `A shot streaks in %s %s %s!` `` | `origin, verb, targetColored` |

\* `dark_twin` is marked `n/a` for lines 87/91/93/101: this file does not
pick between two alternate full-sentence literals the way the other twelve
do. It keeps ONE literal and swaps the `shooter` argument between the
mob-name fragment and the literal `Someone`. **Flag for the YAML author:**
if the migration wants a `dark_twin` axis for shoot.go's personal lines, it
has to be modeled as an argument substitution, not a second event variant —
this is a structural difference from every other file in this inventory and
is the most likely place for a design mismatch.

Sourced elsewhere: lines 62–66, `triadDef` / `triadRoom` via
`combat.RenderChannelDefenceMessages` (attack label is the plain
double-quoted string `"aimed shot"`, not a backtick literal — a label
parameter like `bashLabel`/`attackName` elsewhere, not narration text
itself). Line 80 `combat.ChannelDefenceShortageText(...)` (a plain
`u.SendText`, not part of any Trio). Line 97,
`messaging.Anonymize(personal)` — this programmatically anonymizes the
elsewhere-sourced `triadDef` text; it is not a dark-twin literal owned by
this file.

### throttle.go (11 rows — includes a unique `cast_interrupt` sub-branch)

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 70 | hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> clamps crushing fangs around your throat, cutting off your air! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 72 | hit, unseen | `hit` | actee | yes | `` `Something crushes your throat with savage fangs! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 79 | hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> clamps crushing fangs around <ansi fg="username">%s</ansi>'s throat!` `` | `mobName, target.Name` |
| 88 | cast-interrupt detail (nested in hit, only if `res.InterruptedCast`), seen | `cast_interrupt` | actee | no | `` `<ansi fg="mobname">%s</ansi>'s choke shatters your concentration — your spell collapses!` `` | `mobName` |
| 91 | cast-interrupt detail, unseen | `cast_interrupt` | actee | yes | `` `The crushing grip shatters your concentration — your spell collapses!` `` | — |
| 103 | partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> lunges for your throat and you pull mostly free, but the fangs still catch you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 105 | partial, unseen | `partial` | actee | yes | `` `Something lunges for your throat and you pull mostly free, but it still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 110 | partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> lunges for <ansi fg="username">%s</ansi>'s throat, who pulls mostly free but still gets grazed!` `` | `mobName, target.Name` |
| 133 | miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> lunges for your throat but misses!` `` | `mobName` |
| 135 | miss, unseen | `miss` | actee | yes | `` `Something snaps at your throat but misses!` `` | — |
| 142 | miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> lunges for <ansi fg="username">%s</ansi>'s throat but misses!` `` | `mobName, target.Name` |

`cast_interrupt` (lines 88/91) is unique to `throttle.go` — no other of the
13 files has this sub-branch. It uses `messaging.CategorySystem` (not
`CategoryHitNaturalSharp` like the rest of the file) and its `Observer` line
is explicitly `messaging.NoLine` (never broadcast to the room) — there is no
observer row for this event by design, not by omission.

Note: line 135's dark-twin says "snaps at your throat" where the seen line
133 says "lunges for your throat"; verbatim, not to fix. No knockdown
branch.

Sourced elsewhere: line 112 `defence.ToRoom`, lines 126–127
`defence.ToDefender` / `defence.ToRoom` (attack label `"throttle lunge"`).

### trip.go (27 rows — two variants: trip, tailsweep; both have knockdown; plus whiff_on_prone shared with charge.go)

Tailsweep variant (`res.Variant == actions.TripTailsweep`):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 80 | tailsweep knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi> hammers you with their tail, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 82 | tailsweep knockdown, unseen | `knockdown` | actee | yes | `` `Something hammers you with a powerful sweep, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 89 | tailsweep knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> tailsweeps <ansi fg="username">%s</ansi>, sending them crashing to the ground!` `` | `mobName, targetName` |
| 95 | tailsweep hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> sweeps at you with their tail, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 97 | tailsweep hit, unseen | `hit` | actee | yes | `` `Something sweeps at you powerfully, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 104 | tailsweep hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> tailsweeps <ansi fg="username">%s</ansi>, but they keep their footing!` `` | `mobName, targetName` |
| 155 | tailsweep partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> swings their tail and you keep your feet, but it still cracks into you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 157 | tailsweep partial, unseen | `partial` | actee | yes | `` `Something sweeps at you and you keep your feet, but it still cracks into you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 161 | tailsweep partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> tailsweeps <ansi fg="username">%s</ansi>, who staggers but keeps their feet!` `` | `mobName, targetName` |
| 211 | tailsweep miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> swings their tail at you, but you avoid it!` `` | `mobName` |
| 213 | tailsweep miss, unseen | `miss` | actee | yes | `` `Something sweeps at you powerfully, but you avoid it!` `` | — |
| 220 | tailsweep miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> attempts a tailsweep on <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, targetName` |

Trip variant (`!hasTail`):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 112 | trip knockdown, seen | `knockdown` | actee | no | `` `<ansi fg="mobname">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 114 | trip knockdown, unseen | `knockdown` | actee | yes | `` `Something sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 121 | trip knockdown, observer | `knockdown` | observer | no | `` `<ansi fg="mobname">%s</ansi> trips <ansi fg="username">%s</ansi>, sending them crashing to the ground!` `` | `mobName, targetName` |
| 127 | trip hit, seen | `hit` | actee | no | `` `<ansi fg="mobname">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 129 | trip hit, unseen | `hit` | actee | yes | `` `Something attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 136 | trip hit, observer | `hit` | observer | no | `` `<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but they keep their footing!` `` | `mobName, targetName` |
| 174 | trip partial, seen | `partial` | actee | no | `` `<ansi fg="mobname">%s</ansi> tries to trip you and you keep your feet, but the sweep still catches you! (<ansi fg="damage">%s</ansi>)` `` | `mobName, dmgDesc` |
| 176 | trip partial, unseen | `partial` | actee | yes | `` `Something tries to trip you and you keep your feet, but the sweep still catches you! (<ansi fg="damage">%s</ansi>)` `` | `dmgDesc` |
| 180 | trip partial, observer | `partial` | observer | no | `` `<ansi fg="mobname">%s</ansi> tries to trip <ansi fg="username">%s</ansi>, who staggers but keeps their feet!` `` | `mobName, targetName` |
| 226 | trip miss, seen | `miss` | actee | no | `` `<ansi fg="mobname">%s</ansi> attempts to trip you, but you avoid it!` `` | `mobName` |
| 228 | trip miss, unseen | `miss` | actee | yes | `` `Something attempts to trip you, but you avoid it!` `` | — |
| 235 | trip miss, observer | `miss` | observer | no | `` `<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but misses!` `` | `mobName, targetName` |

Whiff-on-prone (`narrateTripWhiffOnProne`, shared key with `charge.go`,
resolved BEFORE variant selection so it applies to both variants alike):

| Line | Branch | Event key | Role | Dark twin | Format string | Args |
|---:|---|---|---|:---:|---|---|
| 282 | whiff-on-prone, seen | `whiff_on_prone` | actee | no | `` `<ansi fg="mobname">%s</ansi> rushes at you and finds only the ground you are already on.` `` | `mobName` |
| 285 | whiff-on-prone, unseen | `whiff_on_prone` | actee | yes | `` `Something rushes past you and finds only the ground you are already on.` `` | — |
| 293 | whiff-on-prone, observer | `whiff_on_prone` | observer | no | `` `<ansi fg="mobname">%s</ansi> rushes at <ansi fg="username">%s</ansi>, who is already down, and carries straight past.` `` | `mobName, target.Name` |

`tripAttack` (label fed to `moveDefenceLines`, not narration) is `"trip"` /
`"tailsweep"` per variant.

Sourced elsewhere: line 163 `defence.ToRoom` (tailsweep partial-observer
override), line 182 `defence.ToRoom` (trip partial-observer override), lines
204–205 `defence.ToDefender` / `defence.ToRoom` (one shared defended-outright
branch for both variants).

## Consolidated "sourced from elsewhere" list

Do not author wording for any of these; the YAML task must leave them as
pass-throughs to their existing Go source.

| Expression | Files | Origin |
|---|---|---|
| `defence.ToRoom`, `defence.ToDefender`, `defence.ToAttacker` | bash, charge, drain, gore, hamstring, kick, maul, pounce, rake, throttle, trip | `combat.RenderChannelDefenceMessages`, called from `moveDefenceLines` in `internal/mobcommands/skill_move_defence.go:42` |
| `lines.Shortage` / `combat.ChannelDefenceShortageText(...)` | bash, charge, drain, gore, hamstring, kick, maul, pounce, rake, shoot, throttle, trip | `internal/combat`, surfaced via `sendMoveDefenceShortage` (a plain `SendText`, not part of any Trio) |
| `result.DisarmResult.TargetMsg`, `result.DisarmResult.RoomMessage` | grapple | `internal/actions` (`ExecuteGrapple` result) |
| `result.CritFailure.TargetMessage`, `result.CritFailure.RoomMessage` | grapple | `internal/actions` (`ExecuteGrapple` result) |
| `triadDef`, `triadRoom` (via `combat.RenderChannelDefenceMessages`) | shoot | `internal/combat`, label param `"aimed shot"` |
| `messaging.Anonymize(personal)` applied to `triadDef` | shoot | programmatic transform of the above, not a literal owned by shoot.go |

Attack-name/verb label parameters (plain double-quoted Go strings, NOT
backtick literals, fed into the shared defence renderer as its `attack`
argument — these select which stock sentence `RenderChannelDefenceMessages`
produces, they are not narration text authored in the 13 files): `"charge"`
(charge.go), `"draining grasp"` (drain.go), `"goring charge"` (gore.go),
`"hamstring slash"` (hamstring.go), `"kick"`/`"stomp"`/`"knee strike"`
(kick.go, via `attackName`), `"savage bite"` (maul.go), `"pounce"`
(pounce.go), `"claw rake"` (rake.go), `"throttle lunge"` (throttle.go),
`"trip"`/`"tailsweep"` (trip.go, via `tripAttack`), `"aimed shot"`
(shoot.go). Out of scope for the literal table (Step 1's grep was
backtick-only); listed here so the YAML author does not mistake them for
narration prose.

## Event key vocabulary

`knockdown`, `hit`, `partial`, `miss` — shared across bash, charge, drain,
gore, hamstring, kick (x3 variants), maul, pounce, rake, throttle, trip (x2
variants). Not every verb has every key (drain, hamstring, maul, rake,
throttle have no `knockdown`; kick's stomp/knee variants have no
`knockdown`).

`whiff_on_prone` — shared by charge and trip only (charge's own comment
calls it "the charge-flavoured sibling" of trip's version).

Verb-local keys, each used by exactly one file: `engage` / `idle_confused`
(attack.go), `success` / `fail` (grapple.go), `cast_interrupt`
(throttle.go), `fire_announce` / `fire_depart` / `origin_unknown` /
`origin_known` / `arrival` / `anon_shooter_name` / `name_fragment`
(shoot.go — `hit`/`partial`/`miss` are reused for shoot's own actee and
remote_observer lines rather than inventing new keys, since they are the
same semantic outcome).

## Judgment calls and flags for the next task

1. **attack.go is not a special-move file in the SendTrio sense.** Zero
   `SendTrio` calls; it talks to `u`/`room` directly. Its narration
   (`engage`, `idle_confused`) still darkness-branches via `canSeeInDark`
   but through `u.SendText`/`room.SendTextVisual`, not the
   Actor/Actee/Observer trio the other twelve files share. The YAML store
   design needs to decide whether this file's five lines fit the same
   schema or need a variant shape.
2. **grapple.go's outcome vocabulary is `success`/`fail`, not
   `hit`/`miss`.** No damage roll, no knockdown, no partial. Treating it as
   a `hit`/`miss` verb would be a false generalization; flagged above.
3. **shoot.go's darkness handling is an argument substitution
   (`Someone` vs. a name fragment) inside ONE literal, not two alternate
   full-sentence literals.** Every other file in this set represents its
   dark twin as a second, separately-authored sentence. If the YAML schema
   assumes "every event has an optional dark-twin sentence," shoot.go's
   personal lines (91/93/101) don't fit that shape without a rework — they
   need a "who is the actor called" substitution slot instead.
4. **kick.go and trip.go both carry a `variant` axis** (stomp/knee/standard;
   tailsweep/trip) orthogonal to the event key. The event key alone
   (`hit`/`partial`/`miss`/`knockdown`) is not a unique identifier within
   these two files — variant plus key together is.
5. Several dark-twin pairs are NOT word-for-word symmetric with their seen
   counterpart (drain.go:91 drops "a sliver of", gore.go:137 drops "the
   gore", rake.go:87 changes the verb, throttle.go:135 changes the verb).
   These are pre-existing asymmetries in the shipped Go source, reproduced
   here verbatim per the no-fixing-typos instruction. The byte-identity net
   test must pin the asymmetry as-is, not "correct" it.
6. bash.go:147's miss-observer line hardcodes "bash" instead of using
   `bashVerb` the way its sibling branches (101, 115) do — same
   as-is-not-a-bug note.
