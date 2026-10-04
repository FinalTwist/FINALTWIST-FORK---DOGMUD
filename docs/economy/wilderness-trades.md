# Wilderness trades

Hunting, butchery, lumberjacking and forage as real ways to play: go out,
hunt or chop, bring the materials home, then sell them to a merchant or
craft them yourself. Results depend mostly on the player's stats and the
quality of their tool; an existing skill gives a modest edge. Carpentry is
the only new skill.

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Data fixes: species salvage fallback, forage biome fill, iron ore, dead-end recipes, trophy tags | shipped 2026-10-03 |
| 1 | Foundations: item grades, tools, the gather roll, the carpentry skill, harvest tables, new stations, tool recipes | shipped 2026-10-03 |
| 2 | Hunting: `skin`, `butcher`, `harvest`; corpse states; species tables and mob overrides; animal materials; spoilage; furrier and butcher merchants; the scarcity pricing fix | next |
| 3 | Processing: scrape, cure, tan, dress; cord, glue, tallow, smoked meat; bone and horn carving; leather and fur garments; crafted grades | planned |
| 4 | Lumberjacking: timber tables, `survey trees`, `chop`, logs, grove depletion and regrowth, lumber yard | planned |
| 5 | Carpentry crafts: sawing, bows, arrows and bolts, staves, wooden shields, furniture for housing, bowyer | planned |
| 6 | Forage revamp: categories, survey, seasons, room richness, tool-driven finds | planned |
| 7 | Depth: `hunt` spawning, bundling, felling hazards, wanted species, bounties, caravans | planned |

## What phase 0 changed

- `internal/crafting/corpse_salvage.go`: `LookupCorpseSalvageFor` and
  `LookupCorpseSalvageForMob` fall back to the species when no group
  matches. Warm-blooded species (canine, bear, boar, deer, feline, mustelid,
  horse, rodent, bird, raptor) are listed; insects, arachnids, reptiles and
  fish wait for phase 2. Every caller (player salvage, the mob and resolver
  path, the companion's butcher pastime) uses the species-aware lookup.
- `internal/forager/forage_core.go`: `plains`, `dense_forest` and `river`
  get difficulties and yields from existing items; caves and mountains give
  iron ore (40236) instead of the iron ingot.
- New items and recipes: iron ore (40236) and `smelt-iron-ore`; the
  Predator-Pelt Mantle (20102, tailoring, consumes `cascade-hide`); the
  Skitter-Shell Bracer (20103, jewelcrafting, consumes `shrimp-chitin`).
- The spore sac (40008) and Old White's fang (40179) carry component tags.
  The leviathan tooth (40054) does not: it is bounty proof, not a material.

## What phase 1 added

- **Grades** (`internal/items/quality.go`): `Item.Quality`, crude to
  pristine, on the instance. Shown after the name except for standard;
  separates stacks; scales the sell price (`shops.GradedValue`, Balance
  `QualityValue*`). Ungraded is not a grade and prices at 1.0.
- **Tools** (`internal/items/tools.go`): `ItemSpec.Tool` with a type (knife,
  cleaver, bone saw, axe, saw, scraper, sickle, carving knife, trowel), a tier
  (crude, iron, steel, masterwork) and a speed. One-handed blades stand in
  as crude knives, and cleaving weapons as crude cleavers and axes. The tier
  caps the best grade: crude standard, iron fine, steel superb, masterwork
  pristine. A crafted tool's own grade nudges its tier by one.
- **The gather roll** (`internal/gather`): `score = avg(two job stats) x
  tool multiplier + skill x GatherSkillWeight`, against
  `GatherBaseDifficulty + target tier`, through the salvage contest and its
  mercy floor, paying the sight ramp. Grade comes from the margin in roll
  standard deviations. Knobs are in `config.yaml` under GATHERING.
- **Carpentry** is wired through every skill list, the progression tables,
  the chrysifier drift, the homunculus, companions, vendor categories and
  help. Stations `woodworking_bench` and `tanning_rack` exist in nine rooms.
- **Harvest tables** (`internal/species/harvest.go`): `harvest:` on species
  and mobs (skin and butcher sections; item tag, quantity for a medium body,
  tool, rare flag). `mobs.ResolveHarvest` merges a mob's table over its
  species per section. Both are validated at boot. No data is authored yet;
  phase 2 authors the tables and the commands that read them.
