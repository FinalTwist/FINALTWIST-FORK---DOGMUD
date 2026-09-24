#!/usr/bin/env python3
"""Extract the pre-migration mob special-move narration literals into a
fixture the byte-identity net test compares against the shipped YAML store.

WHAT THIS DOES

For the twelve mob special-move files that have a corresponding
_datafiles/world/dogmud/narration/special-moves/<verb>.yaml (bash, charge,
drain, gore, grapple, hamstring, kick, maul, pounce, rake, throttle, trip)
plus shoot.go (handled separately below, see SHOOT), this script:

  1. Parses docs/superpowers/audits/2026-09-21-m4e1-site-inventory.md's
     per-literal tables to get, per (verb, line): the event key, the role,
     and whether the literal is a dark twin. This is the SEMANTIC
     classification, and it required real branch-level reading of the Go
     control flow that a regex cannot cheaply reproduce; the audit already
     did that work and stamped it "verified against source".

  2. Independently regex-extracts (line -> format string, argument list)
     directly from the current internal/mobcommands/<verb>.go source, for
     every `fmt.Sprintf(`...`, args...)` call. This is the MECHANICAL,
     reproducible part: nothing here is copied from the markdown by hand.

  3. Cross-checks the two: at every line the inventory marks as a live
     (non-dark-twin) literal, the freshly-extracted format string and
     argument list must equal what the inventory recorded at that line. Any
     disagreement is collected and printed, not silently resolved in either
     direction.

  4. Skips every dark-twin row (these are deleted, not migrated) and every
     row belonging to attack.go, which has NO corresponding YAML file in the
     shipped store (attack.go does not use messaging.SendTrio and was
     structurally excluded from this migration slice; there is no
     attack.yaml to compare against). The inventory's own prose lists
     "thirteen" files including attack, but the shipped store ships exactly
     twelve melee files plus shoot -- thirteen YAML files total, none of
     them attack.yaml. This is reported, not silently reconciled.

  5. shoot.go is structurally different (see SHOOT below) and is extracted
     by a dedicated routine, not the generic per-line Sprintf scan.

  6. Maps each Go argument name to one of a small canonical token vocabulary
     (see TOKEN VOCABULARY below) and writes the result to
     internal/mobcommands/testdata/pre_migration_literals.json.

TOKEN VOCABULARY

Every fixture row's "args" list names tokens from this fixed vocabulary,
which the Go test's standInArgs() maps to concrete stand-in values:

  actor          bare mob name; the FORMAT STRING supplies the
                 <ansi fg="mobname"> wrap inline (the melee-file pattern:
                 mobName := mob.Character.Name, then
                 `<ansi fg="mobname">%s</ansi>` in the literal itself).
  actor_tagged   an ALREADY-TAGGED mob name substituted with no further
                 wrap in the format string (shoot.go's pattern: mobName is
                 built once as `<ansi fg="mobname">%s</ansi>` and every
                 later Sprintf just drops it in bare with %s).
  actee          bare target name, format string wraps it inline
                 (target.Name / targetName, melee files).
  actee_tagged   an already-tagged target name (shoot.go's targetColored).
  damage         dmgDesc, always bare, format wraps with
                 <ansi fg="damage">.
  label / verb / with   bash.go's species-varying words, always bare.
  weapon         shoot.go's already-tagged weapon name
                 (<ansi fg="itemname">...).
  exitname       a bare direction/exit name (result.ExitName / fromDir).
  position       grapple's result.PositionDesc, bare, format wraps with
                 <ansi fg="cyan">.

The actor/actor_tagged and actee/actee_tagged split exists because the SAME
rendered visual (a colour-tagged name) is produced two different ways in
this codebase: by wrapping a bare value in the format string, or by
pre-building a tagged value and dropping it in with a bare %s. The
byte-identity net has to reproduce whichever one the ORIGINAL Go line
actually does, or it is comparing the wrong thing.

SHOOT

shoot.go does not use the canSee-branched two-full-sentence dark-twin
pattern the other twelve files use, and its cross-room "arrival" line is
built by concatenating three separately-declared fragments (an origin
fragment, a verb fragment, and a shared template) rather than being one
literal. The shipped shoot.yaml (see its own header comment) deliberately
un-composes this into six full-sentence events
(arrival_{unknown,known}_{hit,partial,miss}) rather than mirroring the
fragment shape. So this script extracts shoot.go's three fragment sources by
line (origin_unknown, origin_known, the three verb fragments, and the
template) and composes the six synthetic format strings by hand, with a
comment at each call site. This is the one place this script does not do a
1:1 literal-to-row mapping; see the SHOOT_* constants below.

shoot.go's personal hit/partial/miss lines (the ones sent to the target)
substitute `shooter`, which is `mobName` when not anonymous -- and mobName
on shoot.go:49 is BUILT PRE-TAGGED
(`fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mob.Character.Name)`), unlike
every other file's bare mobName. The implementation plan
(docs/superpowers/plans/2026-09-21-messaging-m4e1-mob-special-moves.md,
around line 945) asserts the opposite -- that `shooter := mobName` is bare
with no ansi tag -- and shoot.yaml's hit/partial/miss events were authored
against that claim, using {actor_plain}. Reading shoot.go itself (this
script's whole premise) shows mobName is tagged. This script trusts the
source, not the plan's prose, and classifies these three rows' shooter
argument as actor_tagged. The resulting mismatch against the shipped
{actor_plain} is reported by the net test, not resolved by this script.

USAGE

    python tools/move_narration_net.py

Writes internal/mobcommands/testdata/pre_migration_literals.json and prints
a summary (row counts per verb, any inventory/source disagreements) to
stdout.
"""
from __future__ import annotations

import json
import os
import re
import sys

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
INVENTORY_PATH = os.path.join(
    REPO_ROOT, "docs", "superpowers", "audits", "2026-09-21-m4e1-site-inventory.md"
)
MOBCOMMANDS_DIR = os.path.join(REPO_ROOT, "internal", "mobcommands")
FIXTURE_PATH = os.path.join(
    REPO_ROOT, "internal", "mobcommands", "testdata", "pre_migration_literals.json"
)

# The 12 files handled by the generic (non-shoot) path. attack.go is
# deliberately excluded: no attack.yaml ships in the store.
GENERIC_VERBS = [
    "bash", "charge", "drain", "gore", "grapple", "hamstring",
    "kick", "maul", "pounce", "rake", "throttle", "trip",
]

