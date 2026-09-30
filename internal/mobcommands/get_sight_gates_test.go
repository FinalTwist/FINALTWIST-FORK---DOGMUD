package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// A mob in the dark picks up nothing, gold included, and stays hidden.
func TestMobGet_DarkRefusesSilently(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	room.Items = append(room.Items, items.Item{ItemId: 96210, Spec: &items.ItemSpec{ItemId: 96210, Name: "pebble"}})
	room.Gold = 7
	Get("pebble", mob, room)
	Get("gold", mob, room)
	_, carried := mob.Character.FindInBackpack("pebble")
	assert.False(t, carried)
	assert.Equal(t, 7, room.Gold)
}

func TestMobGet_ExplodingItemRefused(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.Lamp = rooms.LampPtr(90)
	room.Items = append(room.Items, items.Item{ItemId: 96211, Spec: &items.ItemSpec{ItemId: 96211, Name: "bomb"}, Adjectives: []string{`exploding`}})
	Get("bomb", mob, room)
	_, carried := mob.Character.FindInBackpack("bomb")
	assert.False(t, carried)
}
