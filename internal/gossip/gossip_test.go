package gossip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name      string
		templates map[string][]string
		wantErr   string
	}{
		{"shipped shape passes", map[string][]string{
			"fallback":                {"Quiet day.", "Nothing to report."},
			"MobCraftedRare-Regional": {"I heard {desc}"},
			"fact-default":            {"They say {description}"},
		}, ""},
		{"empty pool refused", map[string][]string{"fallback": {}}, "has no lines"},
		{"blank line refused", map[string][]string{"fallback": {"ok", "   "}}, "is blank"},
		{"repeated desc refused", map[string][]string{"X-Local": {"{desc} and {desc}"}}, "{desc} more than once"},
		{"repeated description refused", map[string][]string{"fact-default": {"{description}, {description}"}}, "{description} more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.templates)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestRenderWith_OneDrawPerLine(t *testing.T) {
	for _, n := range []int{1, 2, 5} {
		pool := make([]string, n)
		for i := range pool {
			pool[i] = "line {desc}"
		}
		calls := 0
		counting := func(size int) int {
			calls++
			if size != n {
				t.Errorf("picker got size %d, want %d", size, n)
			}
			return 0
		}
		if got := renderWith(pool, "{desc}", "X", counting); got != "line X" {
			t.Errorf("pool of %d rendered %q, want %q", n, got, "line X")
		}
		if calls != 1 {
			t.Errorf("pool of %d drew %d times, want exactly 1 (today's util.Rand(len))", n, calls)
		}
	}
}

func TestRenderWith_EmptyPoolDrawsNothing(t *testing.T) {
	calls := 0
	if got := renderWith(nil, "{desc}", "X", func(int) int { calls++; return 0 }); got != "" || calls != 0 {
		t.Errorf("empty pool rendered %q with %d draws, want \"\" and 0", got, calls)
	}
}

func TestRenderWith_NoTokenLeavesLineAlone(t *testing.T) {
	if got := renderWith([]string{"Quiet {desc} day."}, "", "", func(int) int { return 0 }); got != "Quiet {desc} day." {
		t.Errorf("token-less render changed the line: %q", got)
	}
}

func TestRender_UsesTheDefaultPicker(t *testing.T) {
	src, err := os.ReadFile("gossip.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "renderWith(pool, token, value, narration.DefaultPicker)") {
		t.Error("Render must pass narration.DefaultPicker: gossip consumed one util.Rand draw per line before the migration, and FirstPicker would silently remove it and shift every later roll")
	}
}

func TestLoad_MissingFileIsAnEmptyStore(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(t.TempDir())
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest(map[string][]string{"fallback": {"stale"}})()

	Load()

	if got := Pool("fallback"); got != nil {
		t.Errorf("Pool after loading a world with no gossip file = %q, want nil", got)
	}
}

func TestLoad_ReadsAndValidatesTheFile(t *testing.T) {
	dir := t.TempDir()
	body := "fallback:\n  - \"Quiet day.\"\nX-Local:\n  - \"Near: {desc}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gossip_templates.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(dir)
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest(nil)()

	Load()

	if got := Render(Pool("X-Local"), "{desc}", "a bridge fell"); got != "Near: a bridge fell" {
		t.Errorf("rendered %q", got)
	}
	if keys := Keys(); len(keys) != 2 || keys[0] != "X-Local" || keys[1] != "fallback" {
		t.Errorf("Keys() = %q, want sorted [X-Local fallback]", keys)
	}
}
