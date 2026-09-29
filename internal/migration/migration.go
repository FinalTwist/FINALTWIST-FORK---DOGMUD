package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/version"
)

// Migration code goes here.
// They should be put in the order of oldest to newest and follow the pattern as below
func doAllMigrations(lastConfigVersion version.Version) error {

	// 0.0.0 -> 0.9.1
	if lastConfigVersion.IsOlderThan(version.New(0, 9, 1)) {

		if err := migrate_RoomZoneConfig(); err != nil {
			return err
		}

	}

	// 0.9.1 -> 0.10.0
	if lastConfigVersion.IsOlderThan(version.New(0, 10, 0)) {

		if err := migrate_UserStatsRename(); err != nil {
			return err
		}

	}

	// 0.10.0 -> 0.11.0
	if lastConfigVersion.IsOlderThan(version.New(0, 11, 0)) {

		if err := migrate_RollCharacterStats(); err != nil {
			return err
		}

	}

	// 0.11.0 -> 0.12.0
	if lastConfigVersion.IsOlderThan(version.New(0, 12, 0)) {

		if err := migrate_RaceToSpecies(); err != nil {
			return err
		}

	}

	// 0.12.0 -> 0.13.0
	if lastConfigVersion.IsOlderThan(version.New(0, 13, 0)) {

		// Migrations run BEFORE main.go's data-load block, so
		// factions.LoadAllDefinitions() has not yet been called.
		// Load now so the rep-seeding can resolve the warren faction.
		if err := factions.LoadAllDefinitions(); err != nil {
			return err
		}

		if err := migrate_SeedWarrenRepFromQuestToken(); err != nil {
			return err
		}

	}

	if lastConfigVersion.IsOlderThan(version.New(0, 14, 0)) {
		// Player mutation migration: reclassify every save onto the cluster
		// graph (wipe retired-41, grant a cluster seed). Datafiles are backed
		// up by Run() before this and restored on error (spec §7 safety).
		if err := migrate_ReclassifyPlayerMutations(false); err != nil {
			return err
		}
	}

	if lastConfigVersion.IsOlderThan(version.New(0, 15, 0)) {
		// Authored coordinate model: backfill x/y/z/plane onto every non-instance
		// room by crawling spatial exit deltas per connected component. Datafiles
		// are backed up by Run() before this and restored on error.
		if err := migrate_BackfillCoords(false); err != nil {
			return err
		}
	}

	if lastConfigVersion.IsOlderThan(version.New(0, 16, 0)) {
		// One-time exploit remediation: fyttyn's vitality of 411 was ground via a
		// since-fixed exploit and was being compressed to an effective 280 by the
		// stat soft cap. With the soft cap removed, freeze raw vitality at the
		// value actually in play rather than handing back 131 unearned points.
		if err := migrate_FreezeExploitedVitality(false); err != nil {
			return err
		}
	}

	if lastConfigVersion.IsOlderThan(version.New(0, 17, 0)) {
		// Rename the old condition key spellings (conditionrename) in every
		// .yaml and .plugin.dat file under DataFiles (item overrides are saved
		// almost everywhere). Datafiles are backed up by Run() before this and
		// restored on error.
		if err := migrate_ConditionKeys(false); err != nil {
			return err
		}
	}

	if lastConfigVersion.IsOlderThan(version.New(0, 18, 0)) {
		// Rename each player's configoptions.hints to configoptions.tips (the
		// broadcast was renamed in messaging M3 item 7).
		if err := migrate_TipsConfigOption(false); err != nil {
			return err
		}
	}

	return nil
}

// Entrypoint for migrations.
// This is run on server start-up, after config files are loaded.
// NOTE: This means migrations that modify config files themselves would need special consideration
func Run(lastConfigVersion version.Version, serverVersion version.Version) error {

	//
	// If already up to speed on version, we don't really need to do anything.
	//
	if lastConfigVersion.IsEqualTo(serverVersion) {
		return nil
	}

	//
	// Start by making a backup of all datafiles.
	//
	backupFolder, err := datafilesBackup()
	if err != nil {
		return fmt.Errorf(`could not backup datafiles: %w`, err)
	}
	defer os.RemoveAll(backupFolder)

	//
	// If an error occured, restore backup
	//
	if err := doAllMigrations(lastConfigVersion); err != nil {
		copyDir(backupFolder, string(configs.GetFilePathsConfig().DataFiles))
		return err
	}

	//
	// Finally, since successful, update to the version this migration is for
	//
	recordMigratedVersion(serverVersion)

	return nil
}

// isUserSaveFile reports whether path is a user save (users/<id>.yaml) that a
// user-file migration should parse as a mapping. It rejects users.idx and
// <id>.alts.yaml: internal/characters/alts.go writes an alts file as a YAML
// SEQUENCE of characters (`[]` when empty), which a map-shaped user migration
// cannot parse. internal/users/character_index.go applies the same rule when
// it scans users/. It also rejects the legacy <name>-alts.yaml, the same
// sequence under its pre-rename name (internal/users/migration.go renames it).
func isUserSaveFile(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".yaml") &&
		!strings.HasSuffix(name, ".alts.yaml") &&
		!strings.HasSuffix(name, "-alts.yaml") &&
		name != "users.idx"
}

// recordMigratedVersion stores the version the data files now match. It goes
// through configs.SetEngineVal because the shipped Server.Locked names
// Server.CurrentVersion, and SetVal honours that list; a refused write would
// leave the old version on disk and re-run every migration on the next boot.
// A failed write is logged, not returned, so a read-only config directory
// still boots as it did before.
func recordMigratedVersion(v version.Version) {
	if err := configs.SetEngineVal(`Server.CurrentVersion`, v.String()); err != nil {
		mudlog.Error(`migration`, `action`, `record version`, `version`, v.String(), `error`, err)
	}
}
