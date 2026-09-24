package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

//
// Check all zones and update their mutators.
//

func UpdateZoneMutators(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)

	// Update all zone based mutators once a round
	zoneNames, _ := rooms.GetZonesWithMutators()
	for _, zoneName := range zoneNames {
		if zoneInfo := rooms.GetZoneConfig(zoneName); zoneInfo != nil {
			zoneInfo.Mutators.Update(evt.RoundNumber)
		}
	}

	return events.Continue
}
