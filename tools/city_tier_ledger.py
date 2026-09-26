#!/usr/bin/env python3
"""
DOGMud City Tier Ledger Helper

READ-ONLY tool for the graded room lighting plan 3c city pass. It feeds and
checks the markdown ledger a classifier fills in while sorting every
`biome: city` room in the New Plymouth zones, every `biome: fort` room, and
world-wide ruin candidates into the new lighting biomes. It writes NOTHING
but stdout; it never edits a room file or the ledger itself.

Subcommands:

    candidates <zone>...
        Prints a markdown table of every `biome: city` or `biome: fort`
        room in the named zones: id, title, current biome, exits
        (direction -> room id), and the first 240 characters of the
        description, one row per room, for the classifier to read
        alongside the full room file.

    ruins-scan
        Prints every room in the world, on any biome, whose DESCRIPTION
        (not title) matches a set of roofless/collapsed/open-sky phrases,
        with id, zone, title, biome and the matching sentence.

    check <ledger.md>
        Reads the ledger's markdown tables and, for every row classed
        `city_thoroughfare`, reports it if none of its exits leads to
        another room classed `city_thoroughfare` (by the ledger, or by
        the room's current biome when the neighbour is not in the
        ledger). Also reports any ledger row whose room id does not
        exist, or appears more than once in the ledger.

Room YAML lives at
    _datafiles/world/dogmud/rooms/<zone>/<roomid>.yaml
A `zone-config.yaml` file in the same folder is not a room and is
skipped, along with any other non-numeric filename.

Usage:
    python tools/city_tier_ledger.py candidates new_plymouth_temple
    python tools/city_tier_ledger.py ruins-scan
    python tools/city_tier_ledger.py check docs/superpowers/plans/city-tier-ledger.md
"""

import argparse
import re
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
ROOMS_ROOT = PROJECT_ROOT / "_datafiles" / "world" / "dogmud" / "rooms"

FILENAME_RE = re.compile(r"^(\d+)\.yaml$")

RUINS_PATTERN = re.compile(
    r"roofless|open to the sky|roof (has |had )?(fallen|collapsed|gone)|"
    r"collapsed roof|burnt[- ]out|burned[- ]out|no roof|"
    r"sky (shows|shines) through|caved[- ]in roof",
    re.IGNORECASE,
)

# A ledger table row: | Room | Title | Old | New | Reason |
LEDGER_ROW_RE = re.compile(r"^\|(.+)\|\s*$")

CITY_THOROUGHFARE = "city_thoroughfare"


def load_room(path):
    """Parse one room YAML file. Returns (data, error) — error is None on
    success, or a message string if the file failed to parse."""
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = yaml.safe_load(f)
    except Exception as exc:  # noqa: BLE001 - report and keep going
        return None, f"{path}: failed to parse ({exc})"
    if not isinstance(data, dict):
        return None, f"{path}: did not parse to a mapping"
    return data, None


def iter_room_files(zone_dir):
    """Yield (roomid, path) for every room file in a zone folder, in
    ascending numeric order. Skips zone-config.yaml and anything else
    that is not a bare numeric filename."""
    if not zone_dir.is_dir():
        return
    entries = []
    for path in zone_dir.iterdir():
        if not path.is_file() or path.suffix != ".yaml":
            continue
        m = FILENAME_RE.match(path.name)
        if not m:
            continue
        entries.append((int(m.group(1)), path))
    for roomid, path in sorted(entries):
        yield roomid, path


def iter_all_zones():
    """Yield zone directory paths under the rooms root, sorted by name."""
    if not ROOMS_ROOT.is_dir():
        return
    for path in sorted(ROOMS_ROOT.iterdir()):
        if path.is_dir():
            yield path


def format_exits(data):
    """Render exits as 'dir->roomid, dir->roomid, ...'."""
    exits = data.get("exits")
    if not isinstance(exits, dict):
        return ""
    parts = []
    for direction, target in sorted(exits.items()):
        roomid = target.get("roomid") if isinstance(target, dict) else None
        parts.append(f"{direction}->{roomid}")
    return ", ".join(parts)


