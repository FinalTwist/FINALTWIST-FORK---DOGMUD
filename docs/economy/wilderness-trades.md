# Wilderness trades

Hunting, butchery, lumberjacking and forage as real ways to play: go out,
hunt or chop, bring the materials home, then sell them to a merchant or
craft them yourself. Results depend mostly on the player's stats and the
quality of their tool; an existing skill gives a modest edge. Woodwork is
the only new skill.

## Phases

| # | Phase | Status |
|---|---|---|
| 0 | Data fixes: species salvage fallback, forage biome fill, iron ore, dead-end recipes, trophy tags | shipped 2026-10-03 |
| 1 | Foundations: item grades, tools, the gather roll, the woodwork skill, harvest tables, new stations, tool recipes | shipped 2026-10-03 |
| 2 | Hunting: `skin`, `butcher`, `harvest`; corpse states; species tables and mob overrides; animal materials; spoilage; furrier and butcher merchants; the scarcity pricing fix | shipped 2026-10-03 |
| 3 | Processing: scrape, cure, tan, dress; cord, glue, tallow, smoked meat; bone and horn carving; leather and fur garments; crafted grades | shipped 2026-10-03 |
| 4 | Lumberjacking: timber tables, `survey trees`, `chop`, logs, grove depletion and regrowth, lumber yard | shipped 2026-10-04 |
| 5 | Woodwork crafts: sawing, bows, arrows and bolts, staves, wooden shields, furniture for housing, bowyer | shipped 2026-10-04 |
| 6 | Forage revamp: categories, survey, seasons, room richness, tool-driven finds | next |
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
- **Woodwork** is wired through every skill list, the progression tables,
  the chrysifier drift, the homunculus, companions, vendor categories and
  help. Stations `woodworking_bench` and `tanning_rack` exist in nine rooms.
- **Harvest tables** (`internal/species/harvest.go`): `harvest:` on species
  and mobs (skin and butcher sections; item tag, quantity for a medium body,
  tool, rare flag). `mobs.ResolveHarvest` merges a mob's table over its
  species per section. Both are validated at boot. No data is authored yet;
  phase 2 authors the tables and the commands that read them.

## What phase 2 added (hunting)

- **Commands** (`internal/usercommands/carcass.go`, `internal/actions/harvest.go`):
  `skin <corpse>`, `butcher <corpse>` (alias `carve`), `harvest <corpse>`
  (list what is left) and `harvest <part> from <corpse>` (one part, harder
  roll, one grade better). Each is a timed job of 2, 4 or 6 rounds by body
  size, shortened by the knife's speed, run on the Salvaging activity.
  Needs a knife (any one-handed blade is a crude one), sight, no combat and
  loot rights. `salvage <corpse>` points at them when the carcass has a table.
- **The roll**: `gather.Roll` with Dexterity and Perception (skin) or
  Strength and Dexterity (butcher), the knife, and a little Salvage skill,
  against `GatherBaseDifficulty - GatherCarcassEase + statpool *
  GatherStatPoolDifficulty + size`. A baseline gatherer with an iron knife
  wins about half the time on a steppe wolf.
- **Per entry**: a cleaver is needed for bone and fat, a bone saw for horn,
  antler, tusk, fangs and claws; each entry is capped by its own tool's tier.
  Rare parts (fangs, glands, organs) need a Perception roll. Size scales
  quantities; Strength adds meat. A carcass past `CorpseStaleGradeAt` of its
  decay gives one grade worse, past `CorpseMeatLostAt` no meat or organs.
  Butchering before skinning ruins the hide; a botched job spends the section.
- **Tables**: seventeen species carry a `harvest:` block. Mob overrides move
  hand-authored loot onto the carcass: the pack hounds' pack-hide, the
  Cascade Pass and Eastern Highlands predators' thick pelt, the Pass-Apex
  claw, the Blind Stalker and Pale Lurker heat-pit organ, venom sacs. The
  roe deer, hares and feral boar no longer carry meat or sinew as loot. The
  Pronghorn keeps its raw meat: quest 42 depends on it.
- **Materials**: 29 new raw goods (40300 to 40328): pelts, hides, fur,
  scales, chitin, bone, horn, antler, tusk, fang, claw, feathers, talons, fat,
  gut, glands, bear bile, roe, ichor, horsehair, venison, fowl. Deer, boar and
  horse hides share the `hide` tag; harvest entries name them by item id.
- **Spoilage** (`items/spoilage.go`, `hooks/spoilage.go`): `spoil_after` on
  raw meat, organs, fat, gut and raw hides; harvested goods are stamped and
  rot on the game clock; rotten goods are thrown out; merchants pay less as
  goods age and nothing for rotten ones.
- **Merchants**: tailoring merchants buy hides and furs, cooks buy meat,
  jewelers buy bone and horn, apothecaries glands. Corwin the Tanner (New
  Plymouth) now sells the iron skinning knife, the hide scraper, salt and
  bark liquor. The walk-in pricing fix stops a shop paying four times as much
  for the second pelt as the first.

