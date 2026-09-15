package migration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const oldUserSave = `userid: 7
username: probe
character:
  name: Probe
  buffs:
    list:
    - buffid: 122
      triggersleft: 4
      stacks:
      - roundsleft: 3
        amount: -2
    - buffid: 79
      permabuff: true
  pet:
    type: dog
    buffids: [12, 13]
  shop:
  - itemid: 5
  - buffid: 4
    price: 10
  miscdata:
    pinnacle_bandolier_buffs: [51]
    unrelated: keep
`

const oldAltsSave = `- name: AltOne
  buffs:
    list:
    - buffid: 3
      permabuff: true
- name: AltTwo
  pet:
    type: cat
    buffids: [9]
`

const oldRoomInstance = `roomid: 1001
containers:
  chest:
    lock:
      difficulty: 3
      trapbuffids: [45]
    gold: 7
  crate:
    gold: 1
`

func writeFixture(t *testing.T, dir, rel, body string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
	require.NoError(t, os.WriteFile(p, []byte(body), 0644))
	return p
}

func TestConditionKeys_UserSave(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	s := string(raw)
	for _, gone := range []string{"buffs:", "buffid:", "permabuff:", "buffids:", "pinnacle_bandolier_buffs"} {
		assert.NotContains(t, s, gone)
	}

	var u users.UserRecord
	require.NoError(t, yaml.Unmarshal(raw, &u))
	require.Len(t, u.Character.Conditions.List, 2)
	assert.Equal(t, 122, u.Character.Conditions.List[0].ConditionId)
	assert.Len(t, u.Character.Conditions.List[0].Stacks, 1)
	assert.Equal(t, 79, u.Character.Conditions.List[1].ConditionId)
	assert.True(t, u.Character.Conditions.List[1].Permanent)
	assert.Equal(t, []int{12, 13}, u.Character.Pet.ConditionIds)
	require.Len(t, u.Character.Shop, 2)
	assert.Equal(t, 4, u.Character.Shop[1].ConditionId)
	assert.NotNil(t, u.Character.MiscData["pinnacle_bandolier_conditions"])
	assert.Equal(t, "keep", u.Character.MiscData["unrelated"])
}

func TestConditionKeys_AltsList(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.alts.yaml", oldAltsSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var alts []characters.Character
	require.NoError(t, yaml.Unmarshal(raw, &alts))
	require.Len(t, alts, 2)
	require.Len(t, alts[0].Conditions.List, 1)
	assert.Equal(t, 3, alts[0].Conditions.List[0].ConditionId)
	assert.True(t, alts[0].Conditions.List[0].Permanent)
	assert.Equal(t, []int{9}, alts[1].Pet.ConditionIds)
}

func TestConditionKeys_RoomInstance(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "rooms.instances/frostfang/1001.yaml", oldRoomInstance)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var r rooms.Room
	require.NoError(t, yaml.Unmarshal(raw, &r))
	assert.Equal(t, []int{45}, r.Containers["chest"].Lock.TrapConditionIds)
	assert.Equal(t, 1, r.Containers["crate"].Gold)
}

func TestConditionKeys_SecondRunIsANoOp(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	first, err := os.ReadFile(p)
	require.NoError(t, err)
	info1, err := os.Stat(p)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	second, err := os.ReadFile(p)
	require.NoError(t, err)
	info2, err := os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
	assert.Equal(t, info1.ModTime(), info2.ModTime(), "an already-migrated file must not be rewritten")
}

func TestConditionKeys_UntouchedFileNotRewritten(t *testing.T) {
	dir := t.TempDir()
	body := "userid: 8\nusername: clean\ncharacter:\n  name: Clean\n"
	p := writeFixture(t, dir, "users/8.yaml", body)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, body, string(raw), "a save with nothing to rename must keep its exact bytes")
}

func TestConditionKeys_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, true))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, oldUserSave, string(raw))
}

func TestConditionKeys_CollisionIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "users/9.yaml", "character:\n  buffs:\n    list: []\n  conditions:\n    list: []\n")
	err := migrateConditionKeysIn(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9.yaml")
}

func TestConditionKeys_UnparseableIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "users/10.yaml", "character: [unclosed\n")
	err := migrateConditionKeysIn(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10.yaml")
}

func TestConditionKeys_MissingFoldersAreNotErrors(t *testing.T) {
	require.NoError(t, migrateConditionKeysIn(t.TempDir(), false))
}
