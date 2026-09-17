#!/usr/bin/env python3
"""Prove a role-key rename changed labels and nothing else (M4b-1).

Compares each golden at a base git ref against the working tree copy,
after applying the same label translation the rename applied. Any
difference that is not a label is a real change and is reported.

Usage:
  python tools/messaging_role_key_check.py --base HEAD --group combat
"""
import argparse
import re
import subprocess
import sys

GOLDEN_DIR = "internal/narration/testdata/stores"

# Per store group: golden file -> ordered (old, new) label pairs. Longest
# first, so toattackerroom is rewritten before toattacker.
GROUPS = {
    "combat": {
        "combat_messages.golden": [
            ("toattackerroom", "observer"), ("todefenderroom", "remote_observer"),
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
        "defense_messages.golden": [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
        "taunt_messages.golden": [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
    },
    "grapple": {
        "messaging_grapple.golden": [
            ("controller", "actor"), ("controlled", "actee"), ("observers", "observer"),
            ("self", "actor"), ("partner", "actee"),
        ],
    },
    "kindb": {
        "conditions.golden": [
            ("start_user_text", "start_actee"), ("start_room_text", "start_observer"),
            ("trigger_user_text", "trigger_actee"), ("trigger_room_text", "trigger_observer"),
            ("end_user_text", "end_actee"), ("end_room_text", "end_observer"),
        ],
        "spells.golden": [],
        "quests.golden": [
            ("playermessage", "actor"), ("roommessage", "observer"),
            ("send_text", "actor"), ("room_text", "observer"),
        ],
        "crafting.golden": [
            ("success_room_message", "success_observer"), ("failure_room_message", "failure_observer"),
            ("success_message", "success_actor"), ("failure_message", "failure_actor"),
        ],
    },
    "position": {
        "position_control.golden": [
            ("attacker", "actor"), ("target", "actee"), ("room", "observer"),
            ("self", "actor"),
        ],
    },
}


def golden_at(ref, name):
    out = subprocess.run(["git", "show", "%s:%s/%s" % (ref, GOLDEN_DIR, name)],
                         capture_output=True, text=True)
    if out.returncode != 0:
        raise SystemExit("cannot read %s at %s: %s" % (name, ref, out.stderr.strip()))
    return out.stdout


def translate(text, pairs):
    for old, new in pairs:
        # Labels appear in the row key ("|todefender =>") and inside values
        # ("todefender="). Both are word-bounded.
        text = re.sub(r"\b%s\b" % re.escape(old), new, text)
    return text


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True, help="git ref holding the pre-rename goldens")
    ap.add_argument("--group", required=True, choices=sorted(GROUPS))
    args = ap.parse_args()

    failures = 0
    for name, pairs in GROUPS[args.group].items():
        old = translate(golden_at(args.base, name), pairs)
        with open("%s/%s" % (GOLDEN_DIR, name), "r", encoding="utf-8", newline="") as fh:
            new = fh.read()
        if old == new:
            print("%s: identical after label translation" % name)
            continue
        failures += 1
        old_lines, new_lines = old.splitlines(), new.splitlines()
        for i in range(max(len(old_lines), len(new_lines))):
            o = old_lines[i] if i < len(old_lines) else "<missing>"
            n = new_lines[i] if i < len(new_lines) else "<missing>"
            if o != n:
                print("%s: first difference at line %d" % (name, i + 1))
                print("  translated old: %s" % o)
                print("  new:            %s" % n)
                break
    print("goldens differing beyond labels: %d" % failures)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
