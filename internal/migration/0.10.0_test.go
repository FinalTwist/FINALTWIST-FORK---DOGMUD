package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// A users/ directory holds <id>.alts.yaml beside <id>.yaml, and an alts file
// is a YAML sequence (`[]` when empty). 0.10.0 must migrate the save and leave
// the alts file alone, not die parsing it as a map.
func TestUserStatsRename_SkipsAltsFile(t *testing.T) {
	dataDir := t.TempDir()
	usersDir := filepath.Join(dataDir, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatal(err)
	}

	userPath := filepath.Join(usersDir, "1.yaml")
	userBody := "userid: 1\n" +
		"username: tester\n" +
		"character:\n" +
		"  name: tester\n" +
		"  stats:\n" +
		"    speed:\n" +
		"      training: 3\n" +
		"      base: 90\n"
	if err := os.WriteFile(userPath, []byte(userBody), 0644); err != nil {
		t.Fatal(err)
	}

	altsPath := filepath.Join(usersDir, "1.alts.yaml")
	altsBody := []byte("[]\n")
	if err := os.WriteFile(altsPath, altsBody, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(dataDir)
	configs.SetConfigForTest(t, cfg)

	if err := migrate_UserStatsRename(); err != nil {
		t.Fatalf("migrate_UserStatsRename: %v", err)
	}

	raw, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Character struct {
			Stats map[string]map[string]int `yaml:"stats"`
		} `yaml:"character"`
	}
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if _, stale := got.Character.Stats["speed"]; stale {
		t.Errorf("1.yaml still carries speed:\n%s", raw)
	}
	if got.Character.Stats["dexterity"]["base"] != 90 {
		t.Errorf("1.yaml dexterity.base = %d, want 90:\n%s", got.Character.Stats["dexterity"]["base"], raw)
	}

	altsAfter, err := os.ReadFile(altsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(altsAfter) != string(altsBody) {
		t.Errorf("1.alts.yaml changed: got %q, want %q", altsAfter, altsBody)
	}
}

func TestIsUserSaveFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join("users", "1.yaml"), true},
		{filepath.Join("users", "42.yaml"), true},
		{filepath.Join("users", "1.alts.yaml"), false},
		{filepath.Join("users", "someone-alts.yaml"), false},
		{filepath.Join("users", "users.idx"), false},
		{filepath.Join("users", "notes.txt"), false},
		{"1.alts.yaml", false},
	}
	for _, c := range cases {
		if got := isUserSaveFile(c.path); got != c.want {
			t.Errorf("isUserSaveFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
