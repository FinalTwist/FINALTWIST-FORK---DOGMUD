package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKDStats_GetMobKDRatio(t *testing.T) {
	tests := []struct {
		name        string
		totalKills  int
		totalDeaths int
		want        float64
	}{
		{
			name:        "No deaths, some kills",
			totalKills:  10,
			totalDeaths: 0,
			want:        10.0,
		},
		{
			name:        "No kills, no deaths",
			totalKills:  0,
			totalDeaths: 0,
			want:        0.0,
		},
		{
			name:        "Some kills, some deaths",
			totalKills:  8,
			totalDeaths: 2,
			want:        4.0,
		},
		{
			name:        "More deaths than kills",
			totalKills:  3,
			totalDeaths: 6,
			want:        0.5,
		},
		{
			name:        "Equal kills and deaths",
			totalKills:  5,
			totalDeaths: 5,
			want:        1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalKills:  tt.totalKills,
				TotalDeaths: tt.totalDeaths,
			}
			got := kd.GetMobKDRatio()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestKDStats_GetPvpKDRatio(t *testing.T) {
	tests := []struct {
		name           string
		totalPvpKills  int
		totalPvpDeaths int
		want           float64
	}{
		{
			name:           "No PvP deaths, some PvP kills",
			totalPvpKills:  7,
			totalPvpDeaths: 0,
			want:           7.0,
		},
		{
			name:           "No PvP kills, no PvP deaths",
			totalPvpKills:  0,
			totalPvpDeaths: 0,
			want:           0.0,
		},
		{
			name:           "Some PvP kills, some PvP deaths",
			totalPvpKills:  12,
			totalPvpDeaths: 3,
			want:           4.0,
		},
		{
			name:           "More PvP deaths than kills",
			totalPvpKills:  2,
			totalPvpDeaths: 8,
			want:           0.25,
		},
		{
			name:           "Equal PvP kills and deaths",
			totalPvpKills:  5,
			totalPvpDeaths: 5,
			want:           1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalPvpKills:  tt.totalPvpKills,
				TotalPvpDeaths: tt.totalPvpDeaths,
			}
			got := kd.GetPvpKDRatio()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestKDStats_GetMobKills(t *testing.T) {
	tests := []struct {
		name       string
		kills      map[int]int
		totalKills int
		input      []int
		want       int
	}{
		{
			name:       "No mobId provided returns TotalKills",
			kills:      map[int]int{1: 2, 2: 3},
			totalKills: 5,
			input:      []int{},
			want:       5,
		},
		{
			name:       "Single mobId present in Kills map",
			kills:      map[int]int{1: 4, 2: 1},
			totalKills: 5,
			input:      []int{1},
			want:       4,
		},
		{
			name:       "Single mobId not present in Kills map",
			kills:      map[int]int{1: 2, 2: 3},
			totalKills: 5,
			input:      []int{3},
			want:       0,
		},
		{
			name:       "Multiple mobIds, all present",
			kills:      map[int]int{1: 2, 2: 3, 3: 4},
			totalKills: 9,
			input:      []int{1, 2, 3},
			want:       9,
		},
		{
			name:       "Multiple mobIds, some not present",
			kills:      map[int]int{1: 2, 2: 3},
			totalKills: 5,
			input:      []int{1, 3},
			want:       2,
		},
		{
			name:       "Kills map is nil, mobId provided",
			kills:      nil,
			totalKills: 0,
			input:      []int{1},
			want:       0,
		},
		{
			name:       "Kills map is nil, no mobId provided",
			kills:      nil,
			totalKills: 0,
			input:      []int{},
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				Kills:      tt.kills,
				TotalKills: tt.totalKills,
			}
			got := kd.GetMobKills(tt.input...)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestKDStats_AddPlayerKill(t *testing.T) {
	tests := []struct {
		name           string
		initialKills   map[string]int
		initialTotal   int
		killedUserId   int
		killedCharName string
		expectedKey    string
		expectedCount  int
		expectedTotal  int
	}{
		{
			name:           "Add kill to empty PlayerKills map",
			initialKills:   nil,
			initialTotal:   0,
			killedUserId:   42,
			killedCharName: "Alice",
			expectedKey:    "42:Alice",
			expectedCount:  1,
			expectedTotal:  1,
		},
		{
			name:           "Add kill to existing PlayerKills map, new player",
			initialKills:   map[string]int{"1:Bob": 2},
			initialTotal:   2,
			killedUserId:   99,
			killedCharName: "Charlie",
			expectedKey:    "99:Charlie",
			expectedCount:  1,
			expectedTotal:  3,
		},
		{
			name:           "Add kill to existing PlayerKills map, existing player",
			initialKills:   map[string]int{"7:Dave": 3},
			initialTotal:   3,
			killedUserId:   7,
			killedCharName: "Dave",
			expectedKey:    "7:Dave",
			expectedCount:  4,
			expectedTotal:  4,
		},
		{
			name:           "Add kill with username containing colon",
			initialKills:   nil,
			initialTotal:   0,
			killedUserId:   5,
			killedCharName: "Eve:Smith",
			expectedKey:    "5:Eve:Smith",
			expectedCount:  1,
			expectedTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				PlayerKills:   tt.initialKills,
				TotalPvpKills: tt.initialTotal,
			}
			kd.AddPlayerKill(tt.killedUserId, tt.killedCharName)
			assert.Equal(t, tt.expectedCount, kd.PlayerKills[tt.expectedKey])
			assert.Equal(t, tt.expectedTotal, kd.TotalPvpKills)
		})
	}
}
func TestKDStats_AddPlayerDeath(t *testing.T) {
	tests := []struct {
		name             string
		initialDeaths    map[string]int
		killedByUserId   int
		killedByCharName string
		expectedKey      string
		expectedCount    int
	}{
		{
			name:             "Add death to empty PlayerDeaths map",
			initialDeaths:    nil,
			killedByUserId:   10,
			killedByCharName: "Zara",
			expectedKey:      "10:Zara",
			expectedCount:    1,
		},
		{
			name:             "Add death to existing PlayerDeaths map, new killer",
			initialDeaths:    map[string]int{"1:Bob": 2},
			killedByUserId:   99,
			killedByCharName: "Charlie",
			expectedKey:      "99:Charlie",
			expectedCount:    1,
		},
		{
			name:             "Add death to existing PlayerDeaths map, existing killer",
			initialDeaths:    map[string]int{"7:Dave": 3},
			killedByUserId:   7,
			killedByCharName: "Dave",
			expectedKey:      "7:Dave",
			expectedCount:    4,
		},
		{
			name:             "Add death with username containing colon",
			initialDeaths:    nil,
			killedByUserId:   5,
			killedByCharName: "Eve:Smith",
			expectedKey:      "5:Eve:Smith",
			expectedCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				PlayerDeaths: tt.initialDeaths,
			}
			kd.AddPlayerDeath(tt.killedByUserId, tt.killedByCharName)
			assert.Equal(t, tt.expectedCount, kd.PlayerDeaths[tt.expectedKey])
		})
	}
}
func TestKDStats_AddMobKill(t *testing.T) {
	tests := []struct {
		name          string
		initialKills  map[int]int
		initialTotal  int
		mobId         int
		expectedMap   map[int]int
		expectedTotal int
	}{
		{
			name:          "Add kill to empty Kills map",
			initialKills:  nil,
			initialTotal:  0,
			mobId:         1,
			expectedMap:   map[int]int{1: 1},
			expectedTotal: 1,
		},
		{
			name:          "Add kill to existing Kills map, new mobId",
			initialKills:  map[int]int{2: 3},
			initialTotal:  3,
			mobId:         5,
			expectedMap:   map[int]int{2: 3, 5: 1},
			expectedTotal: 4,
		},
		{
			name:          "Add kill to existing Kills map, existing mobId",
			initialKills:  map[int]int{7: 2},
			initialTotal:  2,
			mobId:         7,
			expectedMap:   map[int]int{7: 3},
			expectedTotal: 3,
		},
		{
			name:          "Add kill with negative mobId",
			initialKills:  nil,
			initialTotal:  0,
			mobId:         -10,
			expectedMap:   map[int]int{-10: 1},
			expectedTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				Kills:      tt.initialKills,
				TotalKills: tt.initialTotal,
			}
			kd.AddMobKill(tt.mobId)
			assert.Equal(t, tt.expectedTotal, kd.TotalKills)
			for k, v := range tt.expectedMap {
				assert.Equal(t, v, kd.Kills[k])
			}
		})
	}
}
func TestKDStats_GetMobDeaths(t *testing.T) {
	tests := []struct {
		name        string
		totalDeaths int
		want        int
	}{
		{
			name:        "Zero deaths",
			totalDeaths: 0,
			want:        0,
		},
		{
			name:        "Some deaths",
			totalDeaths: 5,
			want:        5,
		},
		{
			name:        "Many deaths",
			totalDeaths: 123,
			want:        123,
		},
		{
			name:        "Negative deaths (should not happen, but test anyway)",
			totalDeaths: -3,
			want:        -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalDeaths: tt.totalDeaths,
			}
			got := kd.GetMobDeaths()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestKDStats_GetPvpDeaths(t *testing.T) {
	tests := []struct {
		name           string
		totalPvpDeaths int
		want           int
	}{
		{
			name:           "Zero PvP deaths",
			totalPvpDeaths: 0,
			want:           0,
		},
		{
			name:           "Some PvP deaths",
			totalPvpDeaths: 7,
			want:           7,
		},
		{
			name:           "Many PvP deaths",
			totalPvpDeaths: 123,
			want:           123,
		},
		{
			name:           "Negative PvP deaths (should not happen, but test anyway)",
			totalPvpDeaths: -5,
			want:           -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalPvpDeaths: tt.totalPvpDeaths,
			}
			got := kd.GetPvpDeaths()
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestKDStats_AddMobDeath(t *testing.T) {
	tests := []struct {
		name      string
		initial   int
		increment int
		expected  int
	}{
		{
			name:      "Add single mob death to zero",
			initial:   0,
			increment: 1,
			expected:  1,
		},
		{
			name:      "Add multiple mob deaths to zero",
			initial:   0,
			increment: 3,
			expected:  3,
		},
		{
			name:      "Add mob death to existing positive count",
			initial:   5,
			increment: 2,
			expected:  7,
		},
		{
			name:      "Add mob death to negative count (should not happen, but test anyway)",
			initial:   -2,
			increment: 2,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalDeaths: tt.initial,
			}
			for i := 0; i < tt.increment; i++ {
				kd.AddMobDeath()
			}
			assert.Equal(t, tt.expected, kd.TotalDeaths)
		})
	}
}
func TestKDStats_AddPvpDeath(t *testing.T) {
	tests := []struct {
		name      string
		initial   int
		increment int
		expected  int
	}{
		{
			name:      "Add single PvP death to zero",
			initial:   0,
			increment: 1,
			expected:  1,
		},
		{
			name:      "Add multiple PvP deaths to zero",
			initial:   0,
			increment: 4,
			expected:  4,
		},
		{
			name:      "Add PvP death to existing positive count",
			initial:   6,
			increment: 2,
			expected:  8,
		},
		{
			name:      "Add PvP death to negative count (should not happen, but test anyway)",
			initial:   -3,
			increment: 3,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kd := &KDStats{
				TotalPvpDeaths: tt.initial,
			}
			for i := 0; i < tt.increment; i++ {
				kd.AddPvpDeath()
			}
			assert.Equal(t, tt.expected, kd.TotalPvpDeaths)
		})
	}
}
