# Lighting plan 3b: the biome vocabulary

Plan 3b of the graded lighting arc. Written 2026-09-23 against master
`d8562aa5b`, immediately after 3a merged as PR #162.

3a built the mechanism: a room's light is the sky attenuated by how much of it
reaches that floor, plus the place's lamp, plus carried lights. **3b gives the
world a vocabulary honest enough to use it.**

Amends nothing. The arc spec and its celestial amendment both stand.

---

## Facts verified against source

Read from the tree on 2026-09-23. Rows marked 🔴 corrected an assumption made
earlier in the same design session.

| # | Fact | Evidence |
|---|------|----------|
| 1 | 117 of 1386 rooms resolve to the synthetic Go `default` biome, and the breakdown accounts for all 117 exactly: **81 `a_dark_forest`, 21 `endless_trashheap`, 6 `newcomer_antechamber`, 5 `instance_arena`, 3 `instance_planar_oasis`, 1 `shadow_realm`** | parsed from `testdata/lighting_daycycle.golden` against every room file's `roomid` |
| 1b | 🔴 **`shadow_realm` has TWO room files but only ONE shipped room.** `75.yaml` is the Waiting Room; `-1.yaml` is The Void at **roomid -1**, which never appears in the golden, so it is outside the shipped room set and outside `GetAllRoomIds` | golden scan, room files |
| 2 | 🔴 **`a_dark_forest` is dead content.** 81 rooms in 82 files (the 82nd is `zone-config.yaml`), ids 1002-1082, every description the stub *"A room in A Dark Forest. This area has not yet been described."* **Nothing outside the zone exits into it**, it has **zero mob spawns**, and no quest, script or content file references it | room files, inbound-exit scan, `spawninfo` grep |
| 3 | 🔴 **It is DOGMud's own zone, not upstream's.** `_datafiles/world/default/rooms/` has no copy, so deleting it cannot conflict on an upstream merge. It has been swept along by bulk migrations (0.15.0 coordinates) but never authored | directory listing, `git log --follow` |
| 4 | `StartRoom` is **5200**, not in `a_dark_forest` | `_datafiles/config.yaml:2187` |
| 5 | 🔴 **The `plains` and `river` declarations are DEAD CONFIG, not a content request.** Every room in all five declaring zones already carries an explicit biome, so nothing ever falls through to the zone default | per-zone biome histogram |
| 6 | `dustwalk_road` declares `plains`, uses 10 × `land`. `marches_spur_road` declares `plains`, uses farmland 9 / land 8 / cave 1. `stillwater` declares `plains`, uses city 29 / land 7 / water 7 / cave 6 / swamp 1. `watchers_crossing` declares `river`, uses water 7 / land 1 | same |
| 7 | `newcomer_antechamber`'s 6 rooms carry an **explicit `biome: default`**, so they are orphans by choice rather than omission | room files |
| 8 | 🔑 **Those 6 rooms are character creation**: The Threshold, Knowing Yourself, What You Carry, The World Speaks, The Proving, The Landing | room titles |
| 9 | 🔑 **`look.go` refuses a room description below `LightBlindBelow`**, so a dark onboarding room is an unreadable one | `internal/usercommands/look.go` |
| 10 | 🔑 **`instance_planar_oasis` has its own sky and says so**: *"a sky that is simultaneously dawn and dusk"*, *"The sky above has settled into a permanent twilight"*, and *"Shapes move in the heat haze, some are mirages, some are not"* | room descriptions |
| 11 | 🔴 **The Foldweave is a spider lair biomed as `cave`**: The Foldthreshold, Web-Choked Gallery, The Egg Vault, The Web-Hung Deep, The Foldweaver's Court, 12 rooms | room titles, `foldweave-glow` grep |
| 12 | 🔑 **`spiderweb` is a shipped biome with ZERO rooms**, whose own description says *"naturally quite dark due to the thick coating of web everywhere"*, and whose weather classification comment says it is filed apart from stone because *"its darkness is webbing, not stone"* | `biomes/spiderweb.yaml`, `modules/weather/content/emotes.go:239-241` |
| 13 | 🔴 **The Crash Site Interior is a spacecraft biomed as `cave`**: Medical Bay, Fabrication Bay, Command Deck, Records Archive, Signal Array, Sealed Shuttle Bay, Airlock, Navigation Alcove, 31 rooms | room titles |
| 14 | Those 31 plus the Foldweave's 12 are **the 43 rooms held lit by a static `lightmod: 2`** (`hull_suppression` and `foldweave-glow`), and 3a's bridge puts them at exactly **58** | `testdata/lighting_daycycle.golden`, mutator YAML |
| 15 | 🔴 **`new_plymouth_sewers` is 20 rooms of `biome: city`**, which carries no `indoor` flag, so outdoor-only weather mutators render underground. Room 6403 is *"A brick barrel-vault… grey light falls in a single column through the pried drain-cap above"* | room files, `biomes/city.yaml`, `mutators.go` |
| 16 | `new_plymouth_temple` is 25 `city` rooms split between genuine outdoor (Temple Gate Plaza, Temple Courtyard, Censer Court, Garden of Repose, Eastern Processional, Cloister Walk, Sexton's Walk) and genuine interior (Grand Temple Sanctuary, High Altar, Keeper's House, Warden's Cell, Seminary, Archive Stacks, Chapel) | room titles |
| 17 | The 61 `forest` rooms already make an open/dense distinction in their titles that the biome cannot express. Unambiguously dense: Deep Woods, Hidden Grove, Overgrown Hollow, The Deep Timber, Under the Old Trees, Old Stand. 🔴 **CORRECTED: Grove Heart and Whispering Grove were on this list and should not have been.** Their prose reads OPEN, "the trees thin and a clearing opens" and "the trees stop at the edge"; the list had pattern-matched the word Grove in their titles, which is the exact error the plan warns against. A grove's heart is the clearing at its centre. Unambiguously open: Forest Path, The Forest Edge, and every Camp, Track, Road, Crossing, Meadow, Clearing and Glade | room titles by zone |
| 18 | Three arena-ish zones exist and **none is inside the ship**: `instance_arena` (5 rooms, no biome), `test_arena` (3 rooms, `fort`), and a single `shore` Sparring Circle in `pothole_coulee` | title grep |
| 19 | Every biome carries a matching file in `_datafiles/world/dogmud/weather/climate/`, including `house.yaml` and `spiderweb.yaml` | directory listing |
| 20 | `undergroundBiomes` is `{cave, dungeon}`; `surfaceIndoorBiomes` is `{house, fort, spiderweb}`. Classification must be TOTAL and `biome_coupling_test.go` fails the build otherwise | `modules/weather/content/emotes.go:227-247` |
| 21 | 🪤 **Go templates reach methods by REFLECTION**, invisible to `go build`, and `internal/templates/templatesfunctions.go` shadows `lt`/`lte`/`gte` as **int-only**. Both broke `biome.template` during 3a | 3a's record |

---

## What is wrong today

Five things, all of them data rather than mechanism.

1. **117 rooms have no biome at all** and take a synthetic Go fallback that is
   open-sky and unlit. Two thirds of them are a zone nobody can reach.
2. **Places are labelled as what they resemble, not what they are.** A
   spacecraft command deck is a cave. A spider lair is a cave. A sewer is a
   city street, which is why weather renders underground.
3. **`spiderweb` has no rooms** and `the_foldweave` has no honest biome, and
   they are the same gap seen from both ends.
4. **The forest cannot express its own prose.** *Under the Old Trees* and
   *Herb Clearing* carry identical light.
5. **Four zone-configs name biomes that do not exist**, which is inert today
   and a trap the moment anyone adds a room without an explicit biome.

---

## The design

### Six new biomes, one deleted

| Biome | `skylight` | `lamp` | Why |
|---|---|---|---|
| `sewer` | 0.0 | — | Underground, built, wet. Nothing in the existing set means that, and the weather prose class turns on it |
| `interior` | 0.15 | 50 | A built indoor space. **Absorbs `house`**, which is deleted |
| `dense_forest` | **0.25** | — | Canopy thick enough to matter. See the calibration below |
| `plains` | 1.0 | — | Open grassland, distinct from `land`'s generic fallback |
| `river` | 1.0 | — | Flowing water, distinct from `water`'s still |
| `ether` | 0.0 | 60 | Outside the world entirely. Time-invariant by construction |

🔑 **`plains` and `river` are world-building, not lighting.** Both read
identically to `land` and `water` under the model. They are in this plan
because 3b is where the biome vocabulary is settled and fact 5 shows four
zone-configs already reaching for them, not because they change any light.
Say so plainly in the commit rather than implying a lighting benefit.

### `dense_forest` at 0.25, and why the number

🔑 **`forest` at 0.45 is already blind at night**, so a darker forest cannot
differentiate after dark. The distinction is entirely a daytime one: when do
the deep woods stop being readable at midday.

| | midwinter noon | equinox noon | midsummer noon |
|---|---|---|---|
| `forest` 0.45 | 54 full | 61 full | 65 full |
| **`dense_forest` 0.25** | **47 shapes** | 54 full | 58 full |

So the deep timber hides faces at midday in winter, and reads clear the rest
of the year. 🅾️ Owner's calibration, chosen between a safer 0.30 and a bolder
0.15.

### Where every orphan goes

| Rooms | From | To | Note |
|---|---|---|---|
| 81 | `a_dark_forest` | 🅾️ **DELETED** | Fact 2 and 3: unreachable, unpopulated, stub text, not upstream's |
| 21 | `endless_trashheap` | `land` | Open wasteland, 21 rooms all titled The Wasteland |
| 6 | `newcomer_antechamber` | `ether` | Fact 8 and 9 |
| 3 | `instance_planar_oasis` | `ether` + room `lamp: 38` | Fact 10 |
| 5 | `instance_arena` | `interior` | 🅾️ Owner: an arena is an interior place |
| 1 (+1 unshipped) | `shadow_realm` | `ether` | The Waiting Room. **`-1.yaml`, The Void, is biomed for consistency but is not a shipped room** (fact 1b), so it moves no golden and must not be expected to |

### Rooms that move because they were mislabelled

| Rooms | From | To |
|---|---|---|
| 20 | `city` (New Plymouth sewers) | `sewer` |
| 31 | `cave` (Crash Site Interior) | `interior` |
| 12 | `cave` (the Foldweave) | `spiderweb` |
| 15 | `house` | `interior` |
| ~13 | `city` (temple interiors) | `interior` |
| ~12 | `forest` | `dense_forest` |
| 25 | `land` | `plains` |
| 7 | `water` | `river` |

⚠️ **The forest and temple splits are judged by reading each description, not
by pattern-matching the title.** Fact 17 gives the unambiguous anchors at both
ends; the middle is a reading. A room whose intent is unclear stays where it is
and is reported, not guessed.

### The `lightmod: 2` bridge loses all 43 of its rooms

🅾️ Owner ruling. The Crash Site's *"cold blue-white glow that does not flicker
and casts no shadow you can find the source of"* becomes `interior`'s lamp, and
the Foldweave's becomes `spiderweb`'s. The `hull_suppression` and
`foldweave-glow` mutators stop carrying `lightmod`.

🔑 **This is honest rather than merely tidier.** Those rooms are lit because of
what they are, not because a mutator is parked on them, and a biome lamp says
so. It also hands plan 4 a `LightMod` retirement with no load-bearing consumers
left.

⚠️ Both goldens move here. The 43 rooms leave 58 for whatever their new biome
lamp gives them, and the shape must be proven room-for-room before re-recording,
as every golden move in this arc has been.

---

## The onboarding constraint

🔴 **The antechamber must never be dark, and this is not a preference.** Facts
8 and 9 together: a new player reads their character creation in those six
rooms, and `look.go` refuses to print a room description below the blind
threshold. If the antechamber followed the sun, a share of every day's new
players would arrive unable to read their own introduction, and nothing in the
test suite would say so.

`ether` at `skylight: 0.0` and `lamp: 60` makes it time-invariant by
construction rather than by luck. The same reasoning covers `shadow_realm`.

🔑 **The planar oasis is the interesting inverse.** It is not lightless, it has
a sky that is not ours, and the model can say that exactly: no sky term, and a
lamp at 38 that never changes. That lands it in the shapes band, where names
hide, and the room text already reads *"Shapes move in the heat haze, some are
mirages, some are not."* The prose chose the number.

---

## Consequences accepted

- **81 rooms are deleted.** A future builder loses a pre-wired grid. Accepted:
  it is unreachable stub content and the room ids return to the pool.
- **`house` ceases to exist** as a biome id. Anything referring to it by name
  breaks, which is the point of deleting rather than aliasing.
- **Two new biomes change no light.** `plains` and `river` earn their place on
  world-building grounds only.
- **The temple and forest splits are judgement calls** and a later reading may
  move a room or two. That is content drift, not a defect.

---

## Out of scope

- **The city main-street versus back-lane pass.** Every `city` room still
  shares one lamp of 35, so all town streets hide faces. **Plan 3c.**
- 🔴 **Roughly 50 more mis-biomed city interiors, measured during 3b.** Task 5
  moved the temple's 18 and `new_plymouth_crafting`'s 15, then scanned the
  remaining `city` zones and found about 50 further candidates across eight of
  them: offices, chapels, cells, halls, workshops and lofts in `greenford`,
  `new_plymouth_docks`, `new_plymouth_merchant`, `new_plymouth_noble`,
  `new_plymouth_old_quarter`, `stillwater`, `the_confluence` and
  `thornwall_city`. `the_confluence` alone holds 128 city rooms.

  🔑 **It declined to do them, and was right to.** Judging 50 rooms with the
  care the temple got is a task of its own scale, not the tail of another one,
  and a guessed interior is worse than a known-wrong city. **Plan 3c takes
  them**, alongside the main-street split, since both are readings of the same
  477 rooms.
- **`fort`'s split** between its open training yard and its buried vault, which
  needs room-level overrides. **Plan 3c.**
- **Transition notices** when light changes around or beneath a player.
  **Plan 3d.**
- **Retiring `LightMod`** itself. 3b empties it of load-bearing consumers;
  plan 4 deletes the vocabulary.
- **`test_arena`.** A test fixture zone on `fort`. Left alone deliberately.

---

## Traps this plan must respect

🪤 **Fact 21.** `biome.template` reads biome methods by reflection and the
template package shadows `lt`/`lte`/`gte` as int-only. Deleting `house` or
adding a biome accessor can break the `biome` command at runtime with a green
build and a green suite. **Render the template in a test.**

🪤 **Fact 20.** Every `indoor: true` biome must appear in exactly one of
`undergroundBiomes` or `surfaceIndoorBiomes`, or `biome_coupling_test.go` fails
the build. `sewer` belongs underground. `interior` belongs to surface indoor.
`spiderweb` is already classified and must stay where it is.

🪤 **Fact 19.** Every biome has a weather climate file. Adding six biomes and
deleting one means seven climate files, and `modules/weather/sim/climate.go`
binds archetypes by biome name in Go.

🪤 **`gametime`'s `roundDateCache` is keyed on the round with no config
fingerprint.** Any test that changes lighting config and reuses another test's
round silently gets that test's answer. `ClearDateCacheForTest` exists.
