package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Bonded companions (characters.CompanionBonded) are driven by the
// aicompanion module. They differ from every other companion kind in one way
// that matters here: their death is not the end of them. The record stays on
// the owner, the mind the module keeps for them survives, and they recover.
//
// What they lose on death is what any character loses: everything they were
// carrying, wearing and holding goes to the corpse through the normal mob
// loot path (dropMobLootAndSetCorpse), and the owner can recover it from
// there. Clearing the snapshot here is what keeps that from duplicating
// items: if the saved Items survived, the companion would respawn holding
// the same gear that is also lying in the corpse.

// bondedCompanionFell updates a bonded companion's record when its mob dies,
// instead of removing the record. Called from CompanionCleanup.
func bondedCompanionFell(user *users.UserRecord, comp *characters.CompanionInfo, instanceId int) {

	mob := mobs.GetInstance(instanceId)

	// Keep what was learned this session. saveCompanionState does this at
	// logout; a death mid-session would otherwise lose it.
	if mob != nil {
		snapshotCompanionProgression(comp, mob)
	}

	// Gear follows the loot rules. PermaGear suppresses the drop, so a
	// companion under it keeps everything; otherwise the corpse has it all.
	if mob != nil && mob.Character.HasConditionFlag(conditions.PermaGear) {
		if len(mob.Character.Items) > 0 {
			comp.Items = make([]items.Item, len(mob.Character.Items))
			copy(comp.Items, mob.Character.Items)
		} else {
			comp.Items = nil
		}
		comp.Equipment = mob.Character.Equipment
		comp.Gold = mob.Character.Gold
	} else {
		comp.Items = nil
		comp.Equipment = characters.Worn{}
		comp.Gold = 0
	}

	comp.InstanceId = 0
	user.Character.TrackCharmed(instanceId, false)
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	user.SendText(messaging.CategoryDeath, fmt.Sprintf(
		`<ansi fg="red">%s falls.</ansi> You have a feeling this is not the last you will see of them.`,
		comp.Name,
	))
}

// snapshotCompanionProgression copies a live companion's earned progression
// into its record. It mirrors the progression half of saveCompanionState and
// must be kept in step with it.
func snapshotCompanionProgression(comp *characters.CompanionInfo, mob *mobs.Mob) {
	comp.SchemaVersion = mobs.InstanceSchemaVersion
	if comp.StatTraining == nil {
		comp.StatTraining = make(map[string]int)
	}
	comp.StatTraining["strength"] = mob.Character.Stats.Strength.Training
	comp.StatTraining["dexterity"] = mob.Character.Stats.Dexterity.Training
	comp.StatTraining["perception"] = mob.Character.Stats.Perception.Training
	comp.StatTraining["vitality"] = mob.Character.Stats.Vitality.Training
	comp.StatTraining["willpower"] = mob.Character.Stats.Willpower.Training
	comp.StatTraining["charisma"] = mob.Character.Stats.Charisma.Training

	comp.Skills = copyIntMap(mob.Character.Skills)
	comp.SkillUseCount = copyIntMap(mob.Character.SkillUseCount)
	comp.Mutations = copyIntMap(mob.Character.Mutations)
	comp.SpellBook = copyIntMap(mob.Character.SpellBook)
	comp.MutationProgress = mob.Character.MutationProgress
}

// RespawnBondedCompanion brings a fallen bonded companion back into its
// owner's room during a session. It is installed into internal/companionai at
// boot and called by the aicompanion module once the companion has
// recovered. Login respawns fallen bonded companions on its own through
// respawnCompanions, so this is only for mid-session recovery.
//
// Returns the new instance id, or 0 if the owner is offline, the record is
// missing, the companion is already fielded, or the template is gone.
func RespawnBondedCompanion(userId int, mobId int) int {

	user := users.GetByUserId(userId)
	if user == nil || user.Character == nil {
		return 0
	}

	var comp *characters.CompanionInfo
	for i := range user.Character.Companions {
		c := &user.Character.Companions[i]
		if c.SourceType == characters.CompanionBonded && c.MobId == mobId {
			comp = c
			break
		}
	}
	if comp == nil || comp.InstanceId > 0 {
		return 0
	}

	mob := mobs.NewMobByIdFresh(mobs.MobId(comp.MobId), user.Character.RoomId)
	if mob == nil {
		mudlog.Error("RespawnBondedCompanion", "error", fmt.Sprintf("mob template %d not found for companion %s", comp.MobId, comp.Name))
		return 0
	}

	applyCompanionState(mob, comp)
	mob.Character.Name = comp.Name

	// Back on their feet, but not fresh: a recovered companion returns hurt.
	mob.Character.Health = mob.Character.HealthMax.Value / 2
	if mob.Character.Health < 1 {
		mob.Character.Health = 1
	}
	mob.Character.Stamina = mob.Character.StaminaMax.Value / 2
	mob.Character.Conviction = mob.Character.ConvictionMax.Value / 2

	mob.Character.Charm(user.UserId, -1, "")
	user.Character.TrackCharmed(mob.InstanceId, true)

	if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
		room.AddMob(mob.InstanceId)
	}

	comp.InstanceId = mob.InstanceId
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	return mob.InstanceId
}

// RejoinBondedCompanions brings every fielded companion of an owner to the
// owner's room through the normal follow transport. Installed into
// internal/companionai for the aicompanion module's last-resort return from
// an errand. Returns false if the owner is offline.
func RejoinBondedCompanions(userId int) bool {
	user := users.GetByUserId(userId)
	if user == nil || user.Character == nil {
		return false
	}
	TransportCompanions(user, 0, user.Character.RoomId)
	return true
}

// SnapshotBondedCompanion copies a fielded bonded companion's live gear,
// gold and progression into its CompanionInfo without despawning it, the
// same fields saveCompanionState writes at logout. Installed into
// internal/companionai. The engine snapshots companions only at logout, so
// without this a server restart or crash mid-session would lose everything
// the bonded companion bought, looted or learned since it last logged out.
// Returns false if there is no fielded bonded companion.
func SnapshotBondedCompanion(userId int) bool {
	user := users.GetByUserId(userId)
	if user == nil || user.Character == nil {
		return false
	}
	done := false
	for i := range user.Character.Companions {
		comp := &user.Character.Companions[i]
		if comp.SourceType != characters.CompanionBonded || comp.InstanceId == 0 {
			continue
		}
		mob := mobs.GetInstance(comp.InstanceId)
		if mob == nil {
			continue
		}
		snapshotCompanionProgression(comp, mob)
		if len(mob.Character.Items) > 0 {
			comp.Items = make([]items.Item, len(mob.Character.Items))
			copy(comp.Items, mob.Character.Items)
		} else {
			comp.Items = nil
		}
		comp.Equipment = mob.Character.Equipment
		comp.Gold = mob.Character.Gold
		done = true
	}
	return done
}
