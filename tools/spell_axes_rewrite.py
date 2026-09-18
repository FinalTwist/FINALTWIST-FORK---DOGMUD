#!/usr/bin/env python3
"""Rewrite spell YAML from `type:` + `target_defense_type:` to the three axes
(messaging M4b-2), and rewrite Go TEST fixtures that build SpellData with the
retiring fields.

Text-line edits only. NEVER yaml.load/yaml.dump: a round trip destroys
quoting and comment headers. Writes to <file>.tmp then os.replace, so a
crash cannot truncate a shipped file.

Modes:
  --add        insert attack_type/damage_type/targeting after the `type:` line,
               derived from the two legacy keys; keeps both legacy keys.
  --strip      delete the `type:` and `target_defense_type:` lines.
  --check      non-zero if any spell file lacks one of the three keys or still
               carries a legacy key.
  --go-tests   rewrite Go test files: `Type: spells.X,` struct fields (and a
               following `TargetDefenseType: "...",` line) become the three
               axis fields; bare `spells.X` arguments become constructor calls.
  --dry-run    with any of the above: print what would change, write nothing.

The mapping is the spec's table (docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md,
section 6). core-drain is the one special case (owner ruling: physical).
"""
import argparse
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SPELL_DIRS = [
    os.path.join(ROOT, "_datafiles", "world", "dogmud", "spells"),
    os.path.join(ROOT, "_datafiles", "world", "default", "spells"),
]
LEGACY_KEYS = ("type", "target_defense_type")
NEW_KEYS = ("attack_type", "damage_type", "targeting")

TARGETING = {
    "harmsingle": "single", "helpsingle": "single",
    "harmmulti": "multi", "helpmulti": "multi",
    "harmarea": "area", "helparea": "area",
    "neutral": "self",
}
HARM = {"harmsingle", "harmmulti", "harmarea"}
SPECIAL = {"core-drain": ("spell", "physical", "area")}


def derive(spell_id, legacy_type, legacy_def):
    """Return (attack_type, damage_type, targeting) for one spell."""
    if spell_id in SPECIAL:
        return SPECIAL[spell_id]
    if legacy_type not in TARGETING:
        raise ValueError(f"{spell_id}: unknown type {legacy_type!r}")
    targeting = TARGETING[legacy_type]
    if legacy_type in HARM:
        damage = legacy_def if legacy_def in ("physical", "mental", "social") else "mental"
        return ("spell", damage, targeting)
    return ("none", "non_harm", targeting)


KEY_RE = re.compile(r"^(?P<key>[a-z_]+):\s*(?P<val>[^#\n]*?)\s*(#.*)?$")


def read_key(lines, key):
    for line in lines:
        m = KEY_RE.match(line)
        if m and m.group("key") == key:
            return m.group("val").strip().strip('"').strip("'")
    return None


def write_swap(path, lines):
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8", newline="") as f:
        f.writelines(lines)
    os.replace(tmp, path)


def spell_files():
    for d in SPELL_DIRS:
        for name in sorted(os.listdir(d)):
            if name.endswith(".yaml"):
                yield os.path.join(d, name)


def do_add(dry):
    changed = 0
    for path in spell_files():
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        spell_id = read_key(lines, "spellid") or os.path.basename(path)[:-5]
        if read_key(lines, "attack_type") is not None:
            continue
        legacy_type = read_key(lines, "type")
        legacy_def = read_key(lines, "target_defense_type")
        attack, damage, targeting = derive(spell_id, legacy_type, legacy_def)
        out = []
        inserted = False
        for line in lines:
            out.append(line)
            m = KEY_RE.match(line)
            if m and m.group("key") == "type" and not inserted:
                nl = "\r\n" if line.endswith("\r\n") else "\n"
                out.append(f"attack_type: {attack}{nl}")
                out.append(f"damage_type: {damage}{nl}")
                out.append(f"targeting: {targeting}{nl}")
                inserted = True
        if not inserted:
            raise ValueError(f"{path}: no type: line to anchor on")
        changed += 1
        print(f"{'would add' if dry else 'add'} {spell_id}: {attack}/{damage}/{targeting}")
        if not dry:
            write_swap(path, out)
    print(f"{changed} files")


def do_strip(dry):
    changed = 0
    for path in spell_files():
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        out = [l for l in lines if not (KEY_RE.match(l) and KEY_RE.match(l).group("key") in LEGACY_KEYS)]
        if len(out) != len(lines):
            changed += 1
            print(f"{'would strip' if dry else 'strip'} {os.path.basename(path)}: {len(lines) - len(out)} lines")
            if not dry:
                write_swap(path, out)
    print(f"{changed} files")


def do_check():
    bad = 0
    seen = 0
    for path in spell_files():
        seen += 1
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        for k in NEW_KEYS:
            if read_key(lines, k) is None:
                print(f"MISSING {k}: {path}")
                bad += 1
        for k in LEGACY_KEYS:
            if read_key(lines, k) is not None:
                print(f"LEGACY {k}: {path}")
                bad += 1
    print(f"checked {seen} files, {bad} problems")
    if seen == 0:
        print("checked nothing: the spell directories are wrong")
        return 2
    return 1 if bad else 0


