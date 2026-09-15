# Condition YAML schema

> In Go these records are conditions (`internal/conditions`, `ConditionSpec`).

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/conditions/{conditionid}-{ConvertForFilename(name)}.yaml
```

- `{ConvertForFilename(name)}` — lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore.

**Worked examples:**
- conditionid: `2`, name: `"Stunned"` → `2-stunned.yaml`
- conditionid: `3`, name: `"Blinded"` → `3-blinded.yaml`
- conditionid: `0`, name: `"Meditating"` → `0-meditating.yaml`

**Existing conditions** (for reference IDs):
`0-meditating`, `1-illumination`, `2-stunned`, `3-blinded`, `5-minor_potion_healing`, `6-stamina_draught`, `7-conviction_draught`, `24-death_recovery`

---

## 2. Field Reference

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `conditionid` | int | **yes** | Unique integer. Must match filename. |
| `name` | string | **yes** | Display name. Must match filename via `ConvertForFilename`. |
| `description` | string | **yes** | Shown to the player when they have this condition (in status). |
| `secret` | bool | no | If true, the condition is hidden from the player's status display. Default false. |
| `triggernow` | bool | no | If true, the trigger fires immediately on application. Default false. |
| `triggerrate` | string | no | How often the trigger fires. See triggerrate formats below. |
| `triggercount` | int | no | How many times the trigger fires before the condition expires. 0 = permanent until removed. |
| `statmods` | map | no | Stat modifiers applied while the condition is active. |
| `flags` | list | no | Behavior flags. See valid flags below. |
| `tick_pool` | string | no | `"health"`, `"stamina"`, or `"conviction"`. Enables auto-tick. |
| `tick_percent` | float | no | Base % of max pool per tick. Positive=heal, negative=damage. |
| `tick_variance` | float | no | Random variance added to percentage (for DoTs). |
| `tick_min` | int | no | Minimum absolute tick amount. Default 1. |
| `start_remove_conditions` | list | no | Condition IDs removed when this condition starts (cure effects). |
| `start_user_text` | string | no | Text sent to holder when the condition starts. Supports `{source}` token. |
| `start_room_text` | string | no | Text sent to room when the condition starts. Supports `{source}` token. |
| `trigger_user_text` | string | no | Text sent to holder each trigger tick. |
| `trigger_room_text` | string | no | Text sent to room each trigger tick. |
| `end_user_text` | string | no | Text sent to holder when the condition expires. |
| `end_room_text` | string | no | Text sent to room when the condition expires. |

### triggerrate Formats

```yaml
triggerrate: "1 round"          # Each combat/game round
triggerrate: "3 rounds"
triggerrate: "5 real minutes"   # Real-world time
triggerrate: "2 irl hours"      # Alias for real hours
triggerrate: "1 game day"       # In-game time
```

### statmods Sub-fields

```yaml
statmods:
  strength: 10          # Positive or negative integer
  perception: -40       # Applied while the condition is active; removed on expiry
  vitality: 5
  dexterity: -5
  willpower: 15
  charisma: -10
```

### Valid Flags

| Flag | Effect |
|------|--------|
| `no-combat` | Prevents the affected character from engaging in combat. |
| `no-go` | Prevents the affected character from moving between rooms. |
| `cancel-on-combat` | Condition is automatically removed when combat begins. |
| `cancel-on-action` | Condition is automatically removed when any action is taken. |
| `hidden` | Condition is not visible to other players examining the character. |
| `secret` | Condition hidden from the character's own status (same as `secret: true`). |
| `lightsource` | Character acts as a light source while this condition is active. |
| `see-nouns` | Character can perceive hidden nouns in rooms. |
| `nightvision` | Character can see in darkness while this condition is active. |
| `poison` | Marks the condition as a poison. Damage per tick comes from `tick_pool`. |
| `drunk` | Applies intoxication effects. |

Multiple flags can be combined:
```yaml
flags:
  - cancel-on-combat
  - cancel-on-action
```

---

## 3. Annotated Examples

**Simple duration condition with statmod:**
```yaml
# _datafiles/world/dogmud/conditions/3-blinded.yaml
conditionid: 3                 # Must match filename (3-blinded.yaml)
name: Blinded
description: Your vision is obscured, hindering your senses.
triggerrate: 1 round           # Ticks every round
triggercount: 3                # Expires after 3 ticks (3 rounds)
statmods:
  perception: -40              # Heavy perception penalty while blinded
