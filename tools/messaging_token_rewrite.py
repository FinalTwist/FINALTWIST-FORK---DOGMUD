#!/usr/bin/env python3
"""Rewrite shipped narration names to the canonical vocabulary (M4a, M4b-1).

Two modes, deliberately separate:

  --group <g>  rewrites TOKENS inside authored text ({source} -> {actor}).
  --keys <g>   rewrites YAML KEYS only (todefender: -> actee:), anchored to a
               line whose trimmed form starts `<key>:` or `- <key>:`, so the
               same word occurring in prose is never touched.

Text-line edits only. NEVER yaml.load/yaml.dump: a round trip destroys
quoting and comment headers. Writes to <file>.tmp then os.replace, so a
crash cannot truncate a shipped file.

Usage:
  python tools/messaging_token_rewrite.py --group kindb --dry-run
  python tools/messaging_token_rewrite.py --group kindb
  python tools/messaging_token_rewrite.py --keys combat --dry-run
  python tools/messaging_token_rewrite.py --keys combat
  python tools/messaging_token_rewrite.py --check      # non-zero if any old token remains
"""
import argparse
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
W = os.path.join(ROOT, "_datafiles", "world", "dogmud")
# internal/configs/config.filepaths.go falls back to _datafiles/world/default
# when the DataFiles config key is empty, so that tree is a SHIPPED world too
# and its stores must flip with dogmud's. Its combat-messages is the only
# directory under it holding any of these tokens (checked M4a task 4).
DEFAULT_W = os.path.join(ROOT, "_datafiles", "world", "default")
MSG = os.path.join(ROOT, "_datafiles", "messages")

# Per (store, key), never global: the same spelling means different roles in
# different stores. conditions' {source} is the HOLDER, which is the actee.
GROUPS = {
    "kindb": [
        (os.path.join(W, "conditions"), {
            "{source}": "{actee}", "{source_plain}": "{actee_plain}",
            "{target}": "{actor}", "{target_plain}": "{actor_plain}",
        }),
        (os.path.join(W, "spells"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
            "{target}": "{actee}", "{target_plain}": "{actee_plain}",
        }),
        (os.path.join(W, "quests"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
        }),
        (os.path.join(W, "recipes"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
        }),
    ],
    "items": [
        (os.path.join(W, "combat-messages"), {
            "{source}": "{actor}", "{target}": "{actee}",
            "{sourcetype}": "{actortype}", "{targettype}": "{acteetype}",
        }),
        (os.path.join(DEFAULT_W, "combat-messages"), {
            "{source}": "{actor}", "{target}": "{actee}",
            "{sourcetype}": "{actortype}", "{targettype}": "{acteetype}",
        }),
        (os.path.join(W, "defense-messages"), {
            "{attacker}": "{actor}", "{defender}": "{actee}",
        }),
        (os.path.join(W, "taunt-messages"), {
            "{source}": "{actor}", "{target}": "{actee}",
            "{sourcetype}": "{actortype}", "{targettype}": "{acteetype}",
        }),
    ],
    "grapple": [
        (os.path.join(W, "messaging"), {
            "{controllerName}": "{actor}", "{controlledName}": "{actee}",
        }),
    ],
    "position": [
        (MSG, {
            "{attacker}": "{actor}", "{target}": "{actee}",
            "{Controller}": "{actor}", "{Controlled}": "{actee}",
            "{Character}": "{actor}",
        }),
    ],
}