## What phase 3 added (processing)

- **Recipe tools**: `tool:` on a recipe; the crafter must carry it. Craft
  lists show it and refuse without it.
- **Crafted grades** (`gather.CraftGrade`): an output is graded when an input
  is graded or the recipe has a tool; capped one above the worst input and by
  the tool.
- **Recipes** (23): cure-hide, tan-leather, cut-leather (leather strips are
  now craftable from hides); bowstring, horsehair-bowstring, rawhide-cord;
  wolf-fur-cloak, bear-fur-mantle, cat-pelt-hood, fine-fur-collar,
  fur-lined-boots, hide-jerkin, scale-vest (tailoring); chitin-bracers,
  carve-bone-needles, bone-arrowheads, fang-necklace (jewelcrafting);
  smoke-meat, venison-jerky, roast-fowl, render-tallow, bone-glue,
  bark-liquor (cooking).
- **Hide traits** are carried by the garments: wolf gives Perception, bear
  Vitality, cat Skullduggery, fine fur Charisma, scales Dexterity.
- **Processed goods** (40350 to 40357): cured hide, leather hide, bowstring,
  rawhide cord, bone glue, tallow, bark liquor, bone arrowheads. Foods
  30069 to 30071; garments 20104 to 20112.

## What phase 4 added (lumberjacking)

- **Data**: `_datafiles/world/dogmud/timber.yaml` (code `internal/timber`):
  fifteen species in three wood classes, each with a log item (40400 to
  40414), tier 1 to 4 and a survey note; pools for the forest, dense forest
  and swamp biomes, and zone pools for the Fernway, Fernway South (rare
  ironwood), Cascade Pass Road (rare yew), Pothole Coulee, Ashwick and
  Stillwater Marsh.
- **Commands**: `survey` / `survey trees` (what grows, what it is good for,
  how many trees are left, when a cut stand will be ready; tier 3 and 4 woods
  need a Perception and Search read to name) and `chop [tree]` (alias
  `fell`), a timed job of 4 to 7 rounds shortened by the axe's speed.
- **The roll**: `gather.JobChop`, Strength and Vitality times the axe, no
  skill, against `GatherBaseDifficulty - TimberEase + (tier - 1) *
  TimberTierDifficulty`. Logs: one, plus one per 50 Strength above 100, plus
  one for a fine felling, up to `TimberMaxLogs`; graded by the roll and capped
  by the axe. One or two branches, and bark (pine pitch, oak bark, birch bark)
  four times in ten. Logs weigh 8 to 14 and are not bag components: weight is
  the haul.
- **Stands**: each choppable room holds 6 to 10 trees in its long-term data,
  regrowing one per game day (`TimberRegrowRounds` 900), re-rolled toward the
  neighbours' species when regrown from stumps.
- **Merchants**: Camp-Foreman Bertt (Cascade Pass lumber camp) buys woodwork
  goods and sells planks and (since the review) crude axes and saws.

## What phase 5 added (woodwork crafts)

- **Lumber**: `saw-planks` (softwood log to 4 planks, saw), `saw-boards`
  (hardwood log to 2 hardwood boards, saw), `split-staves` (bow-wood log to 2
  bow staves, axe), `cut-shafts` (softwood log to 3 bundles of arrow shafts,
  saw), `whittle-shafts` (2 branches, carving knife, anywhere).
- **Bows**: `self-bow` (new 10057), `hunting-bow` (10041), `longbow` (new
  10058, Strength 110), `horn-bow` (new 10059, uses horn and sinew). The new
  bows sit on the U10d line and are listed in `postDetuneBows`.
- **Ammunition**: `fletch-arrows` (quiver 30062: shafts, feathers,
  arrowheads; anywhere) and `make-bolts` (case 30063). Arrowheads come from
  `iron-arrowheads` (blacksmithing, 40421) or phase 3's bone arrowheads.
- **Arms**: `quarterstaff` (new 10060), `wooden-shield` (20004), `kite-shield`
  (new 20113).
- **Furniture**: `wooden-chest`, `bed-frame`, `woodworking-bench` (40430
  to 40432, `furnishing:` chest, bed, workbench). `use` one in your own
  lodging to place it as a container, a bed or a woodworking bench, through
  the housing deed paths (`internal/housing/crafted.go`).
- **Bark**: `birch-bark-liquor` makes tanning liquor from birch bark.
- **Merchants**: Corwin Ashlade (Amber Valley woodworker) is the bowyer: he
  buys woodwork goods and sells shafts, bowstrings, quivers, self bows and
  (since the review) rough whittling blades. Both wood-trade keepers carry lanterns so their shops can
  trade at night.

## What the review added

- **Woodwork**: the skill once called carpentry is `woodwork` (tag, recipes
  directory, vendor category, help). `characters.validateSkillMigrations`
  folds a saved `carpentry` rank into `woodwork` on load; `reconcileShop`
  moves saved shops' craft support with their template.
