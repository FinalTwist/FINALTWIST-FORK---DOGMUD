package main

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// logBootConfig hands every config value to emit in sorted key order, read
// through DisplayConfigData so no secret reaches the boot log. It is split out
// of main() so a test can read exactly what the boot log would print.
func logBootConfig(c configs.Config, emit func(name string, value any)) {
	cfgData := c.DisplayConfigData()
	cfgKeys := make([]string, 0, len(cfgData))
	for k := range cfgData {
		cfgKeys = append(cfgKeys, k)
	}
	slices.Sort(cfgKeys)
	for _, k := range cfgKeys {
		emit(k, cfgData[k])
	}
}
