#!/usr/bin/env python3
"""Prove the counters-slice golden regeneration changed only what it should.

Compares defense_messages.golden at a base git ref against the working tree
copy. Rows are `pool|band|role => text` and `melee|pool|band|role => text`.

Expected differences, and ONLY these:
  - every counter-melee row reappears as counter-dodge with IDENTICAL text;
  - every counter-ranged row is gone;
  - counter-parry and counter-block rows are new;
  - counter-defy rows may change (re-toned);
  - every other row (dodge, parry, block, quell, defy, counter-quell) is
    byte-identical.
Header comment lines are reported, not compared.

Usage:
  python tools/counter_pool_rekey_check.py --base master
"""
import argparse
import subprocess
import sys

GOLDEN = "internal/narration/testdata/stores/defense_messages.golden"


def rows(text):
    out = {}
    for line in text.splitlines():
        if not line or line.startswith("#"):
            continue
        key, sep, value = line.partition(" => ")
        if not sep:
            continue
        out[key] = value
    return out


def pool_of(key):
    parts = key.split("|")
    return parts[1] if parts[0] == "melee" else parts[0]


def translate(key):
    return key.replace("counter-melee|", "counter-dodge|", 1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="master")
    args = ap.parse_args()

    # bytes, decoded explicitly: text=True would use the Windows locale codec
    # and mojibake any non-ASCII character into a false difference.
    base_bytes = subprocess.run(["git", "show", f"{args.base}:{GOLDEN}"],
                                check=True, capture_output=True).stdout
    base = rows(base_bytes.decode("utf-8"))
    with open(GOLDEN, encoding="utf-8") as fh:
        new = rows(fh.read())

    problems = []
    seen_new = set()
    for key, text in base.items():
        pool = pool_of(key)
        if pool == "counter-ranged":
            if key in new:
                problems.append(f"counter-ranged row survived: {key}")
            continue
        target = translate(key) if pool == "counter-melee" else key
        seen_new.add(target)
        if target not in new:
            problems.append(f"row missing after regeneration: {target}")
        elif pool == "counter-defy":
            continue
        elif new[target] != text:
            problems.append(f"text changed: {target}\n  base: {text}\n  new:  {new[target]}")

    for key in new:
        if key in seen_new:
            continue
        pool = pool_of(key)
        if pool in ("counter-parry", "counter-block"):
            continue
        problems.append(f"unexpected new row: {key}")

    counts = {}
    for key in new:
        counts[pool_of(key)] = counts.get(pool_of(key), 0) + 1
    print("rows per pool after regeneration:", dict(sorted(counts.items())))
    if problems:
        print("\n".join(problems))
        print(f"\nFAIL: {len(problems)} problem(s)")
        return 1
    print("OK: dodge rows are the old melee rows, ranged rows gone, parry/block new, defy re-toned, rest identical")
    return 0


if __name__ == "__main__":
    sys.exit(main())
