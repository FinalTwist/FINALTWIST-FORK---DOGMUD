package worldevents

// ResetForTest empties the ring buffer, so a test that drives a real death or
// milestone cannot leak its event into a later test that reads the feed (the
// hooks gossip tests assert on an empty feed).
//
// FOR TEST USE ONLY.
func ResetForTest() {
	eventBuffer = eventBuffer[:0]
}
