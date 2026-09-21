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


def main():
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

    return 1 if disagreements else 0


if __name__ == "__main__":
    sys.exit(main())