# kick.go and trip.go carry a variant axis (stomp/knee/standard;
# tailsweep/trip) orthogonal to the event key: the store's shipped YAML
# keys these as "<variant>_<event>" (e.g. "standard_hit", "tailsweep_miss"),
# not the bare event alone, because the bare key would collide across
# variants. whiff_on_prone is the one event shared by both variants (in
# trip.go) and with charge.go, and ships UNPREFIXED in both kick... no,
# trip.yaml and charge.yaml. The inventory's "Branch" column names the
# variant as its first word (e.g. "standard knockdown, seen",
# "tailsweep hit, observer"); this reads it from there rather than
# hardcoding a line-number table.
VARIANT_PREFIXED_VERBS = {
    "kick": {"standard", "stomp", "knee"},
    "trip": {"tailsweep", "trip"},
}


def event_key_for(verb, event, branch):
    variants = VARIANT_PREFIXED_VERBS.get(verb)
    if not variants:
        return event
    first_word = branch.strip().split()[0].lower() if branch.strip() else ""
    if first_word in variants:
        return f"{first_word}_{event}"
    return event  # e.g. trip.go's shared whiff_on_prone, unprefixed


# Go argument identifier -> canonical token name, for the 12 generic files.
ARG_NAME_TO_TOKEN = {
    "mobName": "actor",
    "target.Name": "actee",
    "targetName": "actee",
    "dmgDesc": "damage",
    "bashLabel": "label",
    "bashVerb": "verb",
    "bashWith": "with",
    "result.PositionDesc": "position",
}

# Matches a backtick-delimited Go string literal, optionally followed on the
# SAME line by ", args...)" -- the tail of an fmt.Sprintf call. Deliberately
# does not require the literal "fmt.Sprintf(" text before the backtick,
# because several calls in this file set wrap onto a second line:
#
#   fmt.Sprintf(
#       `<ansi fg="mobname">%s</ansi> thunders in, ...`, mobName))
#
# On that second line, "fmt.Sprintf(" is not present at all -- it is on the
# line above -- but the backtick literal and its trailing ", mobName))" are.
# A plain backtick assignment with no Sprintf at all (shoot.go's fragment
# lines, e.g. `verb = `and strikes``) matches the same way with an empty
# args group.
BACKTICK_LITERAL_RE = re.compile(r"`([^`]*)`(?:\s*,\s*([^()]*)\))?")


# --------------------------------------------------------------------------
# Inventory markdown parsing
# --------------------------------------------------------------------------

def strip_code_span(cell: str) -> str:
    c = cell.strip()
    if c.startswith("``") and c.endswith("``") and len(c) >= 4:
        c = c[2:-2].strip()
    if c.startswith("`") and c.endswith("`") and len(c) >= 2:
        c = c[1:-1]
    return c


def parse_args_cell(cell: str):
    c = strip_code_span(cell)
    if c in ("", "—", "-"):
        return []
    return [a.strip() for a in c.split(",") if a.strip()]


def parse_inventory(path):
    """Returns dict[(verb, line)] -> row dict."""
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()

    rows = {}
    verb = None
    header_re = re.compile(r"^###\s+(\w+)\.go")
    for raw in lines:
        m = header_re.match(raw)
        if m:
            verb = m.group(1)
            continue
        if not raw.startswith("|"):
            continue
        cells = [c.strip() for c in raw.strip().strip("|").split("|")]
        if len(cells) < 7:
            continue
        line_cell = cells[0]
        if not re.match(r"^\d+$", line_cell):
            continue  # header row, separator row, or not a data row
        if verb is None:
            continue
        line_no = int(line_cell)
        row = {
            "verb": verb,
            "line": line_no,
            "branch": cells[1],
            "event": strip_code_span(cells[2]),
            "role": cells[3],
            "dark_twin": cells[4],
            "format": strip_code_span(cells[5]),
            "args": parse_args_cell(cells[6]),
        }
        # A few lines are reused across sub-tables inside one file section
        # (kick.go's three variants share line-number-looking headers only
        # in prose, never in the actual table -- but guard anyway) by
        # keying on (verb, line, role) so nothing silently overwrites.
        rows[(verb, line_no, row["role"])] = row
    return rows


# --------------------------------------------------------------------------
# Go source extraction
# --------------------------------------------------------------------------

# A bare argument-list continuation line: the third line of a call like
#
#   fmt.Sprintf(
#       `...format on its own line...`,
#       mobName, target.Name)),
#
# where even the literal's trailing comma is on the NEXT line. No backtick,
# just identifiers/dots and a closing paren.
CONTINUATION_ARGS_RE = re.compile(r"^\s*([^`()]*)\)")


def extract_sprintf_calls(go_path):
    """Returns dict[line_no] -> (format, [arg, ...]).

    BACKTICK_LITERAL_RE handles a single-line `fmt.Sprintf(`...`, args...)`
    call, a multi-line call's second line (format literal immediately
    followed by ", args))"), and a bare backtick assignment with no Sprintf
    and no args at all. When neither pattern finds args on the literal's own
    line, this also checks the NEXT line for a bare continuation
    (`mobName, target.Name)),` with no backtick), which is how a small
    number of these calls wrap across three lines.
    """
    with open(go_path, encoding="utf-8") as f:
        lines = f.readlines()

    out = {}
    for i, line in enumerate(lines, start=1):
        for m in BACKTICK_LITERAL_RE.finditer(line):
            fmt_str = m.group(1)
            args_group = m.group(2)
            if args_group is None and i < len(lines):
                cm = CONTINUATION_ARGS_RE.match(lines[i])  # lines[i] is line i+1, 0-indexed
                if cm is not None:
                    args_group = cm.group(1)
            args_str = (args_group or "").strip()
            args = [a.strip() for a in args_str.split(",") if a.strip()]
            # A line with more than one backtick literal is not expected in
            # this file set; if it happens, keep the first and let the
            # cross-check catch anything that looks wrong.
            out.setdefault(i, (fmt_str, args))
    return out


def map_args(args, arg_map, verb, line, disagreements):
    tokens = []
    for a in args:
        tok = arg_map.get(a)
        if tok is None:
            disagreements.append(
                f"{verb}.go:{line}: unmapped Go argument identifier {a!r}; "
                "add it to ARG_NAME_TO_TOKEN or the shoot-specific map"
            )
            tok = f"UNMAPPED({a})"
        tokens.append(tok)
    return tokens


# --------------------------------------------------------------------------
# Generic (12-file) extraction
# --------------------------------------------------------------------------

