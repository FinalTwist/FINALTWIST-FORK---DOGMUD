// Package tips is the periodic gameplay tip store: short pieces of advice
// broadcast to every player who has not turned them off (`set tips`), one per
// interval, in file order. Loaded from DataFiles/tips.yaml. Before messaging
// M3 item 7 these were called hints, a name that collided with the quest
// `hint` command and with dialogue hints.
//
// Tips are sent as one line each with no server-side wrap; see the messaging
// M6 content ledger, row 22.
package tips

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

const fileName = "tips.yaml"

var (
	all  []string
	next int
)

// Load reads DataFiles/tips.yaml (key `tips:`). A world with no such file gets
// an empty store. Any other read error, a parse error or a Validate failure
// panics. The rotation position survives a reload.
func Load() {
	start := time.Now()
	path := string(configs.GetFilePathsConfig().DataFiles) + "/" + fileName

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			all = nil
			mudlog.Info("tips.Load()", "loadedCount", 0, "note", "this world has no "+fileName)
			return
		}
		panic(fmt.Errorf("tips: read %s: %w", path, err))
	}

	var file struct {
		Tips []string `yaml:"tips"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		panic(fmt.Errorf("tips: parse %s: %w", path, err))
	}
	if err := Validate(file.Tips); err != nil {
		panic(fmt.Errorf("tips: %s: %w", path, err))
	}

	all = file.Tips
	mudlog.Info("tips.Load()", "loadedCount", len(all), "Time Taken", time.Since(start))
}

// Validate refuses a blank tip. There is deliberately no length rule: most
// shipped tips run past 80 characters, and wrapping is the messaging arc's M5
// decision, not an authoring rule this store can enforce.
func Validate(t []string) error {
	for i, tip := range t {
		if strings.TrimSpace(tip) == "" {
			return fmt.Errorf("tip %d is blank", i)
		}
	}
	return nil
}

// Next returns the next tip in rotation and advances, or "" for an empty store.
func Next() string {
	if len(all) == 0 {
		return ""
	}
	tip := all[next%len(all)]
	next++
	return tip
}

// Count is the number of loaded tips.
func Count() int { return len(all) }

// All returns a copy of the loaded tips in rotation order.
func All() []string {
	return append([]string(nil), all...)
}
