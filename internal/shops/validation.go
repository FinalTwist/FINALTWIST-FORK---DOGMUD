package shops

import (
	"fmt"
	"sort"
	"strings"
)

// ShopBearingMob is the minimal interface ValidateShopMobTags needs.
// Production code wraps mobs.Mob in this interface (see main.go).
type ShopBearingMob interface {
	GetMobId() int
	GetName() string
	GetZone() string
	HasShop() bool
	IsCrafter() bool
	GetShopCraftSupport() string
	IsFence() bool
}

// ValidateShopMobTags walks all candidate mobs and returns a non-nil
// error if any shop-bearing mob (HasShop or IsCrafter) lacks a valid
// craft_support: tag. The error message lists every offending mob so
// a single restart surfaces the full set.
//
// A fence (IsFence: a mob in Balance.BaubleFenceGroups) may omit the tag:
// its shop then buys no ordinary loot (vendorAcceptsAny matches nothing)
// and keeps its gold for baubles, which the fence buys on its own path. A
// tag a fence does carry must still be valid.
//
// Caller behavior on non-nil error:
//   - Cold boot: panic. The server refuses to start with bad tags.
//   - /reload: log a structured Error with the remediation hint and
//     keep running. Mob templates may be in an inconsistent state
//     (the dashboard will show "(uncategorized)" rows for affected
//     shops) until the admin fixes the YAMLs and reloads again.
//
// See main.go::loadAllDataFiles for the actual dispatch.
func ValidateShopMobTags(mobs []ShopBearingMob) error {
	type fault struct {
		mobId int
		name  string
		zone  string
		tag   string
		why   string
	}
	var faults []fault

	for _, m := range mobs {
		if !m.HasShop() && !m.IsCrafter() {
			continue
		}
		tag := m.GetShopCraftSupport()
		if tag == "" && m.IsFence() {
			continue
		}
		if tag == "" {
			faults = append(faults, fault{m.GetMobId(), m.GetName(), m.GetZone(), tag, "missing craft_support:"})
			continue
		}
		if !IsValidCraftSupport(tag) {
			faults = append(faults, fault{m.GetMobId(), m.GetName(), m.GetZone(), tag, "invalid value (not in ValidCraftSupports)"})
		}
	}

	if len(faults) == 0 {
		return nil
	}

	sort.Slice(faults, func(i, j int) bool { return faults[i].mobId < faults[j].mobId })
	var b strings.Builder
	fmt.Fprintf(&b, "shop-bearing mobs with bad craft_support tags (%d):\n", len(faults))
	for _, f := range faults {
		fmt.Fprintf(&b, "  - mob %d (%s, zone=%s): %s (got %q)\n", f.mobId, f.name, f.zone, f.why, f.tag)
	}
	b.WriteString("Valid values: ")
	b.WriteString(strings.Join(ValidCraftSupports, ", "))
	return fmt.Errorf("%s", b.String())
}