def build_generic_rows(inventory, disagreements):
    fixture_rows = []
    counts = {}
    for verb in GENERIC_VERBS:
        go_path = os.path.join(MOBCOMMANDS_DIR, f"{verb}.go")
        src_calls = extract_sprintf_calls(go_path)

        verb_rows = [r for (v, _line, _role), r in inventory.items() if v == verb]
        verb_rows.sort(key=lambda r: (r["line"], r["role"]))

        n = 0
        for row in verb_rows:
            if row["dark_twin"].lower() == "yes":
                continue
            line = row["line"]
            src = src_calls.get(line)
            if src is None:
                disagreements.append(
                    f"{verb}.go:{line}: inventory lists a live literal here but "
                    "no fmt.Sprintf(...) call was found at this line by fresh "
                    "regex extraction; falling back to the inventory's own "
                    "format/args, UNVERIFIED against current source"
                )
                fmt_str, args = row["format"], row["args"]
            else:
                fmt_str, args = src
                if fmt_str != row["format"]:
                    disagreements.append(
                        f"{verb}.go:{line}: format string disagreement\n"
                        f"    inventory: {row['format']!r}\n"
                        f"    source:    {fmt_str!r}"
                    )
                if args != row["args"]:
                    disagreements.append(
                        f"{verb}.go:{line}: argument list disagreement\n"
                        f"    inventory: {row['args']!r}\n"
                        f"    source:    {args!r}"
                    )

            tokens = map_args(args, ARG_NAME_TO_TOKEN, verb, line, disagreements)
            fixture_rows.append({
                "verb": verb,
                "event": event_key_for(verb, row["event"], row["branch"]),
                "role": row["role"],
                "format": fmt_str,
                "args": tokens,
                "source_line": line,
            })
            n += 1
        counts[verb] = n
    return fixture_rows, counts


# --------------------------------------------------------------------------
# shoot.go extraction (structurally special, see module docstring)
# --------------------------------------------------------------------------

SHOOT_ARG_TO_TOKEN = {
    "mobName": "actor_tagged",
    "shooter": "actor_tagged",
    "weapon": "weapon",
    "targetColored": "actee_tagged",
    "result.ExitName": "exitname",
    "fromDir": "exitname",
}


def build_shoot_rows(inventory, disagreements):
    go_path = os.path.join(MOBCOMMANDS_DIR, "shoot.go")
    src_calls = extract_sprintf_calls(go_path)

    def want(line, expect_fmt=None, expect_args=None):
        src = src_calls.get(line)
        if src is None:
            disagreements.append(f"shoot.go:{line}: expected a Sprintf call, found none")
            return expect_fmt or "", expect_args or []
        fmt_str, args = src
        if expect_fmt is not None and fmt_str != expect_fmt:
            disagreements.append(
                f"shoot.go:{line}: format string disagreement\n"
                f"    expected: {expect_fmt!r}\n    source:   {fmt_str!r}"
            )
        if expect_args is not None and args != expect_args:
            disagreements.append(
                f"shoot.go:{line}: argument list disagreement\n"
                f"    expected: {expect_args!r}\n    source:   {args!r}"
            )
        return fmt_str, args

    rows = []

    def add(event, role, fmt_str, args):
        rows.append({
            "verb": "shoot",
            "event": event,
            "role": role,
            "format": fmt_str,
            "args": map_args(args, SHOOT_ARG_TO_TOKEN, "shoot", "-", disagreements),
            "source_line": "composed" if not isinstance(args, list) else "-",
        })

    # --- personal lines (actee): hit / partial / miss ---------------------
    fmt_str, args = want(91, "%s's shot strikes you!", ["shooter"])
    add("hit", "actee", fmt_str, args)
    fmt_str, args = want(93, "%s's shot goes wide, but the edge of it still clips you!", ["shooter"])
    add("partial", "actee", fmt_str, args)
    fmt_str, args = want(101, "%s's shot narrowly misses you!", ["shooter"])
    add("miss", "actee", fmt_str, args)

    # --- same-room announce / cross-room depart (observer) ----------------
    fmt_str, args = want(127, "%s fires their %s at %s!", ["mobName", "weapon", "targetColored"])
    add("fire_announce", "observer", fmt_str, args)
    fmt_str, args = want(142, "%s fires their %s %sward.", ["mobName", "weapon", "result.ExitName"])
    add("fire_depart", "observer", fmt_str, args)

    # --- cross-room arrival, composed from three fragments -----------------
    # The three raw fragments this composes, extracted (not retyped) from
    # source so a change to any of them is caught by the cross-check below.
    origin_unknown_fmt, _ = want(155, "from somewhere nearby", [])
    origin_known_fmt, origin_known_args = want(
        157, 'from beyond the <ansi fg="exit">%s</ansi>', ["fromDir"]
    )
    verb_hit_fmt, _ = want(162, "and strikes", [])
    verb_partial_fmt, _ = want(164, "and clips", [])
    verb_miss_fmt, _ = want(166, "and narrowly misses", [])
    template_fmt, template_args = want(
        169, "A shot streaks in %s %s %s!", ["origin", "verb", "targetColored"]
    )
    if template_args != ["origin", "verb", "targetColored"]:
        disagreements.append(
            "shoot.go:169: template argument order changed; the composed "
            "arrival rows assume [origin, verb, targetColored]"
        )

    def composed(origin_fmt, origin_args, verb_fmt):
        # origin_fmt is either the bare "from somewhere nearby" (no args) or
        # the 'from beyond the <ansi ...>%s</ansi>' template (one arg,
        # fromDir/exitname). verb_fmt is always a bare fragment, no args.
        full = template_fmt % (origin_fmt, verb_fmt, "%s")
        return full, (list(origin_args) + ["targetColored"])

    fmt_str, args = composed(origin_unknown_fmt, [], verb_hit_fmt)
    add("arrival_unknown_hit", "remote_observer", fmt_str, args)
    fmt_str, args = composed(origin_unknown_fmt, [], verb_partial_fmt)
    add("arrival_unknown_partial", "remote_observer", fmt_str, args)
    fmt_str, args = composed(origin_unknown_fmt, [], verb_miss_fmt)
    add("arrival_unknown_miss", "remote_observer", fmt_str, args)
    fmt_str, args = composed(origin_known_fmt, origin_known_args, verb_hit_fmt)
    add("arrival_known_hit", "remote_observer", fmt_str, args)
    fmt_str, args = composed(origin_known_fmt, origin_known_args, verb_partial_fmt)
    add("arrival_known_partial", "remote_observer", fmt_str, args)
    fmt_str, args = composed(origin_known_fmt, origin_known_args, verb_miss_fmt)
    add("arrival_known_miss", "remote_observer", fmt_str, args)

    return rows, {"shoot": len(rows)}


# --------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------

STORE_DIR = os.path.join(
    REPO_ROOT, "_datafiles", "world", "dogmud", "narration", "special-moves"
)

ROLE_TO_YAML_KEY = {
    "actor": "actor",
    "actee": "actee",
    "observer": "observer",
    "remote_observer": "remote_observer",
}


