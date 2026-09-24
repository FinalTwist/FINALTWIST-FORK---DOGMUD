"""Check a combat-message pad against a baseline git ref.

M3 item 8 PR 1 both APPENDS new lines and REORDERS existing ones, so that
index N describes the same moment in every role. Reordering means the golden
diff can no longer be "additions only", and a plain diff of the YAML cannot
tell a moved line from an edited one.

This compares, per (file, verb, split, role, tier) group, the multiset of
authored lines against a baseline ref. The contract it enforces:

  * no line is DELETED  (every baseline line still exists in its own group)
  * no line is EDITED   (an edit shows up as one deletion plus one addition)
  * no line MIGRATES    (groups are compared independently)
  * lines may be REORDERED freely within their group
  * lines may be ADDED

READ ONLY. Never write YAML from Python: yaml.dump would destroy the token
comment header, the quoting and the key order of every file it touched.

Usage:
    python tools/combat_message_pad_check.py <baseline-ref> [subtype]

Exit 0 when the contract holds, 1 on any deletion or edit.
"""
import collections
import glob
import os
import subprocess
import sys

import yaml

STORE = os.path.join("_datafiles", "world", "dogmud", "combat-messages")
TIERS = ["beginner", "expert", "master"]


def pools(doc):
    """Return {(verb, split, role, tier): [lines]} for one parsed document."""
    out = {}
    for verb, verb_body in ((doc or {}).get("options") or {}).items():
        for split, split_body in (verb_body or {}).items():
            if not split_body:
                continue
            for role, role_body in split_body.items():
                if not isinstance(role_body, dict):
                    continue
                for tier in TIERS:
                    lines = role_body.get(tier) or []
                    if lines:
                        out[(verb, split, role, tier)] = list(lines)
    return out


def baseline_doc(ref, path):
    rel = path.replace(os.sep, "/")
    try:
        blob = subprocess.run(
            ["git", "show", f"{ref}:{rel}"],
            capture_output=True, check=True, text=True, encoding="utf-8",
        ).stdout
    except subprocess.CalledProcessError:
        return None
    return yaml.safe_load(blob)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 1
    ref = sys.argv[1]
    only = sys.argv[2] if len(sys.argv) > 2 else None
    pattern = f"{only}.yaml" if only else "*.yaml"
    paths = sorted(glob.glob(os.path.join(STORE, pattern)))
    if not paths:
        print(f"no files matched {pattern} under {STORE}")
        return 1

    bad = 0
    added_total = 0
    moved_total = 0
    for path in paths:
        name = os.path.basename(path)
        before = baseline_doc(ref, path)
        if before is None:
            print(f"{name}: not present at {ref}, skipping")
            continue
        after = yaml.safe_load(open(path, "r", encoding="utf-8"))

        b_pools = pools(before)
        a_pools = pools(after)
        rows = []
        added = 0
        moved = 0

        for key in sorted(set(b_pools) | set(a_pools)):
            b_lines = b_pools.get(key, [])
            a_lines = a_pools.get(key, [])
            b_count = collections.Counter(b_lines)
            a_count = collections.Counter(a_lines)

            lost = b_count - a_count
            gained = a_count - b_count
            if lost:
                for text, n in lost.items():
                    rows.append(f"    LOST  {'/'.join(key)} x{n}: {text[:90]}")
            added += sum(gained.values())
            # A line that stayed but changed position.
            if not lost:
                common = [x for x in b_lines if a_count[x]]
                pos = {x: i for i, x in enumerate(a_lines)}
                for i, x in enumerate(common):
                    if pos.get(x) != i:
                        moved += 1
                        break

        if rows:
            bad += 1
            print(f"{name}")
            for r in rows:
                print(r)
        added_total += added
        moved_total += moved

    print()
    if bad:
        print(f"CONTRACT VIOLATED in {bad} file(s): a line was deleted or edited.")
        print("A reorder never loses a line. Restore the text, then move it.")
        return 1
    print(f"OK against {ref}: nothing deleted or edited. "
          f"{added_total} line(s) added, {moved_total} group(s) reordered.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
