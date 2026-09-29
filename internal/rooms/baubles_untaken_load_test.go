package rooms

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v2"
)

// A find that has lain untaken past its limit is gone the moment its room
// is loaded from its instance file, before anything (a mob wandering in,
// which never prepares the room) can pick it up. The bauble sweep stops
// counting such a find as a reference and may prune its record, so this
// must hold.
func TestLoadRoomInstanceRemovesExpiredUntakenFinds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	tempDir := t.TempDir()
	prev := configs.GetFilePathsConfig()
	if err := configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": tempDir}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev.DataFiles.String()})
	}()

	template := &Room{RoomId: 90103, Zone: "test_zone", Title: "Untaken Test", Description: "A test room.", Exits: map[string]exit.RoomExit{}}
	templateYAML, err := yaml.Marshal(template)
	if err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(tempDir, "rooms", "test_zone", "90103.yaml")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(templatePath, templateYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	instance := map[string]any{
		"items": []map[string]any{
			{"itemid": items.BaubleItemId, "bauble": "B0000001", "baubleleftat": now.Add(-25 * time.Hour).Unix()},
			{"itemid": items.BaubleItemId, "bauble": "B0000002", "baubleleftat": now.Add(-time.Hour).Unix()},
		},
	}
	instanceYAML, err := yaml.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	instancePath := filepath.Join(tempDir, "rooms.instances", "test_zone", "90103.yaml")
	if err := os.MkdirAll(filepath.Dir(instancePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instancePath, instanceYAML, 0o644); err != nil {
		t.Fatal(err)
	}
	roomManager.setCachedFilePath(90103, "test_zone/90103.yaml")

	loaded := LoadRoomInstance(90103)
	if loaded == nil {
		t.Fatal("room did not load")
	}
	if len(loaded.Items) != 1 || loaded.Items[0].Bauble != "B0000002" {
		t.Fatalf("items %+v, want only the young find B0000002", loaded.Items)
	}
}