def check_against_shipped_store(all_rows, disagreements):
    """Sanity check ONLY: every fixture row's (verb, event, role) must name a
    key that actually exists in the shipped YAML, so a fixture/store naming
    drift (e.g. a missing variant prefix) is caught here rather than showing
    up as a confusing "verb has no such event" failure inside the Go test.
    This does not require PyYAML to be present in every environment the
    script runs in; it degrades to a skip with a note if the import fails.
    """
    try:
        import yaml  # type: ignore
    except ImportError:
        print("(PyYAML not available; skipping the shipped-store cross-check)")
        return

    stores = {}
    for fname in os.listdir(STORE_DIR):
        if not fname.endswith(".yaml"):
            continue
        with open(os.path.join(STORE_DIR, fname), encoding="utf-8") as f:
            doc = yaml.safe_load(f)
        stores[doc["moveid"]] = doc.get("events", {})

    for row in all_rows:
        events = stores.get(row["verb"])
        if events is None:
            disagreements.append(
                f"{row['verb']}/{row['event']}/{row['role']}: verb has no "
                f"{row['verb']}.yaml in the shipped store"
            )
            continue
        ev = events.get(row["event"])
        if ev is None:
            disagreements.append(
                f"{row['verb']}/{row['event']}/{row['role']}: event {row['event']!r} "
                f"not in {row['verb']}.yaml (has: {sorted(events.keys())})"
            )
            continue
        yaml_key = ROLE_TO_YAML_KEY[row["role"]]
        if not ev.get(yaml_key):
            disagreements.append(
                f"{row['verb']}/{row['event']}/{row['role']}: role {yaml_key!r} "
                f"is empty or absent in the shipped event"
            )


# ============================================================================
# usercommands (player-side) extraction -- messaging-M4e1b Task 1
#
# WHAT THIS DOES
#
# The mobcommands extraction above leans on a hand-verified inventory doc
# (docs/superpowers/audits/2026-09-21-m4e1-site-inventory.md) that does not
# exist for the player files -- no PR has audited them line by line yet. So
# this half of the script is fully self-verifying instead: every fixture row
# is checked against a FRESH parse of the current internal/usercommands/*.go
# source at extraction time, and any drift between what is hand-listed below
# and what the source actually contains is reported as a disagreement, never
# silently resolved.
#
# Two extraction strategies, matching the two shapes PR 1a's census
# (docs/superpowers/plans/2026-09-21-messaging-m4e1b-player-special-moves.md)
# found in these twelve files:
#
#   POOLED files (drain, gore, kick, maul, pounce, rake, throttle) declare
#   `varName := []string{ "...", "...", ... }` pools and pick with
#   util.Rand(len(pool)). extract_all_pools_in_order() walks the file and
#   harvests every backtick literal from every such pool IN FILE ORDER; a
#   hand-authored table (verified against source by direct reading, cross
#   checked here by variable name and pool count) says which (event, role,
#   args) each pool corresponds to. Every entry of every pool becomes its own
#   fixture row, tagged with its index in the pool -- this is what lets the
#   net fail on entry 5 and not just entry 0.
#
#   UNPOOLED files (bash, trip, grapple, shoot, throw) already ship one
#   literal per role per branch (verified: zero `[]string{` / `util.Rand`
#   matches in any of the five). Each row is hand-transcribed from source and
#   then matched by EXACT TEXT against extract_literal_calls()'s fresh
#   extraction of every backtick literal + its call-site argument list; a
#   transcription typo or a source edit that moves the text produces a
#   reported disagreement rather than a silent pass.
#
# extract_literal_calls() is a second, independent Sprintf-call extractor
# (not the mobcommands section's extract_sprintf_calls / BACKTICK_LITERAL_RE)
# because several of these files pass an argument that is ITSELF a call
# carrying its own parentheses, e.g. trip.go:
#
#   fmt.Sprintf(`...%s...(<ansi fg="damage">%s</ansi>)`, targetName,
#       combat.GetDamageDescription(result.Damage, result.TargetMaxHP))
#
# BACKTICK_LITERAL_RE's `([^()]*)\)` argument capture cannot see past the
# first `)`, which belongs to GetDamageDescription's own call, not the
# enclosing Sprintf. extract_literal_calls() instead scans character by
# character with a paren-depth counter, splitting on top-level commas only,
# so a nested call in argument position does not truncate the argument list.
#
# WHAT SHOOT.GO (USER SIDE) IS PARAMETERISED DIFFERENTLY FROM SHOOT.GO (MOB
# SIDE)
#
# The mob's shoot.go has no `actor` role (a mob has no client) and speaks the
# shooter's own outcome nowhere. The player's shoot.go speaks it in
# `shooterLine` (Fire's own hit/partial/miss lines back to the shooter) --
# three genuinely new rows the mob-side extraction never had reason to
# produce. The two files' fire_announce, fire_depart and six arrival_* rows
# are otherwise the SAME events, reached the same way (a shot leaving/
# arriving is described identically regardless of who fired it), composed
# from an origin fragment and an outcome template exactly as the mob-side
# SHOOT section already documents; the same compose-from-fragments technique
# is reused here for the player's origin/template split.
#
# THROW.GO has no equivalent in the mob-side store at all (mobs cannot throw
# grenades) and is AREA-only: it has no actee ever (an AoE has no single
# target), so every row extracted from it is actor or observer.
#
# TOKEN VOCABULARY ADDITIONS beyond the mobcommands section's actor/
# actor_tagged/actee/actee_tagged/damage/label/verb/with/weapon/exitname/
# position:
#
#   heal    drain's lifesteal detail line (combat.GetHealDescription), a
#           HEAL amount, never conflated with a DAMAGE amount even though
#           both are opaque prose strings.
#   item    throw's matchItem.DisplayName(), always bare in the Go call site
#           (the format string supplies its own <ansi fg="itemname"> wrap).
#
# WHAT COUNTS AS "IN SCOPE" HERE, PER TASK 1'S BRIEF
#
# Skipped, not extracted at all (see the task brief for the full reasoning):
#   - prose sourced from OUTSIDE the file under test: moveDefenceLines()'s
#     defence.ToRoom/ToDefender/ToAttacker, combat.ChannelDefenceShortageText,
#     result.DisarmResult.*, result.CritFailure.*, combat.
#     RenderChannelDefenceMessages' triad.* (shoot.go, throw.go)
#   - user.SendText(...) calls: pre-flight refusals, validation messages, and
#     (throw.go specifically) the per-target flavour notes sent directly to
#     the thrower rather than through messaging.SendTrio -- none of these go
#     through the store's actor/actee/observer/remote_observer roles, so none
#     of them are within this net's surface
#   - drain's healMsgs ACTEE/OBSERVER (both messaging.NoLine by design, a
#     private detail line); the ACTOR line is real content and IS extracted
#   - throttle's InterruptedCast trio: a hardcoded 1/1/1 already flagged by
#     the plan census as "not a pool", and per Task 1's brief, not extracted
#
# Included despite not being explicitly named in the plan's 30-event census
# (which counts only what Task 2 must SQUARE): drain's two healMsgs actor
# pools (legitimate Actor-only content, per the task brief), grapple's
# PositionPenalty/DefensePenalty single Actor-only literals (the same
# NoLine-twice, Actor-only shape as drain's heal lines), and throw's four
# genuinely-narrated events (hurl, fumble, a boss-interrupt cast-disruption
# announcement, and the defended-partial local literal). None of these has a
# home in the shipped store yet, so every row from them skips at the
# verb/event tier until a later task decides whether and how to migrate them.
# ============================================================================

