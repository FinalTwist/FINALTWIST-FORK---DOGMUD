package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobMasteries_GetAllTame(t *testing.T) {
	tests := []struct {
		name     string
		tame     map[int]int
		expected map[int]int
	}{
		{
			name:     "nil Tame map",
			tame:     nil,
			expected: map[int]int{},
		},
		{
			name:     "empty Tame map",
			tame:     map[int]int{},
			expected: map[int]int{},
		},
		{
			name:     "single entry",
			tame:     map[int]int{1: 10},
			expected: map[int]int{1: 10},
		},
		{
			name:     "multiple entries",
			tame:     map[int]int{1: 10, 2: 20, 3: 30},
			expected: map[int]int{1: 10, 2: 20, 3: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MobMasteries{Tame: tt.tame}
			got := m.GetAllTame()
			assert.Equal(t, tt.expected, got)

			// Ensure returned map is a copy, not the original
			if tt.tame != nil {
				if len(got) > 0 {
					got[999] = 999
					assert.NotEqual(t, got, m.Tame)
				}
			}
		})
	}
}
func TestMobMasteries_GetTame(t *testing.T) {
	tests := []struct {
		name     string
		tame     map[int]int
		mobId    int
		expected int
	}{
		{
			name:     "nil Tame map",
			tame:     nil,
			mobId:    1,
			expected: 0,
		},
		{
			name:     "empty Tame map",
			tame:     map[int]int{},
			mobId:    2,
			expected: 0,
		},
		{
			name:     "mobId not present",
			tame:     map[int]int{1: 10, 2: 20},
			mobId:    3,
			expected: 0,
		},
		{
			name:     "mobId present",
			tame:     map[int]int{1: 10, 2: 20, 3: 30},
			mobId:    2,
			expected: 20,
		},
		{
			name:     "mobId present with zero value",
			tame:     map[int]int{4: 0},
			mobId:    4,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MobMasteries{Tame: tt.tame}
			got := m.GetTame(tt.mobId)
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestMobMasteries_SetTame(t *testing.T) {
	tests := []struct {
		name         string
		initialTame  map[int]int
		mobId        int
		amt          int
		expectedTame map[int]int
	}{
		{
			name:         "nil Tame map, set new mobId",
			initialTame:  nil,
			mobId:        1,
			amt:          10,
			expectedTame: map[int]int{1: 10},
		},
		{
			name:         "empty Tame map, set new mobId",
			initialTame:  map[int]int{},
			mobId:        2,
			amt:          20,
			expectedTame: map[int]int{2: 20},
		},
		{
			name:         "set new mobId in non-empty map",
			initialTame:  map[int]int{1: 5},
			mobId:        3,
			amt:          30,
			expectedTame: map[int]int{1: 5, 3: 30},
		},
		{
			name:         "overwrite existing mobId",
			initialTame:  map[int]int{4: 40, 5: 50},
			mobId:        4,
			amt:          99,
			expectedTame: map[int]int{4: 99, 5: 50},
		},
		{
			name:         "set mobId to zero",
			initialTame:  map[int]int{6: 60},
			mobId:        6,
			amt:          0,
			expectedTame: map[int]int{6: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MobMasteries{Tame: tt.initialTame}
			m.SetTame(tt.mobId, tt.amt)
			assert.Equal(t, tt.expectedTame, m.Tame)
		})
	}
}