# Go test fixtures.
CTOR = {
    "Neutral": "combatvocab.NonHarm(combatvocab.TargetSelf)",
    "HelpSingle": "combatvocab.NonHarm(combatvocab.TargetSingle)",
    "HelpMulti": "combatvocab.NonHarm(combatvocab.TargetMulti)",
    "HelpArea": "combatvocab.NonHarm(combatvocab.TargetArea)",
    "HarmSingle": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetSingle)",
    "HarmMulti": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetMulti)",
    "HarmArea": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetArea)",
}
FIELDS = {
    "Neutral": ("AttackNone", "DamageNonHarm", "TargetSelf"),
    "HelpSingle": ("AttackNone", "DamageNonHarm", "TargetSingle"),
    "HelpMulti": ("AttackNone", "DamageNonHarm", "TargetMulti"),
    "HelpArea": ("AttackNone", "DamageNonHarm", "TargetArea"),
    "HarmSingle": ("AttackSpell", "Damage{D}", "TargetSingle"),
    "HarmMulti": ("AttackSpell", "Damage{D}", "TargetMulti"),
    "HarmArea": ("AttackSpell", "Damage{D}", "TargetArea"),
}
TYPE_FIELD_RE = re.compile(r"^(?P<indent>\s*)Type:(?P<sp>\s*)spells\.(?P<t>\w+),(?P<rest>.*)$")
INLINE_TYPE_RE = re.compile(r"\bType:\s*spells\.(?P<t>\w+),")
DEF_FIELD_RE = re.compile(r'^\s*TargetDefenseType:\s*"(?P<d>\w*)",\s*$')
INLINE_DEF_RE = re.compile(r'\s*TargetDefenseType:\s*"(?P<d>\w*)",')
BARE_RE = re.compile(r"\bspells\.(Neutral|HelpSingle|HelpMulti|HelpArea|HarmSingle|HarmMulti|HarmArea)\b")


def damage_word(d):
    return {"physical": "Physical", "mental": "Mental", "social": "Social"}.get(d, "Mental")


def fields_for(t, d):
    a, dm, tg = FIELDS[t]
    dm = dm.replace("{D}", damage_word(d))
    return f"AttackType: combatvocab.{a}, DamageType: combatvocab.{dm}, Targeting: combatvocab.{tg},"


def rewrite_go(path, dry):
    with open(path, encoding="utf-8", newline="") as f:
        lines = f.readlines()
    out = []
    i = 0
    changed = False
    while i < len(lines):
        line = lines[i]
        m = TYPE_FIELD_RE.match(line.rstrip("\r\n"))
        if m:
            nl = "\r\n" if line.endswith("\r\n") else "\n"
            # Look ahead (same literal, within 12 lines) for a TargetDefenseType line.
            d = ""
            for j in range(i + 1, min(i + 13, len(lines))):
                dm = DEF_FIELD_RE.match(lines[j])
                if dm:
                    d = dm.group("d")
                    del lines[j]
                    break
                if lines[j].strip() in ("}", "})", "},"):
                    break
            out.append(f"{m.group('indent')}{fields_for(m.group('t'), d)}{m.group('rest')}{nl}")
            changed = True
            i += 1
            continue
        im = INLINE_TYPE_RE.search(line)
        if im:
            d = ""
            dm = INLINE_DEF_RE.search(line)
            if dm:
                d = dm.group("d")
                line = line[:dm.start()] + line[dm.end():]
            line = INLINE_TYPE_RE.sub(lambda mm: fields_for(mm.group("t"), d), line, count=1)
            changed = True
        if BARE_RE.search(line):
            line = BARE_RE.sub(lambda mm: CTOR[mm.group(1)].replace("{D}", "Mental"), line)
            changed = True
        out.append(line)
        i += 1
    if changed:
        print(f"{'would rewrite' if dry else 'rewrite'} {os.path.relpath(path, ROOT)}")
        if not dry:
            write_swap(path, out)
    return changed


def do_go_tests(dry):
    n = 0
    for base in ("internal", "modules"):
        for dirpath, _, names in os.walk(os.path.join(ROOT, base)):
            for name in names:
                if name.endswith("_test.go"):
                    n += rewrite_go(os.path.join(dirpath, name), dry)
    print(f"{n} test files")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--add", action="store_true")
    ap.add_argument("--strip", action="store_true")
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--go-tests", action="store_true")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()
    if args.add:
        do_add(args.dry_run)
    elif args.strip:
        do_strip(args.dry_run)
    elif args.go_tests:
        do_go_tests(args.dry_run)
    elif args.check:
        sys.exit(do_check())
    else:
        ap.print_help()
        sys.exit(2)


if __name__ == "__main__":
    main()