USERCOMMANDS_DIR = os.path.join(REPO_ROOT, "internal", "usercommands")
USER_FIXTURE_PATH = os.path.join(
    REPO_ROOT, "internal", "usercommands", "testdata", "pre_migration_literals.json"
)

USER_POOLED_VERBS = ["drain", "gore", "kick", "maul", "pounce", "rake", "throttle"]
USER_UNPOOLED_VERBS = ["bash", "trip", "grapple", "shoot", "throw"]


# --------------------------------------------------------------------------
# A second, paren-aware Sprintf-call extractor (see module docstring above
# for why the mobcommands section's extract_sprintf_calls is not reused).
# --------------------------------------------------------------------------

BACKTICK_ONLY_RE = re.compile(r"`([^`]*)`")


def split_call_tail(text):
    """text starts right after a format string's closing backtick, e.g.
    ", targetName, combat.GetDamageDescription(a, b))) " or just ")),".
    Returns the top-level comma-separated argument list up to the `)` that
    closes THIS Sprintf call (paren depth back to zero), or None if that
    closing paren is not present in `text` at all (caller should append more
    source text and retry).
    """
    i, n = 0, len(text)
    while i < n and text[i] in " \t":
        i += 1
    if i < n and text[i] == ")":
        return []
    if i >= n or text[i] != ",":
        return None
    i += 1
    args = []
    depth = 0
    current = []
    while i < n:
        c = text[i]
        if c == "(":
            depth += 1
            current.append(c)
        elif c == ")":
            if depth == 0:
                tail = "".join(current).strip()
                if tail:
                    args.append(tail)
                return args
            depth -= 1
            current.append(c)
        elif c == "," and depth == 0:
            args.append("".join(current).strip())
            current = []
        else:
            current.append(c)
        i += 1
    return None


def extract_literal_calls(go_path, max_lookahead=4):
    """Returns dict[line_no] -> (format, [arg, ...]) for every backtick
    literal in go_path, using split_call_tail (paren-depth aware) rather than
    a single-shot regex, so a nested call in argument position (trip.go's
    inline combat.GetDamageDescription(...)) does not truncate the argument
    list. Multi-line calls (throw.go's format-on-one-line,
    args-on-the-next-line style) are handled by extending the search window
    up to max_lookahead further lines.
    """
    with open(go_path, encoding="utf-8") as f:
        lines = f.readlines()

    out = {}
    for i, line in enumerate(lines, start=1):
        for m in BACKTICK_ONLY_RE.finditer(line):
            fmt_str = m.group(1)
            tail = line[m.end():]
            args = split_call_tail(tail)
            j = i
            while args is None and (j - i) < max_lookahead and j < len(lines):
                tail += lines[j]
                args = split_call_tail(tail)
                j += 1
            if args is None:
                args = []
            out.setdefault(i, (fmt_str, args))
    return out


# --------------------------------------------------------------------------
# Pooled-file extraction: harvest every []string{...} pool IN FILE ORDER,
# zip against a hand-authored table of (event, role, args) per pool.
# --------------------------------------------------------------------------

POOL_DECL_RE = re.compile(r"^\s*(\w+)\s*:?=\s*\[\]string\{\s*$")


def extract_all_pools_in_order(go_path):
    """Returns [(var_name, decl_line_1based, [entry, ...]), ...] in file
    order, for every `varName := []string{` / `varName = []string{` block
    whose entries are one backtick literal per line (verified true of all
    seven pooled usercommands files by direct reading).
    """
    with open(go_path, encoding="utf-8") as f:
        lines = f.readlines()

    pools = []
    i, n = 0, len(lines)
    while i < n:
        m = POOL_DECL_RE.match(lines[i])
        if m:
            var_name = m.group(1)
            entries = []
            j = i + 1
            while j < n and not lines[j].strip().startswith("}"):
                bm = BACKTICK_ONLY_RE.search(lines[j])
                if bm:
                    entries.append(bm.group(1))
                j += 1
            pools.append((var_name, i + 1, entries))
            i = j + 1
            continue
        i += 1
    return pools


def branch_triplet(event, actor_var, actee_var, observer_var, has_damage):
    """The uniform per-branch shape every one of the seven pooled files uses:
    actor's pool takes (actee[, damage]), actee's takes (actor[, damage]),
    observer's always takes (actor, actee). Returns the three
    (var_name, event, role, args) table rows for one branch.
    """
    actor_args = ["actee", "damage"] if has_damage else ["actee"]
    actee_args = ["actor", "damage"] if has_damage else ["actor"]
    return [
        (actor_var, event, "actor", actor_args),
        (actee_var, event, "actee", actee_args),
        (observer_var, event, "observer", ["actor", "actee"]),
    ]


def rows_from_pool_table(verb, go_path, table):
    pools = extract_all_pools_in_order(go_path)
    if len(pools) != len(table):
        raise ValueError(
            f"{go_path}: found {len(pools)} pool declarations in file order "
            f"but the hand-authored table has {len(table)} entries -- the "
            f"table is out of sync with source (a pool was added, removed, "
            f"or reordered)"
        )
    rows = []
    for (var_name, decl_line, entries), (expect_var, event, role, args) in zip(pools, table):
        if var_name != expect_var:
            raise ValueError(
                f"{go_path}:{decl_line}: expected pool {expect_var!r} at this "
                f"position in file order, found {var_name!r} instead -- the "
                f"table is out of sync with source"
            )
        if not entries:
            raise ValueError(f"{go_path}:{decl_line}: {var_name} pool extracted ZERO entries")
        for idx, fmt_str in enumerate(entries):
            rows.append({
                "verb": verb, "event": event, "role": role, "index": idx,
                "format": fmt_str, "args": list(args),
            })
    return rows


# Per-verb pool tables. Verified against source by direct reading; the
# variable-name + pool-count cross-check in rows_from_pool_table() catches
# any drift between this table and the live file.