- **Tool ladder**: every tool type (knife, cleaver, bone saw, axe, saw,
  scraper, sickle, carving knife) has a crude item (10061 to 10064, 40241 to
  40244) sold by merchants, and iron, steel (10065, 10066, 40245 to 40248,
  blacksmithing 15 to 22) and masterwork (10067 to 10070, 40249 to 40252,
  blacksmithing 45 to 55) items made only at a forge. Masterwork takes
  `crucible-steel` (40253, blacksmithing 40: steel, basalt-iron ore, coal
  dust) and an `ironwood-haft` (40254, woodwork 35 with a saw, from the
  ironwood log, now tagged `ironwood-log`). A recipe whose output is a tool is
  always graded (`gather.CraftGradeOutput`), so the smith's margin sets how
  well it works (`EffectiveToolTier`) and how long it lasts.
- **No resale**: `items.NeverResold` (iron and better tools). `sell.go`
  never shelves one; `reconcileShop` takes any off saved shelves;
  `TestToolLadderContent` refuses one in any shop list.
- **Wear**: `Item.Wear` against `ToolDurability` (Balance
  `ToolDurability*` 30 / 80 / 160 / 320 by tier, times grade 0.75 to 2.0).
  Skinning, butchering, harvesting, felling, foraging with a sickle and
  crafting with a recipe tool each add one; the tool breaks at its
  durability. Improvised weapons do not wear.
- **Rare access**: `HarvestEntry.min_tool` gates a part on the tool tier
  (checked after the rare roll; misses are reported as prizes the tool could
  not take). Trophy parts: prime wolf pelt (40255, canine, steel knife),
  trophy antlers (40256, deer, steel bone saw), trophy tusks (40258, boar,
  steel bone saw), great bear pelt (40257, bear, masterwork knife). Rare
  chances are multiplied by `RareToolMult*` (0.5 / 1.0 / 1.5 / 2.0).
  `timber.Species.MinAxe`: yew and walnut need iron, ironwood steel. A
  sickle multiplies forage score by its tier multiplier (never below 1) and
  draws extra candidates, keeping the dearest.
- **Field merchants** (`hunting` craft support and vendor category on raw
  animal goods and crude tools): Hunter Delk (Pothole Coulee, Hunter's
  Hollow), Trapper Maudry (new 9840, Stillwater travelers' camp 4142),
  Trapper Ottar (new 9841, Fernway eastern trailhead 4147). Woodcutter
  Hagen (North Road 4059) now trades in woodwork. Each carries a lantern for
  night trade. They buy at the ordinary walk-in prices.
- **Stations**: woodworking benches in Thornwall (469), the New Plymouth
  cooperage (5720) and the Confluence cooperage (6234); a forge in the New
  Plymouth forge yard (5709). `TestToolLadderContent` checks every recipe
  station exists in some room. The field crafts (cut leather, rawhide cord,
  whittle shafts, fletch arrows) stay station-free but need their tool.
- **Rot on the floor**: `rooms.removeSpoiledGoods` removes spoiled raw goods
  lying in a room or its stash each round tick.

## Hunter's gear (after the review)

Rare and trophy parts feed top-tier recipes, so a better tool pays off in
better gear and not only in coin. `TestToolLadderContent` checks that every
trophy part and the bone-saw rares (antler, tusk, claw, talons) are an
ingredient somewhere.

- Jewelcrafting (jeweler bench, carving knife): `claw-bracelet` (14, 20114),
  `talon-ring` (16, 20115), `antler-amulet` (18, 20116), `tusk-armlet` (24,
  20117); `tuskbound-torc` (42, 20118, trophy tusks), `antler-circlet` (45,
  20119, a head piece, trophy antlers), `hunt-kings-pendant` (55, 20120, both
  trophies).
- Blacksmithing (forge): `antler-hilt-hunting-sword` (28, 10071);
  `tusk-knuckle-gauntlets` (48, 20122), `crucible-plate-helm` (50, 20121,
  great bear pelt), `stag-hilted-crucible-blade` (52, 10072, trophy antlers).
- Tailoring (loom, scraper): `prime-wolf-cloak` (35, 20123) and
  `great-bearskin-cloak` (55, 20124).
- Values sit at roughly 1.4 times the materials' value, above the stat-implied
  value (`tools/item_value_audit.py` flags them as pinnacle pieces).

## Deferred

- The kill-damage penalty (fire, acid or overkill spoiling a hide) needs the
  killing blow recorded on the corpse; not done.
- NPC salvagers and the companion's butcher pastime still use corpse
  salvage, not skin and butcher.
- A shop resells bought goods ungraded (forged tools are no longer resold
  at all).
- Crafted grade raises value only; superb and pristine weapons and armour do
  not yet get a stat bump.
- No Thornwall bowyer or lumber merchant yet, though Thornwall now has a
  woodworking bench.
- Felling hazards and the `hunt` command are phase 7.
- Still unused: flight feathers, musk gland, silk gland and bear bile (no
  recipe), and the trowel tool (no item). Yew and black walnut share their
  tags with ash and oak, so a yew bow is no better than an ash one.
