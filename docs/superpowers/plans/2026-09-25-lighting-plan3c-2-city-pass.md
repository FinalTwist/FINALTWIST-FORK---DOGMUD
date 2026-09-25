# Lighting plan 3c-2: the city pass, second half, and `city` deleted

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sort the remaining 268 `city` rooms into the tiers 3c-1 built, then delete `city` from the `dogmud` world so nothing can fall back onto it.

**Architecture:** The same machinery as 3c-1 (`tools/city_tier_ledger.py`, a ledger, an owner checkpoint, one-line `biome:` edits), then a deletion enumerated by guards: `biome_coupling_test.go` names every emote key still pointing at `city`, and a new guard names every room or zone that does. Classification happens once, with one owner checkpoint; shipping is TWO PRs because one would be about 299 files, on CI's 300-file lint inversion.

**Tech Stack:** Go tests, YAML content, the read-only ledger tool.

**Spec:** `docs/superpowers/specs/2026-09-25-lighting-plan3c-city-pass-design.md` (3c-2 is its second slice). **Prior art:** `docs/superpowers/plans/2026-09-25-lighting-plan3c-1-city-pass.md` and its ledger `docs/superpowers/audits/2026-09-25-lighting-plan3c-ledger.md`, whose "Owner answers" section is binding here.

**Every commit** ends with `Co-Authored-By: <your model> <noreply@anthropic.com>`.

---

## Facts verified against source (master `2b4e59024`, 2026-09-25)

| # | Fact | Source |
|---|---|---|
| 1 | 268 rooms still `biome: city`: `the_confluence` 128, `greenford` 42, `thornwall_city` 32, `stillwater` 29, `hartcharn` 26, `kilnreach_works` 9, `pothole_coulee` 2 | grep |
| 2 | 12 zone-configs set `defaultbiome: city` (all 8 New Plymouth zones plus `greenford`, `hartcharn`, `kilnreach_works`, `thornwall_city`) | grep |
| 3 | `city` survives in `biomes/city.yaml`, `weather/climate/city.yaml`, and as the `city:` key (anchored `&city_*`) in six emote files | 3c-1 |
| 4 | `shipped_climate_test.go` lists `city` as required (line 26) and skips its tier-parity check when `city` has no profile (line 50) | the test |
| 5 | `city_tier_emotes_test.go` passes with or without a `city:` key, as long as both tier keys exist and match | the test |
| 6 | `biome_coupling_test.go` fails on any emote key that names no biome | the test |
| 7 | Three goldens move whenever rooms change biome: `testdata/lighting_daycycle.golden`, `testdata/lighting_parity.golden` (root package), and `internal/narration/testdata/stores/weather_emotes.golden`. Each requires a room-for-room accounting before `-update` | 3c-1 commit `83541f914` |
| 8 | Every player starts with `chrysalis-glow` (`internal/characters/character.go:367`), so a dark room is playable | source |
| 9 | Deferred from 3c-1's ruin scan, with that group's lean: 4144 Old Chapel Ruin (stillwater, a clear ruin), 6219 Inkwell Court (worth a second look), 6297 The University Stair, 6129 The Weighhouse Yard, 6185 The Cloister Garth, 6197 The Kitchen Court, 6229 The Quiet Garden (open courts by design, not ruins) | 3c-1 ledger |

## Traps. Read before Task 1.

1. **3c-1's traps all apply** (plan 3c-1 lines 21-62): LF storage and `git diff --numstat`, named-path `git add`, `grep -c` exit 1, `go test ./...`, boot from `HEAD`.
2. **Run the FULL suite after any data change,** not one package. 3c-1 Task 4 ran only `./modules/weather/...` and missed a golden in `internal/narration`.
3. **Goldens are re-recorded only with a room-for-room accounting** (fact 7). A move you cannot explain is a defect.
4. **Two PRs, each under 300 files.** Count before pushing.

---

## Rubric (3c-1's, plus the owner's 3c-1 rulings)

| Class | It is this when |
|---|---|
| `city_thoroughfare` | A named main way through the city, a square or plaza, an open market, a city or district gate, a bridge, a quay a crowd would use |
| `city_backstreet` | Any outdoor city space that is not a main way: side streets, lanes, alleys, rows, courts, yards, steps, slums. Also the default when unsure |
| `interior` | Inside a roofed building: shop, office, tavern, bathhouse, hall, chapel, cell, loft, rooms above a shop |
| `ruins` | Built, and its roof is GONE, so the sky shows |
| `dungeon` | (owner ruling) Built and underground or sealed with no sky: cellars below cellars, buried streets, crypts. Add `lamp: 38` (reason prefix `LAMP 38:`) only when the room's OWN text names a burning lamp; otherwise dark (`DARK:`) |
| `sewer` | (owner ruling) Drains, underdocks, bilge spaces; same lamp rule |
| `road` / `river` / other open biome | (owner ruling) Unpaved tracks, fords and open country outside the walls. A room at the wall may get `lamp: 38` as a security lamp (`LAMP 38:`) |

Rules: read the whole room file (description, exits, nouns), never the title alone; a shop front on the street is the street's class; abandoned is not ruined; a roofed gate passage is `interior`, its open approach `city_thoroughfare`; unsure means `city_backstreet` with `UNSURE:`. A thoroughfare should connect to other thoroughfares; the checker flags islands for a second read.

---

### Task 1: Classify all six zones into the ledger