# Role KEYS, an ORDERED list per file, not a dict: `toattackerroom` must be
# rewritten before `toattacker` or it becomes `actorroom`. Nothing here may
# sort the pairs.
#
# `observer` appearing twice in the combat-messages table is correct, not a
# typo: a `together` group's room line and a `separate` group's attacker-room
# line are the same audience, the observers standing with the actor.
#
# _datafiles/world/default is deliberately ABSENT. The owner ruled on
# 2026-09-17 that the vestigial default world is left alone: it cannot boot
# (it has no defense-messages directory, so the item loader panics first) and
# nothing loads its combat-messages, since every test that loads these stores
# points FilePaths.DataFiles at the dogmud world first.
KEY_GROUPS = {
    # grapple points at the FILE, not at W/messaging, and that is load-bearing.
    # position_control.yaml sits in the same directory and authors `controller:`,
    # `controlled:` and `self:` keys of its own, which belong to the POSITION
    # group a later task renames. A directory target would rewrite them here,
    # silently folding two stores into one task. files_under() accepts a file
    # path, so naming the file costs nothing.
    "grapple": [
        (os.path.join(W, "messaging", "grapple_outcomes.yaml"), [
            ("controller", "actor"), ("controlled", "actee"), ("observers", "observer"),
            ("self", "actor"), ("partner", "actee"),
        ]),
    ],
    "combat": [
        (os.path.join(W, "defense-messages"), [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
        (os.path.join(W, "combat-messages"), [
            ("toattackerroom", "observer"), ("todefenderroom", "remote_observer"),
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
        (os.path.join(W, "taunt-messages"), [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
    ],
}


def files_under(path):
    if os.path.isfile(path):
        return [path]
    out = []
    for dirpath, _, names in os.walk(path):
        for n in sorted(names):
            if n.endswith(".yaml") or n.endswith(".yml"):
                out.append(os.path.join(dirpath, n))
    return sorted(out)


def rewrite(path, table, dry_run):
    with open(path, "r", encoding="utf-8", newline="") as fh:
        original = fh.read()
    updated = original
    for old, new in sorted(table.items(), key=lambda kv: -len(kv[0])):
        updated = updated.replace(old, new)
    if updated == original:
        return 0
    hits = sum(original.count(old) for old in table)
    if not dry_run:
        tmp = path + ".tmp"
        with open(tmp, "w", encoding="utf-8", newline="") as fh:
            fh.write(updated)
        os.replace(tmp, path)
    return hits


def rewrite_keys(path, pairs, dry_run):
    """Rewrite YAML role KEYS, and only keys.

    A key line is one whose trimmed form starts `<key>:` or `- <key>:`. Every
    other occurrence of the word is authored prose or a token inside a value,
    and must survive untouched: combat-messages values are full of
    `{actor}`/`{actee}` and of ordinary English.

    The pairs are applied in the order given, longest first, so
    `toattackerroom` is consumed before `toattacker` is tried. The trailing
    colon in the pattern is a second line of defence for the same hazard.
    """
    with open(path, "r", encoding="utf-8", newline="") as fh:
        original = fh.read()

    hits = 0
    out = []
    for line in original.splitlines(keepends=True):
        for old, new in pairs:
            pattern = r"^(\s*(?:-\s+)?)%s(?=:)" % re.escape(old)
            replaced, n = re.subn(pattern, lambda m: m.group(1) + new, line)
            if n:
                line = replaced
                hits += n
                break
        out.append(line)

    updated = "".join(out)
    if updated == original:
        return 0
    if not dry_run:
        tmp = path + ".tmp"
        with open(tmp, "w", encoding="utf-8", newline="") as fh:
            fh.write(updated)
        os.replace(tmp, path)
    return hits


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--group", choices=sorted(GROUPS))
    ap.add_argument("--keys", choices=sorted(KEY_GROUPS))
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()

    if args.check:
        stale = []
        for group in GROUPS.values():
            for path, table in group:
                for f in files_under(path):
                    with open(f, "r", encoding="utf-8", newline="") as fh:
                        body = fh.read()
                    for old in table:
                        if old in body:
                            stale.append("%s: %s x%d" % (f, old, body.count(old)))
        for row in stale:
            print(row)
        print("stale token occurrences: %d" % len(stale))
        return 1 if stale else 0

    if args.keys:
        if args.group:
            ap.error("--group and --keys are separate modes; pass one")
        total, touched = 0, 0
        for path, pairs in KEY_GROUPS[args.keys]:
            for f in files_under(path):
                hits = rewrite_keys(f, pairs, args.dry_run)
                if hits:
                    touched += 1
                    total += hits
                    print("%s%s: %d" % ("[dry-run] " if args.dry_run else "", f, hits))
        print("files touched: %d, keys rewritten: %d" % (touched, total))
        return 0

    if not args.group:
        ap.error("--group is required unless --check or --keys is given")
    total, touched = 0, 0
    for path, table in GROUPS[args.group]:
        for f in files_under(path):
            hits = rewrite(f, table, args.dry_run)
            if hits:
                touched += 1
                total += hits
                print("%s%s: %d" % ("[dry-run] " if args.dry_run else "", f, hits))
    print("files touched: %d, tokens rewritten: %d" % (touched, total))
    return 0


if __name__ == "__main__":
    sys.exit(main())
