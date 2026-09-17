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
        "spells.golden": [
            ("cast_user_text", "cast_actor"), ("cast_room_text", "cast_observer"),
            ("wait_user_text", "wait_actor"), ("wait_room_text", "wait_observer"),
            ("magic_user_text", "magic_actor"), ("magic_room_text", "magic_observer"),
        ],
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
    # BYTES, decoded as UTF-8 by hand. subprocess's text=True decodes with the
    # locale codec, which on Windows is cp1252: taunt_messages.golden holds an
    # em dash, and that round trip turned it into mojibake, so the check
    # reported a difference in a row the rename never touched.
    out = subprocess.run(["git", "show", "%s:%s/%s" % (ref, GOLDEN_DIR, name)],
                         capture_output=True)
    if out.returncode != 0:
        raise SystemExit("cannot read %s at %s: %s"
                         % (name, ref, out.stderr.decode("utf-8", "replace").strip()))
    return out.stdout.decode("utf-8")


def translate(text, pairs):
    """Rewrite role labels in the two positions a golden puts them, and NOWHERE
    else.

    A bare word-boundary replace is wrong, and the grapple golden proves it:
    line 120 ends "you're fully controlled." That is prose, and the real rename
    does not touch it, so translating it would bake a permanent false failure
    into the check. Labels only ever appear as

      "|<label> =>"     the row key's last field, every store
      "|<label>|"       the row key's middle field, combat_messages' derived row
      "<label>="        inside the value, combat_messages only

    so all three patterns are anchored to that shape instead. The middle-field
    one was added in M4b-1 task 4 for combat_messages.golden's
    "derived|bite|weak|actor|skill=10|n=4 =>" rows, where the label is neither
    last in the key nor followed by "=". It is as safe as the other two: a
    pipe-delimited field is never prose.
    """
    for old, new in pairs:
        text = re.sub(r"\|%s(?= =>)" % re.escape(old), "|" + new, text)
        text = re.sub(r"\|%s(?=\|)" % re.escape(old), "|" + new, text)
        text = re.sub(r"(?<![\w-])%s=" % re.escape(old), new + "=", text)
    return text


def rows(text):
    """The data rows of a golden: every line that is not a generated comment.

    Header comments are provenance written by snapshot_test.go, and the rename
    updates their wording by hand. Holding them to label translation would fail
    on a correct rename, so they are reported separately instead of compared.
    """
    return [ln for ln in text.splitlines() if not ln.startswith("#")]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True, help="git ref holding the pre-rename goldens")
    ap.add_argument("--group", required=True, choices=sorted(GROUPS))
    args = ap.parse_args()

    failures = 0
    for name, pairs in GROUPS[args.group].items():
        old_text = translate(golden_at(args.base, name), pairs)
        with open("%s/%s" % (GOLDEN_DIR, name), "r", encoding="utf-8", newline="") as fh:
            new_text = fh.read()
        old_lines, new_lines = rows(old_text), rows(new_text)
        header_moved = old_text.splitlines()[:len(old_text.splitlines()) - len(old_lines)] != \
            new_text.splitlines()[:len(new_text.splitlines()) - len(new_lines)]
        note = " (header comments differ, not compared)" if header_moved else ""
        if old_lines == new_lines:
            print("%s: %d rows identical after label translation%s" % (name, len(new_lines), note))
            continue
        failures += 1
        for i in range(max(len(old_lines), len(new_lines))):
            o = old_lines[i] if i < len(old_lines) else "<missing>"
            n = new_lines[i] if i < len(new_lines) else "<missing>"
            if o != n:
                print("%s: first differing row %d of %d%s" % (name, i + 1, len(new_lines), note))
                print("  translated old: %s" % o)
                print("  new:            %s" % n)
                break
    print("goldens differing beyond labels: %d" % failures)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