**Files:** Create `docs/superpowers/audits/2026-09-25-lighting-plan3c2-ledger.md` (same format as 3c-1's).

Dispatch read-only classifiers in parallel, each returning its sections as text:

| Group | Zones | Rooms |
|---|---|---|
| E | `the_confluence` rooms 1-64 by id order | 64 |
| F | `the_confluence` rooms 65-128 | 64 |
| G | `greenford` + `kilnreach_works` | 51 |
| H | `thornwall_city` + `stillwater` + `hartcharn` + `pothole_coulee`'s 2 | 89 |

Each gets the rubric and rules above verbatim, `python tools/city_tier_ledger.py candidates <zones>`, and the fact-9 deferred rows that fall in its zones. `the_confluence` is split by room id so both halves see the whole zone's exits: give each half the full candidate list and assign rows by id.

- [ ] **Step 1:** dispatch E to H.
- [ ] **Step 2:** assemble; run `python tools/city_tier_ledger.py check <ledger>`; second-read every flagged island and record the outcome in its row.
- [ ] **Step 3:** close the ledger with a summary table (per zone per class), the UNSURE list, every `LAMP 38` row, every move to a biome outside the rubric's first four classes with its movement-cost change, and numbered questions for anything outside the rubric.
- [ ] **Step 4:** commit the ledger.
- [ ] **Step 5: 🛑 OWNER CHECKPOINT.** Summary, UNSURE rows, lamp rows and questions inline. No room file changes before the owner answers; record the answers at the ledger's foot.

---

### Task 2 (PR 3c-2a): apply `the_confluence` and `greenford`

- [ ] **Step 1:** apply the ledger rows for these two zones exactly as 3c-1 Task 7 did: replace the exact `biome: city` line (confirm it appears once), insert `lamp: 38` directly after it for `LAMP 38:` rows (confirm no existing `lamp:`), `sed -i` is safe (LF storage).
- [ ] **Step 2:** per zone, `git diff --numstat` shows `1 1` per biome-only row and `2 1` per lamp row; `grep -l '^biome: city$'` in the zone prints nothing (standalone).
- [ ] **Step 3:** one commit per zone.
- [ ] **Step 4:** full suite. The three goldens (fact 7) will move: account room for room (every moved room is a changed ledger row in these two zones, each landing on the values of existing rooms of its new biome, nothing else moves), then re-record and commit them together with the accounting in the body.
- [ ] **Step 5:** patch note entry for these towns (name their lit ways; `dogmud-player-copy`), `docs/README.md` rows for this plan and the ledger.
- [ ] **Step 6:** pre-PR gate (gofmt, full suite, `golangci-lint --new-from-merge-base=origin/master`, boot check in `C:/tmp/dogmud-boot-check`, file count under 300), then PR, review, CI on the head commit, merge.

### Task 3 (PR 3c-2b): the other five zones, then delete `city`

Branch from master after 3c-2a merges.

- [ ] **Step 1: The guard, failing first.** Create `internal/rooms/no_city_biome_test.go`: walk `../../_datafiles/world/dogmud/rooms`, and fail naming every room file whose `biome:` is `city` and every `zone-config.yaml` whose `defaultbiome:` is `city`. Read files as text and match the exact lines `biome: city` / `defaultbiome: city` (anchored, so `city_backstreet` never matches). Run it: it must FAIL listing the remaining rooms and the 12 zone-configs.
- [ ] **Step 2:** apply the ledger rows for `thornwall_city`, `stillwater`, `hartcharn`, `kilnreach_works`, `pothole_coulee` (Task 2 steps 1-3).
- [ ] **Step 3:** change the 12 zone-configs' `defaultbiome: city` to `defaultbiome: city_backstreet`. The guard now PASSES; prove it can fail by setting one room back to `biome: city`, then restore.
- [ ] **Step 4: Emotes.** In each of the six files, move the anchor from `city:` to `city_thoroughfare:` (the pool's lines move under `city_thoroughfare: &city_<x>`), keep `city_backstreet: *city_<x>`, delete the `city:` key. `city_tier_emotes_test.go` must still pass.
- [ ] **Step 5: Climate.** Delete `_datafiles/world/dogmud/weather/climate/city.yaml`. In `shipped_climate_test.go`, remove `city` from the required list, and replace the skip-when-absent parity block with an always-on `reflect.DeepEqual(climate["city_thoroughfare"], climate["city_backstreet"])` plus a pin of one literal value the old `city` profile had (so the tiers cannot drift together unnoticed).
- [ ] **Step 6: Delete `_datafiles/world/dogmud/biomes/city.yaml`.** Run the full suite. `biome_coupling_test.go` must stay green (Step 4 removed every `city:` key); any other failure names a remaining consumer to fix, not a test to loosen. `modules/weather/sim/climate.go`'s built-in `"city"` profile is NOT touched: the upstream `default` world uses it.
- [ ] **Step 7:** goldens, accounted for as in Task 2 Step 4.
- [ ] **Step 8:** docs: `internal/rooms/context.md` (`city` is gone; the vocabulary is the tiers), patch note for these towns, `docs/README.md`, and mark the 3c spec's out-of-scope list done where it now is.
- [ ] **Step 9:** pre-PR gate, PR, review, CI on the head commit, merge.

## After 3c-2b

Plan 3d (transition notices) is next in the arc. The spec's movement-cost acceptances and the owner's lamp rule carry forward to it unchanged.
