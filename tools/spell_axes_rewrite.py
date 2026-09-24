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
  --go-tests   rewrite Go test files: a `Type: spells.X,` struct field (start
               of line or mid-line, alongside other fields) becomes the three
               axis fields, pulling its damage from a same-line or lookahead
               `TargetDefenseType: "...",`; bare `spells.X` arguments become
               constructor calls; package spells files see the same shapes
               without the `spells.` prefix. Prints a LEFTOVER line and exits
               non-zero for every _test.go line the pass could not rewrite
               (a variable Type field, a lone TargetDefenseType, a SpellType
               slice/map type, ...).
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
# The seven legacy SpellType names, shared by every regex and lookup table
# below so the set can never drift between them.
TYPENAMES = "(?:" + "|".join(FIELDS.keys()) + ")"

# Qualified (`spells.HarmSingle`) is required outside package spells; inside
# package spells the same identifiers appear bare (`HarmSingle`), so the
# `spells.` prefix is optional there. Either way the match spans the whole
# `Type: ...,` field (wherever it sits on the line), so a mid-line field next
# to other fields (SpellId, Name, ...) is found exactly like one that opens
# the line.
TYPE_ANY_QUALIFIED_RE = re.compile(rf"\bType:\s*spells\.(?P<t>{TYPENAMES}),")
TYPE_ANY_UNQUALIFIED_RE = re.compile(rf"\bType:\s*(?:spells\.)?(?P<t>{TYPENAMES}),")
# Matches a TargetDefenseType field wherever it sits on a line, consuming its
# own leading whitespace and trailing comma so removing the match leaves the
# rest of the line (other fields, or a trailing comment) intact.
INLINE_DEF_RE = re.compile(r'\s*TargetDefenseType:\s*"(?P<d>\w*)",')
BARE_RE = re.compile(r"\bspells\.(Neutral|HelpSingle|HelpMulti|HelpArea|HarmSingle|HarmMulti|HarmArea)\b")

# Leftover detector (item 3): after a --go-tests pass, any of these appearing
# in a _test.go file is a shape the pass could not rewrite (a variable Type
# field, a TargetDefenseType with no accompanying Type field, a SpellType
# slice/map type, etc). Printed and reported non-zero so a partial sweep
# cannot pass as complete; Task 7 fixes these by hand.
LEFTOVER_COMMON_RE = re.compile(rf"TargetDefenseType|spells\.SpellType|SpellType\{{|spells\.{TYPENAMES}\b")
# Package-spells files also spell these names bare; a bare Type: field that
# --go-tests did not consume is a leftover there too. (Other bare uses of
# these names, not preceded by `Type:`, are left for the compiler per the
# fix's own rule -- BARE_RE never fires without the `spells.` prefix.)
LEFTOVER_PKG_SPELLS_RE = re.compile(rf"\bType:\s*{TYPENAMES}\b")


def damage_word(d):
    return {"physical": "Physical", "mental": "Mental", "social": "Social"}.get(d, "Mental")


def fields_for(t, d):
    a, dm, tg = FIELDS[t]
    dm = dm.replace("{D}", damage_word(d))
    return f"AttackType: combatvocab.{a}, DamageType: combatvocab.{dm}, Targeting: combatvocab.{tg},"


def first_code_line(lines):
    """The first non-blank, non-comment line, stripped -- used to tell a
    package spells file (bare enum names) from every other package
    (qualified spells.X names)."""
    for line in lines:
        s = line.strip()
        if s == "" or s.startswith("//"):
            continue
        return s
    return ""


def rewrite_go(path, dry):
    with open(path, encoding="utf-8", newline="") as f:
        lines = f.readlines()
    type_re = TYPE_ANY_UNQUALIFIED_RE if first_code_line(lines) == "package spells" else TYPE_ANY_QUALIFIED_RE
    out = []
    i = 0
    changed = False
    while i < len(lines):
        line = lines[i]
        m = type_re.search(line)
        if m:
            nl = "\r\n" if line.endswith("\r\n") else "\n"
            bare = line.rstrip("\r\n")
            # Same-line TargetDefenseType first; only if absent, look ahead
            # (same literal, within 12 lines, stopping at its closing brace)
            # for one -- which may share its line with other fields (the
            # bug this fix closes: it used to only check the SAME line as
            # Type, silently defaulting a mid-line Type's damage to mental).
            dm = INLINE_DEF_RE.search(bare)
            if dm:
                d = dm.group("d")
                bare = bare[:dm.start()] + bare[dm.end():]
                m = type_re.search(bare)  # the removal may have shifted it
            else:
                d = None
                for j in range(i + 1, min(i + 13, len(lines))):
                    ddm = INLINE_DEF_RE.search(lines[j])
                    if ddm:
                        d = ddm.group("d")
                        rest = lines[j][:ddm.start()] + lines[j][ddm.end():]
                        if rest.strip() == "":
                            del lines[j]
                        else:
                            lines[j] = rest
                        break
                    if lines[j].strip().startswith("}"):
                        break
                if d is None:
                    d = ""
            prefix = bare[:m.start()]
            suffix = bare[m.end():]
            out.append(f"{prefix}{fields_for(m.group('t'), d)}{suffix}{nl}")
            changed = True
            i += 1
            continue
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


def go_test_files():
    for base in ("internal", "modules"):
        for dirpath, _, names in os.walk(os.path.join(ROOT, base)):
            for name in names:
                if name.endswith("_test.go"):
                    yield os.path.join(dirpath, name)


def scan_leftovers():
    """Print a LEFTOVER line for every _test.go line a --go-tests pass could
    not (or, on --dry-run, will not) rewrite. Scans the files as they stand
    when called -- after the real rewrite for a live run, or untouched for a
    dry run, per the item-3 ruling that a dry-run simulation is not required."""
    bad = []
    for path in go_test_files():
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        pkg_spells = first_code_line(lines) == "package spells"
        for lineno, line in enumerate(lines, start=1):
            hit = LEFTOVER_COMMON_RE.search(line)
            if not hit and pkg_spells:
                hit = LEFTOVER_PKG_SPELLS_RE.search(line)
            if hit:
                rel = os.path.relpath(path, ROOT)
                print(f"LEFTOVER {rel}:{lineno}: {line.rstrip(chr(13) + chr(10))}")
                bad.append((rel, lineno))
    return bad


def do_go_tests(dry):
    n = 0
    for path in go_test_files():
        n += rewrite_go(path, dry)
    print(f"{n} test files")
    bad = scan_leftovers()
    print(f"{len(bad)} leftover lines")
    return 1 if bad else 0


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
        sys.exit(do_go_tests(args.dry_run))
    elif args.check:
        sys.exit(do_check())
    else:
        ap.print_help()
        sys.exit(2)


if __name__ == "__main__":
    main()