def get_exit_targets(data):
    """Return a list of target room ids from a room's exits."""
    exits = data.get("exits")
    if not isinstance(exits, dict):
        return []
    targets = []
    for target in exits.values():
        if isinstance(target, dict) and target.get("roomid") is not None:
            targets.append(target.get("roomid"))
    return targets


def clean_field(value):
    """Flatten a YAML scalar into single-line text safe for a table cell."""
    if value is None:
        return ""
    text = str(value)
    text = text.replace("\r\n", " ").replace("\n", " ").replace("\r", " ")
    text = re.sub(r"\s+", " ", text).strip()
    text = text.replace("|", "/")
    return text


def cmd_candidates(zones):
    errors = []
    rows = []
    for zone in zones:
        zone_dir = ROOMS_ROOT / zone
        if not zone_dir.is_dir():
            errors.append(f"zone not found: {zone} ({zone_dir})")
            continue
        for roomid, path in iter_room_files(zone_dir):
            data, err = load_room(path)
            if err:
                errors.append(err)
                continue
            biome = data.get("biome")
            if biome not in ("city", "fort"):
                continue
            title = clean_field(data.get("title"))
            desc = clean_field(data.get("description"))[:240]
            exits = format_exits(data)
            rows.append((zone, roomid, title, biome, exits, desc))

    print("| Zone | Room | Title | Biome | Exits | Description (first 240 chars) |")
    print("|---|---|---|---|---|---|")
    for zone, roomid, title, biome, exits, desc in rows:
        print(f"| {zone} | {roomid} | {title} | {biome} | {exits} | {desc} |")

    for err in errors:
        print(f"# ERROR: {err}", file=sys.stderr)

    return rows, errors


def cmd_ruins_scan():
    errors = []
    rows = []
    for zone_dir in iter_all_zones():
        zone = zone_dir.name
        for roomid, path in iter_room_files(zone_dir):
            data, err = load_room(path)
            if err:
                errors.append(err)
                continue
            desc = data.get("description")
            if not desc:
                continue
            desc_text = clean_field(desc)
            m = RUINS_PATTERN.search(desc_text)
            if not m:
                continue
            title = clean_field(data.get("title"))
            biome = clean_field(data.get("biome"))
            # Grab a short window of context around the match as the
            # "matching sentence".
            start = max(0, m.start() - 60)
            end = min(len(desc_text), m.end() + 60)
            snippet = desc_text[start:end].strip()
            rows.append((roomid, zone, title, biome, snippet))

    print("| Room | Zone | Title | Biome | Matching text |")
    print("|---|---|---|---|---|")
    for roomid, zone, title, biome, snippet in rows:
        print(f"| {roomid} | {zone} | {title} | {biome} | {snippet} |")

    for err in errors:
        print(f"# ERROR: {err}", file=sys.stderr)

    return rows, errors


