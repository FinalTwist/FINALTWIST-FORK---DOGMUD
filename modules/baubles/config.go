package baubles

import (
	"strconv"
	"strings"
)

// Config is the resolved Modules.baubles configuration. Values arrive
// through the plugin config bag as `any`, so the readers tolerate the types
// YAML and the config overlay can produce (the same rules as aicompanion).
//
// The key, the endpoint, the day's token budget and the breaker are NOT
// here: they are the server's, shared with every feature that calls a model
// (the APIFramework config section, internal/apiframework).
type Config struct {
	Enabled bool

	// UsePlayerKeys lets a finder's own key (the companion key relay) name
	// what they find, when they ticked "Also name things I find while
	// searching" on the key page. Their key is tried first; the server's
	// key second.
	UsePlayerKeys bool

	Model               string
	ReasoningEffort     string
	Temperature         float64
	TimeoutSeconds      int
	MaxCompletionTokens int
	RetryTransient      bool
	MaxConcurrent       int

	ModerateOutput  bool
	ModerationModel string
	LogRequests     bool
}

type getter func(key string) any

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(t))
		return b
	}
	return false
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case uint64:
		return int(t)
	case float64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	}
	return def
}

func asFloat(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f
		}
	}
	return def
}

func clampInt(v int, lo int, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// effort normalises a reasoning-effort setting; anything the API does not
// accept means "do not send".
func effort(v string) string {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case `none`, `minimal`, `low`, `medium`, `high`:
		return v
	}
	return ``
}

// buildConfig resolves the config with safe defaults and bounds, so a missing
// or mistyped key can never produce a zero timeout.
//
// Enabled defaults to FALSE: nothing is named, and no room text goes to any
// model, until an operator switches it on (and Balance.BaublesEnabled must
// be on for there to be finds at all).
func buildConfig(get getter) Config {
	if get == nil {
		get = func(string) any { return nil }
	}
	c := Config{
		UsePlayerKeys:       true,
		Model:               strings.TrimSpace(asString(get(`Model`))),
		ReasoningEffort:     `minimal`,
		Temperature:         asFloat(get(`Temperature`), 0),
		TimeoutSeconds:      clampInt(asInt(get(`TimeoutSeconds`), 15), 3, 30),
		MaxCompletionTokens: clampInt(asInt(get(`MaxCompletionTokens`), 800), 200, 4000),
		RetryTransient:      asBool(get(`RetryTransient`)),
		MaxConcurrent:       clampInt(asInt(get(`MaxConcurrent`), 4), 1, 32),
		ModerateOutput:      true,
		ModerationModel:     strings.TrimSpace(asString(get(`ModerationModel`))),
		LogRequests:         asBool(get(`LogRequests`)),
	}
	c.Enabled = asBool(get(`Enabled`))
	if v := get(`UsePlayerKeys`); v != nil {
		c.UsePlayerKeys = asBool(v)
	}
	if v := get(`ModerateOutput`); v != nil {
		c.ModerateOutput = asBool(v)
	}
	if v := get(`ReasoningEffort`); v != nil {
		c.ReasoningEffort = effort(asString(v))
	}
	if c.Model == `` {
		c.Model = `gpt-5-nano`
	}
	if c.ModerationModel == `` {
		c.ModerationModel = `omni-moderation-latest`
	}
	return c
}
