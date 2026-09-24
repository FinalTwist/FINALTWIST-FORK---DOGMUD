package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
)

// Handles force rebuilding maps via event
func HandleMapRebuild(e events.Event) events.ListenerReturn {

	evt := e.(events.RebuildMap)

	mapper.GetMapper(evt.MapRootRoomId, !evt.SkipIfExists)

	return events.Continue
}