```

**Behavior-controlling condition:**
```yaml
# _datafiles/world/dogmud/conditions/0-meditating.yaml
# condition 0 is special: natural expiry removes the player from the game
conditionid: 0
name: Meditating
description: You are meditating before leaving the realm.
triggerrate: 1 round
triggercount: 5                # 5-round countdown
flags:
  - cancel-on-action           # Any action interrupts meditation
  - cancel-on-combat           # Combat interrupts meditation
```

**Utility condition (light source, no expiry):**
```yaml
# _datafiles/world/dogmud/conditions/1-illumination.yaml
conditionid: 1
name: Illumination
description: A soft glow surrounds you, lighting the way.
triggercount: 0                # 0 = permanent until removed
flags:
  - lightsource
```

**Worn-item condition (referenced from item's wornconditionids):**
```yaml
conditionid: 15
name: Ring of Swiftness
description: Your movements feel unusually fluid.
triggercount: 0                # Permanent while worn
statmods:
  dexterity: 8
```

---

## 4. YAML Text Fields

Condition messaging lives in YAML. The engine sends the text automatically
when the condition starts, triggers and ends. Use `{source}` for the
condition holder's name.

**Flavor-only condition:**
```yaml
conditionid: 27
name: Iron Will
description: Your mind is fortified against intrusion.
triggerrate: 1 round
triggercount: 12
statmods:
  willpower: 10
start_user_text: "Your mind hardens like iron, walling off intrusion."
end_user_text: "The iron resolve softens, leaving your thoughts exposed."
```

**Healing condition using tick_pool:**
```yaml
conditionid: 32
name: Vital Surge
description: Chrysalis energy steadily mends your body over time.
triggerrate: 2 rounds
triggercount: 9
tick_pool: health
tick_percent: 0.05
start_user_text: "Chrysalis energy suffuses your body with a warm, mending pulse."
end_user_text: "The vital surge fades."
```

**Poison DoT using tick_pool with variance:**
```yaml
conditionid: 39
name: Venom
description: Poison courses through your veins.
triggerrate: 1 round
triggercount: 5
triggernow: true
flags:
  - poison
tick_pool: health
tick_percent: -0.08
tick_variance: 0.04
tick_min: 3
start_user_text: "Venom seeps into your blood, burning from within."
start_room_text: "{source} winces as venom takes hold."
end_user_text: "The venom finally runs its course."
```

**Cure condition (removes poison on start, then heals over time):**
```yaml
conditionid: 47
name: Minor Antidote
triggerrate: 1 round
triggercount: 6
tick_pool: health
tick_percent: 0.05
start_remove_conditions:
  - 39
  - 40
start_user_text: "The antidote burns through your veins, purging toxins."
end_user_text: "The antidote fades from your system."
```

**Stat scaling note:** When a condition with `tick_pool` is applied by a spell,
the tick amount scales with the caster's spellcasting skill and weapon spell
multiplier. When applied by a potion or other non-spell source, no stat
scaling is applied.

---

## 5. Gotchas

**`{conditionid}-{ConvertForFilename(name)}.yaml` — both parts required.**
Unlike spells, condition filenames MUST include both the ID and the converted name. `3.yaml` won't be found; it must be `3-blinded.yaml`.

**`triggercount: 0` means permanent.**
A condition with `triggercount: 0` never expires from tick-down. It must be explicitly removed by a spell, item, or other game effect.

**`triggernow: true` fires the trigger on application.**
If the condition does damage per tick (e.g. poison), setting `triggernow: true` means it also fires immediately when applied. Usually desirable for harmful conditions.

**statmods are active only while the condition is applied.**
When the condition expires, all statmods are automatically reversed.

**Condition IDs must be globally unique.**
Scan the existing conditions folder before assigning a new ID. The engine indexes conditions by ID — a collision will cause one condition to overwrite the other at load time.

**condition 0 is reserved.**
conditionid 0 (`Meditating`) has special engine behavior: when it expires naturally, it removes the player from the game gracefully. Do not use conditionid 0 for any other purpose.