def parse_ledger(ledger_path):
    """Parse the ledger's markdown tables. Returns a list of dicts with
    keys room, title, old, new, reason, line (1-based line number)."""
    rows = []
    with open(ledger_path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    for lineno, raw in enumerate(lines, start=1):
        line = raw.rstrip("\r\n")
        m = LEDGER_ROW_RE.match(line)
        if not m:
            continue
        cells = [c.strip() for c in m.group(1).split("|")]
        if len(cells) != 5:
            continue
        room, title, old, new, reason = cells
        # Skip header rows ("Room"/"Old"/... labels).
        if room.lower() == "room":
            continue
        # Skip separator rows ("---", ":---", etc).
        if re.fullmatch(r":?-+:?", room):
            continue
        if not re.fullmatch(r"\d+", room):
            # Not a data row we can key by room id (e.g. a summary table
            # row); skip it silently.
            continue
        rows.append(
            {
                "room": int(room),
                "title": title,
                "old": old,
                "new": new,
                "reason": reason,
                "line": lineno,
            }
        )
    return rows


def load_all_rooms():
    """Return {roomid: data} for every room in the world, plus a list of
    parse-error messages."""
    rooms = {}
    errors = []
    for zone_dir in iter_all_zones():
        for roomid, path in iter_room_files(zone_dir):
            data, err = load_room(path)
            if err:
                errors.append(err)
                continue
            rooms[roomid] = data
    return rooms, errors


def cmd_check(ledger_path):
    ledger_path = Path(ledger_path)
    if not ledger_path.is_file():
        print(f"ERROR: ledger not found: {ledger_path}", file=sys.stderr)
        return 1

    ledger_rows = parse_ledger(ledger_path)
    rooms, room_errors = load_all_rooms()

    problems = []

    for err in room_errors:
        problems.append(f"room parse error: {err}")

    # Duplicate room ids in the ledger.
    seen = {}
    for row in ledger_rows:
        seen.setdefault(row["room"], []).append(row["line"])
    for roomid, lines in seen.items():
        if len(lines) > 1:
            problems.append(
                f"room {roomid}: appears {len(lines)} times in the ledger "
                f"(lines {', '.join(str(l) for l in lines)})"
            )

    # Nonexistent room ids.
    for row in ledger_rows:
        if row["room"] not in rooms:
            problems.append(
                f"room {row['room']} (line {row['line']}): not found under "
                f"{ROOMS_ROOT}"
            )

    # Build a lookup of the ledger's classification per room id (last
    # occurrence wins, duplicates are already reported above).
    ledger_new_by_room = {}
    for row in ledger_rows:
        ledger_new_by_room[row["room"]] = row["new"]

    # Thoroughfare connectivity check.
    for row in ledger_rows:
        if row["new"] != CITY_THOROUGHFARE:
            continue
        roomid = row["room"]
        data = rooms.get(roomid)
        if data is None:
            continue  # already reported as nonexistent
        targets = get_exit_targets(data)
        if not targets:
            problems.append(
                f"room {roomid} '{row['title']}' (line {row['line']}): "
                f"classed {CITY_THOROUGHFARE} but has no exits"
            )
            continue
        has_thoroughfare_neighbour = False
        for target in targets:
            if target in ledger_new_by_room:
                neighbour_class = ledger_new_by_room[target]
            elif target in rooms:
                neighbour_class = rooms[target].get("biome")
            else:
                continue
            if neighbour_class == CITY_THOROUGHFARE:
                has_thoroughfare_neighbour = True
                break
        if not has_thoroughfare_neighbour:
            problems.append(
                f"room {roomid} '{row['title']}' (line {row['line']}): "
                f"classed {CITY_THOROUGHFARE} but no exit leads to another "
                f"{CITY_THOROUGHFARE} room"
            )

    if not problems:
        print(f"OK: {len(ledger_rows)} ledger rows checked, no problems found.")
        return 0

    print(f"{len(problems)} problem(s) found in {ledger_path}:")
    for problem in problems:
        print(f"  - {problem}")
    return 1


def main():
    parser = argparse.ArgumentParser(
        description="READ-ONLY helper for the lighting plan 3c city tier ledger."
    )
    sub = parser.add_subparsers(dest="command", required=True)

    p_candidates = sub.add_parser(
        "candidates", help="List city/fort rooms in the named zones."
    )
    p_candidates.add_argument("zones", nargs="+", help="Zone folder name(s).")

    sub.add_parser("ruins-scan", help="Scan the whole world for ruin candidates.")

    p_check = sub.add_parser("check", help="Validate a ledger markdown file.")
    p_check.add_argument("ledger", help="Path to the ledger markdown file.")

    args = parser.parse_args()

    if args.command == "candidates":
        cmd_candidates(args.zones)
        sys.exit(0)  # per-file parse errors go to stderr but are not fatal
    elif args.command == "ruins-scan":
        cmd_ruins_scan()
        sys.exit(0)
    elif args.command == "check":
        sys.exit(cmd_check(args.ledger))


if __name__ == "__main__":
    main()