DRAIN_TABLE = (
    branch_triplet("hit", "drainMsgs", "drainTargetMsgs", "drainRoomMsgs", True)
    + [("healMsgs", "hit_heal", "actor", ["heal"])]
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + [("healMsgs", "partial_heal", "actor", ["heal"])]
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

GORE_TABLE = (
    branch_triplet("knockdown", "goreMsgs", "goreTargetMsgs", "goreRoomMsgs", True)
    + branch_triplet("hit", "goreMsgs", "goreTargetMsgs", "goreRoomMsgs", True)
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

MAUL_TABLE = (
    branch_triplet("hit", "maulMsgs", "maulTargetMsgs", "maulRoomMsgs", True)
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

POUNCE_TABLE = (
    branch_triplet("knockdown", "pounceMsgs", "pounceTargetMsgs", "pounceRoomMsgs", True)
    + branch_triplet("hit", "pounceMsgs", "pounceTargetMsgs", "pounceRoomMsgs", True)
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

RAKE_TABLE = (
    branch_triplet("hit", "rakeMsgs", "rakeTargetMsgs", "rakeRoomMsgs", True)
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

THROTTLE_TABLE = (
    branch_triplet("hit", "hitMsgs", "hitTargetMsgs", "hitRoomMsgs", True)
    + branch_triplet("partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
    + branch_triplet("miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
)

KICK_TABLE = []
for _variant in ("stomp", "knee"):
    KICK_TABLE += branch_triplet(f"{_variant}_hit", "kickMsgs", "kickTargetMsgs", "kickRoomMsgs", True)
    KICK_TABLE += branch_triplet(f"{_variant}_miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
    KICK_TABLE += branch_triplet(f"{_variant}_partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)
KICK_TABLE += branch_triplet("standard_hit", "kickMsgs", "kickTargetMsgs", "kickRoomMsgs", True)
KICK_TABLE += branch_triplet("standard_knockdown", "knockdownMsgs", "knockdownTargetMsgs", "knockdownRoomMsgs", True)
KICK_TABLE += branch_triplet("standard_miss", "missMsgs", "missTargetMsgs", "missRoomMsgs", False)
KICK_TABLE += branch_triplet("standard_partial", "partialMsgs", "partialTargetMsgs", "partialRoomMsgs", True)

POOL_VERB_TABLES = {
    "drain": DRAIN_TABLE,
    "gore": GORE_TABLE,
    "kick": KICK_TABLE,
    "maul": MAUL_TABLE,
    "pounce": POUNCE_TABLE,
    "rake": RAKE_TABLE,
    "throttle": THROTTLE_TABLE,
}


def build_pooled_rows(disagreements):
    rows = []
    counts = {}
    for verb, table in POOL_VERB_TABLES.items():
        go_path = os.path.join(USERCOMMANDS_DIR, f"{verb}.go")
        try:
            verb_rows = rows_from_pool_table(verb, go_path, table)
        except ValueError as e:
            disagreements.append(str(e))
            verb_rows = []
        rows += verb_rows
        counts[verb] = len(verb_rows)
    return rows, counts


# --------------------------------------------------------------------------
# Unpooled-file extraction: hand-transcribed (event, role, format, args)
# rows, matched against extract_literal_calls() by EXACT format-string text.
# --------------------------------------------------------------------------

def find_and_add(rows, calls, used_lines, verb, event, role, expected_fmt, token_args, disagreements):
    match_line = None
    match_args = None
    for line, (fmt_str, args) in sorted(calls.items()):
        if line in used_lines:
            continue
        if fmt_str == expected_fmt:
            match_line = line
            match_args = args
            break
    if match_line is None:
        disagreements.append(
            f"{verb}.go: {event}/{role}: expected literal not found verbatim "
            f"in a fresh source scan: {expected_fmt!r}"
        )
        return
    used_lines.add(match_line)
    if len(match_args) != len(token_args):
        disagreements.append(
            f"{verb}.go:{match_line}: {event}/{role}: expected {len(token_args)} "
            f"call-site arg(s) {token_args!r} but source has {len(match_args)} "
            f"({match_args!r})"
        )
    rows.append({
        "verb": verb, "event": event, "role": role, "index": 0,
        "format": expected_fmt, "args": list(token_args),
    })


# ---- bash.go ---------------------------------------------------------------

BASH_ROWS = [
    ("knockdown", "actor", """Your <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("knockdown", "actee", """<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("knockdown", "observer", """<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!""", ["actor", "actee"]),
    ("hit", "actor", """Your <ansi fg="yellow-bold">shield bash</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("hit", "actee", """<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("hit", "observer", """<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield!""", ["actor", "actee"]),
    ("partial", "observer", """<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield, who staggers but stays up!""", ["actor", "actee"]),
    ("partial", "actor", """Your <ansi fg="yellow-bold">shield bash</ansi> fails to floor <ansi fg="mobname">%s</ansi>, but still slams into them! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("partial", "actee", """<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> fails to floor you, but still slams into you! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("miss", "actor", """Your <ansi fg="yellow-bold">shield bash</ansi> misses <ansi fg="mobname">%s</ansi>!""", ["actee"]),
    ("miss", "actee", """<ansi fg="username">%s</ansi> attempts to bash you with their shield, but misses!""", ["actor"]),
    ("miss", "observer", """<ansi fg="username">%s</ansi> attempts to bash <ansi fg="mobname">%s</ansi>, but misses!""", ["actor", "actee"]),
]


def build_bash_rows(disagreements):
    go_path = os.path.join(USERCOMMANDS_DIR, "bash.go")
    calls = extract_literal_calls(go_path)
    used = set()
    rows = []
    for event, role, fmt_str, args in BASH_ROWS:
        find_and_add(rows, calls, used, "bash", event, role, fmt_str, args, disagreements)
    return rows


# ---- trip.go ----------------------------------------------------------------

TRIP_ROWS = [
    ("tailsweep_knockdown", "actor", """Your <ansi fg="yellow-bold">tailsweep</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("tailsweep_knockdown", "actee", """<ansi fg="username">%s</ansi> hammers you with their tail, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("tailsweep_knockdown", "observer", """<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!""", ["actor", "actee"]),
    ("tailsweep_hit", "actor", """Your <ansi fg="yellow-bold">tailsweep</ansi> strikes <ansi fg="mobname">%s</ansi>, but they keep their footing! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("tailsweep_hit", "actee", """<ansi fg="username">%s</ansi> sweeps at you with their tail, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("tailsweep_hit", "observer", """<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, but they keep their footing!""", ["actor", "actee"]),
    ("trip_knockdown", "actor", """Your <ansi fg="yellow-bold">trip</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("trip_knockdown", "actee", """<ansi fg="username">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("trip_knockdown", "observer", """<ansi fg="username">%s</ansi> trips <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!""", ["actor", "actee"]),
    ("trip_hit", "actor", """Your <ansi fg="yellow-bold">trip</ansi> strikes <ansi fg="mobname">%s</ansi>, but they stay on their feet! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("trip_hit", "actee", """<ansi fg="username">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("trip_hit", "observer", """<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but they keep their footing!""", ["actor", "actee"]),
    ("tailsweep_partial", "observer", """<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, who staggers but keeps their feet!""", ["actor", "actee"]),
    ("tailsweep_partial", "actor", """Your <ansi fg="yellow-bold">tailsweep</ansi> fails to trip <ansi fg="mobname">%s</ansi>, but still cracks into them! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("tailsweep_partial", "actee", """<ansi fg="username">%s</ansi> swings their tail and you keep your feet, but it still cracks into you! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("trip_partial", "observer", """<ansi fg="username">%s</ansi> tries to trip <ansi fg="mobname">%s</ansi>, who staggers but keeps their feet!""", ["actor", "actee"]),
    ("trip_partial", "actor", """Your <ansi fg="yellow-bold">trip</ansi> fails to take <ansi fg="mobname">%s</ansi> down, but still catches them hard! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
    ("trip_partial", "actee", """<ansi fg="username">%s</ansi> tries to trip you and you keep your feet, but the sweep still catches you! (<ansi fg="damage">%s</ansi>)""", ["actor", "damage"]),
    ("tailsweep_miss", "actor", """Your <ansi fg="yellow-bold">tailsweep</ansi> misses <ansi fg="mobname">%s</ansi>!""", ["actee"]),
    ("tailsweep_miss", "actee", """<ansi fg="username">%s</ansi> swings their tail at you, but you avoid it!""", ["actor"]),
    ("tailsweep_miss", "observer", """<ansi fg="username">%s</ansi> attempts a tailsweep on <ansi fg="mobname">%s</ansi>, but misses!""", ["actor", "actee"]),
    ("trip_miss", "actor", """Your <ansi fg="yellow-bold">trip</ansi> attempt misses <ansi fg="mobname">%s</ansi>!""", ["actee"]),
    ("trip_miss", "actee", """<ansi fg="username">%s</ansi> attempts to trip you, but you avoid it!""", ["actor"]),
    ("trip_miss", "observer", """<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but misses!""", ["actor", "actee"]),
]


def build_trip_rows(disagreements):
    go_path = os.path.join(USERCOMMANDS_DIR, "trip.go")
    calls = extract_literal_calls(go_path)
    used = set()
    rows = []
    for event, role, fmt_str, args in TRIP_ROWS:
        find_and_add(rows, calls, used, "trip", event, role, fmt_str, args, disagreements)
    return rows


# ---- grapple.go ---------------------------------------------------------------

GRAPPLE_ROWS = [
    ("success", "actor", """You <ansi fg="yellow-bold">grapple</ansi> <ansi fg="mobname">%s</ansi>, transitioning to <ansi fg="cyan">%s</ansi> position!""", ["actee", "position"]),
    ("success", "actee", """<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!""", ["actor", "position"]),
    ("success", "observer", """<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="mobname">%s</ansi> into <ansi fg="cyan">%s</ansi> position!""", ["actor", "actee", "position"]),
    ("fail", "actor", """Your <ansi fg="yellow-bold">grapple</ansi> attempt against <ansi fg="mobname">%s</ansi> fails!""", ["actee"]),
    ("fail", "actee", """<ansi fg="username">%s</ansi> tries to grapple you, but you slip away!""", ["actor"]),
    ("fail", "observer", """<ansi fg="username">%s</ansi> tries to grapple <ansi fg="mobname">%s</ansi>, but fails!""", ["actor", "actee"]),
    ("prone_penalty", "actor", """<ansi fg="yellow">%s was already prone - they had little chance to resist!</ansi>""", ["actee"]),
    ("defense_exposed", "actor", """<ansi fg="red">Your failed attempt leaves you exposed!</ansi>""", []),
]


def build_grapple_rows(disagreements):
    go_path = os.path.join(USERCOMMANDS_DIR, "grapple.go")
    calls = extract_literal_calls(go_path)
    used = set()
    rows = []
    for event, role, fmt_str, args in GRAPPLE_ROWS:
        find_and_add(rows, calls, used, "grapple", event, role, fmt_str, args, disagreements)
    return rows


# ---- shoot.go (player side, Fire) --------------------------------------------

SHOOT_USER_ROWS = [
    ("hit", "actor", """Your shot takes %s (<ansi fg="damage">%s</ansi>)!""", ["actee_tagged", "damage"]),
    ("partial", "actor", """Your shot goes wide of %s, but the edge of it still clips them! (<ansi fg="damage">%s</ansi>)""", ["actee_tagged", "damage"]),
    ("miss", "actor", """Your shot goes wide of %s!""", ["actee_tagged"]),
    ("hit", "actee", """%s's shot strikes you (<ansi fg="damage">%s</ansi>)!""", ["actor_tagged", "damage"]),
    ("partial", "actee", """%s's shot goes wide, but the edge of it still clips you! (<ansi fg="damage">%s</ansi>)""", ["actor_tagged", "damage"]),
    ("miss", "actee", """%s's shot narrowly misses you!""", ["actor_tagged"]),
    ("fire_announce", "observer", """%s fires their %s at %s!""", ["actor_tagged", "weapon", "actee_tagged"]),
    ("fire_depart", "observer", """%s fires their %s %sward.""", ["actor_tagged", "weapon", "exitname"]),
]

# The arrival_* remote_observer rows are composed from an origin fragment
# (unknown vs known exit) and an outcome template, mirroring the mob-side
# SHOOT section's own compose() technique -- see this file's module
# docstring. Each template is verified fresh against source, then the FIRST
# %s of the template (the origin slot) is replaced with the origin
# fragment's own raw text, exactly as the live Go code builds `origin` before
# handing it to the outer Sprintf.
SHOOT_ARRIVAL_TEMPLATES = {
    "hit": """A shot streaks in %s and strikes %s!""",
    "partial": """A shot streaks in %s and clips %s!""",
    "miss": """A shot streaks in %s and narrowly misses %s!""",
}
SHOOT_ORIGIN_UNKNOWN = "from somewhere nearby"
SHOOT_ORIGIN_KNOWN = """from beyond the <ansi fg="exit">%s</ansi>"""


def build_shoot_user_rows(disagreements):
    go_path = os.path.join(USERCOMMANDS_DIR, "shoot.go")
    calls = extract_literal_calls(go_path)
    used = set()
    rows = []
    for event, role, fmt_str, args in SHOOT_USER_ROWS:
        find_and_add(rows, calls, used, "shoot", event, role, fmt_str, args, disagreements)

    # Cross-check the origin fragments and outcome templates against a fresh
    # scan, the same way the mob-side SHOOT section verifies its fragments,
    # before composing the six arrival_* rows by hand.
    origin_unknown_seen = False
    origin_known_seen = False
    for _line, (fmt_str, fargs) in calls.items():
        if fmt_str == SHOOT_ORIGIN_UNKNOWN and fargs == []:
            origin_unknown_seen = True
        if fmt_str == SHOOT_ORIGIN_KNOWN and fargs == ["fromDir"]:
            origin_known_seen = True
    if not origin_unknown_seen:
        disagreements.append(
            f"shoot.go: expected the bare origin fragment {SHOOT_ORIGIN_UNKNOWN!r} "
            "not found verbatim in a fresh source scan"
        )
    if not origin_known_seen:
        disagreements.append(
            f"shoot.go: expected the known-exit origin fragment {SHOOT_ORIGIN_KNOWN!r} "
            "(args=['fromDir']) not found verbatim in a fresh source scan"
        )

    for outcome, template in SHOOT_ARRIVAL_TEMPLATES.items():
        found = False
        for _line, (fmt_str, fargs) in calls.items():
            if fmt_str == template and fargs == ["origin", "targetColored"]:
                found = True
                break
        if not found:
            disagreements.append(
                f"shoot.go: arrival template for {outcome!r} not found verbatim "
                f"with args ['origin','targetColored'] in a fresh source scan: {template!r}"
            )
            continue
        unknown_fmt = template.replace("%s", SHOOT_ORIGIN_UNKNOWN, 1)
        rows.append({
            "verb": "shoot", "event": f"arrival_unknown_{outcome}", "role": "remote_observer",
            "index": 0, "format": unknown_fmt, "args": ["actee_tagged"],
        })
        known_fmt = template.replace("%s", SHOOT_ORIGIN_KNOWN, 1)
        rows.append({
            "verb": "shoot", "event": f"arrival_known_{outcome}", "role": "remote_observer",
            "index": 0, "format": known_fmt, "args": ["exitname", "actee_tagged"],
        })

    return rows


# ---- throw.go -----------------------------------------------------------------

THROW_ROWS = [
    ("hurl", "actor", """<ansi fg="yellow-bold">You hurl the <ansi fg="itemname">%s</ansi> into the fray!</ansi>""", ["item"]),
    ("hurl", "observer", """<ansi fg="yellow-bold"><ansi fg="username">%s</ansi> hurls a <ansi fg="itemname">%s</ansi> into the fray!</ansi>""", ["actor", "item"]),
    ("fumble", "actor", """<ansi fg="red-bold">Your throw goes horribly wrong — the projectile detonates in your hand!</ansi>""", []),
    ("fumble", "observer", """<ansi fg="red"><ansi fg="username">%s</ansi>'s throw backfires spectacularly!</ansi>""", ["actor"]),
    ("cast_interrupt", "actor", """<ansi fg="cyan-bold">The blast shatters %s's concentration -- its spell collapses!</ansi>""", ["actee"]),
    ("cast_interrupt", "observer", """<ansi fg="cyan">%s's spell collapses as the blast strikes!</ansi>""", ["actee"]),
    ("partial_hit", "actor", """The edge of the blast still catches <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)""", ["actee", "damage"]),
]


def build_throw_rows(disagreements):
    go_path = os.path.join(USERCOMMANDS_DIR, "throw.go")
    calls = extract_literal_calls(go_path)
    used = set()
    rows = []
    for event, role, fmt_str, args in THROW_ROWS:
        find_and_add(rows, calls, used, "throw", event, role, fmt_str, args, disagreements)
    return rows


def build_unpooled_rows(disagreements):
    rows = []
    counts = {}
    builders = {
        "bash": build_bash_rows,
        "trip": build_trip_rows,
        "grapple": build_grapple_rows,
        "shoot": build_shoot_user_rows,
        "throw": build_throw_rows,
    }
    for verb, builder in builders.items():
        verb_rows = builder(disagreements)
        rows += verb_rows
        counts[verb] = len(verb_rows)
    return rows, counts


def main_usercommands():
    disagreements = []

    pooled_rows, pooled_counts = build_pooled_rows(disagreements)
    unpooled_rows, unpooled_counts = build_unpooled_rows(disagreements)

    all_rows = pooled_rows + unpooled_rows
    counts = {**pooled_counts, **unpooled_counts}

    os.makedirs(os.path.dirname(USER_FIXTURE_PATH), exist_ok=True)
    with open(USER_FIXTURE_PATH, "w", encoding="utf-8") as f:
        json.dump(all_rows, f, indent=2, ensure_ascii=False)
        f.write("\n")

    pool_entry_rows = sum(1 for r in all_rows if r["index"] > 0)

    print(f"Wrote {len(all_rows)} fixture rows to {USER_FIXTURE_PATH}")
    for verb in USER_POOLED_VERBS + USER_UNPOOLED_VERBS:
        print(f"  {verb:12s} {counts.get(verb, 0)}")
    print(f"  ({pool_entry_rows} rows are pool entries beyond index 0)")

    if disagreements:
        print(f"\n{len(disagreements)} disagreement(s) found:")
        for d in disagreements:
            print(f"  - {d}")
    else:
        print("\nNo disagreements between fresh source extraction and the hand-authored tables.")

    return disagreements

def main_mobcommands():
    disagreements = []
    inventory = parse_inventory(INVENTORY_PATH)

    generic_rows, generic_counts = build_generic_rows(inventory, disagreements)
    shoot_rows, shoot_counts = build_shoot_rows(inventory, disagreements)

    all_rows = generic_rows + shoot_rows
    counts = {**generic_counts, **shoot_counts}

    check_against_shipped_store(all_rows, disagreements)

    os.makedirs(os.path.dirname(FIXTURE_PATH), exist_ok=True)
    with open(FIXTURE_PATH, "w", encoding="utf-8") as f:
        json.dump(all_rows, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"Wrote {len(all_rows)} fixture rows to {FIXTURE_PATH}")
    for verb in GENERIC_VERBS + ["shoot"]:
        print(f"  {verb:12s} {counts.get(verb, 0)}")

    if disagreements:
        print(f"\n{len(disagreements)} disagreement(s) found:")
        for d in disagreements:
            print(f"  - {d}")
    else:
        print("\nNo disagreements between fresh source extraction and the inventory.")

    return disagreements


def main():
    print("=== mobcommands ===")
    mob_disagreements = main_mobcommands()

    print("\n=== usercommands ===")
    user_disagreements = main_usercommands()

    total = len(mob_disagreements) + len(user_disagreements)
    return 1 if total else 0


if __name__ == "__main__":
    sys.exit(main())
