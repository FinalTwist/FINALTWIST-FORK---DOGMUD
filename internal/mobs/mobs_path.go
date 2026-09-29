package mobs

type PathRoom interface {
	ExitName() string
	RoomId() int
	Waypoint() bool
}

type PathQueue struct {
	roomQueue   []PathRoom
	currentRoom PathRoom
}

func (p PathQueue) Len() int {
	return len(p.roomQueue)
}

func (p *PathQueue) Clear() {
	p.roomQueue = []PathRoom{}
	p.currentRoom = nil
}

func (p PathQueue) Current() PathRoom {
	return p.currentRoom
}

func (p *PathQueue) Next() PathRoom {
	if len(p.roomQueue) == 0 {
		return nil
	}
	p.currentRoom = p.roomQueue[0]
	p.roomQueue = p.roomQueue[1:]
	return p.currentRoom
}

// Peek returns the next step without advancing the queue, or nil when the
// queue is empty. The path walker quotes a step before taking it, and a step
// it cannot afford yet must stay queued (movement parity 4b).
func (p PathQueue) Peek() PathRoom {
	if len(p.roomQueue) == 0 {
		return nil
	}
	return p.roomQueue[0]
}

// returns a list of remaining waypoint roomIds
func (p *PathQueue) Waypoints() []int {
	wpList := []int{}
	if p.currentRoom != nil && p.currentRoom.Waypoint() {
		wpList = append(wpList, p.currentRoom.RoomId())
	}

	for _, r := range p.roomQueue {
		if r.Waypoint() {
			wpList = append(wpList, r.RoomId())
		}
	}

	return wpList
}

func (p *PathQueue) SetPath(path []PathRoom) {
	p.roomQueue = path
	p.currentRoom = nil
}
