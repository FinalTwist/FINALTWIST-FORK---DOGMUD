#!/usr/bin/env python3
"""Rewrite shipped narration name tokens to the canonical vocabulary (M4a).

Text-line edits only. NEVER yaml.load/yaml.dump: a round trip destroys
quoting and comment headers. Writes to <file>.tmp then os.replace, so a
crash cannot truncate a shipped file.

Usage:
  python tools/messaging_token_rewrite.py --group kindb --dry-run
  python tools/messaging_token_rewrite.py --group kindb
  python tools/messaging_token_rewrite.py --check      # non-zero if any old token remains
"""
import argparse
import os
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


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--group", choices=sorted(GROUPS))
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

    if not args.group:
        ap.error("--group is required unless --check is given")
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
