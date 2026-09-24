// Package gossip is the gossip template store: pools of lines a gossiping
// NPC says about recent world events and known facts, keyed
// "EventType-Significance[-Local|-Distant]", "fact-<id|tag|default>" and
// "fallback", loaded from DataFiles/gossip_templates.yaml.
//
// Which key and which token apply is decided by the gossiper in
// internal/hooks (buildGossipLine); this package owns the text, its
// validation and the pick.
package gossip

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"gopkg.in/yaml.v2"
)

const fileName = "gossip_templates.yaml"

// tokens a template may carry; each at most once per line.
var tokens = []string{"{desc}", "{description}"}

var templates map[string][]string

// Load reads DataFiles/gossip_templates.yaml. A world with no such file (the
// default world) gets an empty store. A read error other than a missing file,
// a parse error, or a Validate failure panics, as a bad recipe or spell does:
// before the store existed these were logged and gossip went silently empty
// for the life of the process.
func Load() {
	start := time.Now()
	path := string(configs.GetFilePathsConfig().DataFiles) + "/" + fileName

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			templates = map[string][]string{}
			mudlog.Info("gossip.Load()", "loadedKeys", 0, "note", "this world has no "+fileName)
			return
		}
		panic(fmt.Errorf("gossip: read %s: %w", path, err))
	}

	loaded := map[string][]string{}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		panic(fmt.Errorf("gossip: parse %s: %w", path, err))
	}
	if err := Validate(loaded); err != nil {
		panic(fmt.Errorf("gossip: %s: %w", path, err))
	}

	templates = loaded
	mudlog.Info("gossip.Load()", "loadedKeys", len(templates), "Time Taken", time.Since(start))
}

// Validate refuses an empty pool, a blank line, and a line that carries the
// same token twice. The last rule is what keeps rendering byte-identical to
// the pre-store code, which replaced {desc} only once: with no repeats, once
// and every-occurrence substitution produce the same line.
func Validate(t map[string][]string) error {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pool := t[key]
		if len(pool) == 0 {
			return fmt.Errorf("key %q has no lines", key)
		}
		for i, line := range pool {
			if strings.TrimSpace(line) == "" {
				return fmt.Errorf("key %q line %d is blank", key, i)
			}
			for _, tok := range tokens {
				if strings.Count(line, tok) > 1 {
					return fmt.Errorf("key %q line %d uses %s more than once", key, i, tok)
				}
			}
		}
	}
	return nil
}

// Pool returns the lines for key, or nil.
func Pool(key string) []string {
	return templates[key]
}

// Keys returns every loaded key, sorted.
func Keys() []string {
	out := make([]string, 0, len(templates))
	for k := range templates {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Render picks one line from pool and substitutes token with value. It draws
// exactly once, through narration.DefaultPicker (util.Rand), as the pre-store
// code did with util.Rand(len(pool)). An empty token substitutes nothing.
func Render(pool []string, token, value string) string {
	return renderWith(pool, token, value, narration.DefaultPicker)
}

// renderWith is Render with an explicit picker. A gossiping NPC is the only
// speaker, so the pool is the Actor role and nothing else is authored.
func renderWith(pool []string, token, value string, pick narration.Picker) string {
	if len(pool) == 0 {
		return ""
	}
	var subs map[string]string
	if token != "" {
		subs = map[string]string{token: value}
	}
	return narration.Render(narration.Variants{Actor: pool}, subs, pick).Actor
}
