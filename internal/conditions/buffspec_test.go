package conditions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
)

func TestBuffSpec_GetValue(t *testing.T) {
	tests := []struct {
		name        string
		spec        BuffSpec
		expectedVal int
	}{
		{
			name: "No statmods, no flags, no triggercount",
			spec: BuffSpec{
				RoundInterval: 5,
			},
			expectedVal: 0,
		},
		{
			name: "Statmods positive and negative, no flags, no triggercount",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{"str": 3, "dex": -2},
				RoundInterval: 5,
			},
			expectedVal: 5, // |3| + |−2| = 5, freqVal = 0, flags = 0
		},
		{
			name: "Statmods, flags, no triggercount",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{"str": 2},
				Flags:         []Flag{NoCombat, Hidden},
				RoundInterval: 3,
			},
			expectedVal: 2 + (5 - 3) + 2*5, // 2 + 2 + 10 = 14
		},
		{
			name: "Statmods, flags, triggercount > 0",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{"str": 1, "dex": 2},
				Flags:         []Flag{NoCombat},
				RoundInterval: 2,
				TriggerCount:  3,
			},
			expectedVal: (1 + 2 + (5 - 2) + 1*5) * 3, // (3 + 3 + 5) * 3 = 11 * 3 = 33
		},
		{
			name: "Negative RoundInterval",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{"str": 2},
				Flags:         []Flag{},
				RoundInterval: 10,
			},
			expectedVal: 2, // freqVal = 5-10 = -5 -> 0, so 2+0+0=2
		},
		{
			name: "Zero RoundInterval",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{},
				Flags:         []Flag{NoCombat},
				RoundInterval: 0,
			},
			expectedVal: 0 + 5 + 5, // freqVal = 5-0=5, flags=1*5=5, total=10
		},
		{
			name: "TriggerCount is 1 (should not multiply)",
			spec: BuffSpec{
				StatMods:      statmods.StatMods{"str": 2},
				Flags:         []Flag{},
				RoundInterval: 4,
				TriggerCount:  1,
			},
			expectedVal: (2 + (5 - 4) + 0) * 1, // (2+1+0)*1=3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.spec.GetValue()
			assert.Equal(t, tt.expectedVal, val)
		})
	}
}
func TestBuffSpec_Listed(t *testing.T) {
	cases := []struct {
		name string
		spec BuffSpec
		want bool
	}{
		{"plain", BuffSpec{Name: "Stoneskin"}, true},
		{"hidden", BuffSpec{Name: "Hidden", Flags: []Flag{Hidden}}, false},
		{"secret", BuffSpec{Name: "Respawn Grace", Secret: true}, false},
		{"hidden and secret", BuffSpec{Name: "Both", Secret: true, Flags: []Flag{Hidden}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.spec.Listed())
		})
	}
}
func TestGetBuffSpec(t *testing.T) {
	// Save and restore original buffs map
	origBuffs := buffs
	defer func() { buffs = origBuffs }()

	// Setup test buffs
	buffs = map[int]*BuffSpec{
		1: {BuffId: 1, Name: "Test Buff 1"},
		2: {BuffId: 2, Name: "Test Buff 2"},
	}

	tests := []struct {
		name      string
		inputId   int
		wantBuff  *BuffSpec
		wantFound bool
	}{
		{
			name:      "Existing positive buffId",
			inputId:   1,
			wantBuff:  &BuffSpec{BuffId: 1, Name: "Test Buff 1"},
			wantFound: true,
		},
		{
			name:      "Existing negative buffId (should convert to positive)",
			inputId:   -2,
			wantBuff:  &BuffSpec{BuffId: 2, Name: "Test Buff 2"},
			wantFound: true,
		},
		{
			name:      "Non-existing buffId",
			inputId:   99,
			wantBuff:  nil,
			wantFound: false,
		},
		{
			name:      "Non-existing negative buffId",
			inputId:   -99,
			wantBuff:  nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBuffSpec(tt.inputId)
			if tt.wantFound {
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantBuff.BuffId, got.BuffId)
				assert.Equal(t, tt.wantBuff.Name, got.Name)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
func TestGetAllBuffIds(t *testing.T) {
	origBuffs := buffs
	defer func() { buffs = origBuffs }()

	tests := []struct {
		name       string
		setupBuffs map[int]*BuffSpec
		wantIds    []int
	}{
		{
			name:       "No buffs",
			setupBuffs: map[int]*BuffSpec{},
			wantIds:    []int{},
		},
		{
			name: "Single buff",
			setupBuffs: map[int]*BuffSpec{
				10: {BuffId: 10, Name: "Solo Buff"},
			},
			wantIds: []int{10},
		},
		{
			name: "Multiple buffs",
			setupBuffs: map[int]*BuffSpec{
				1: {BuffId: 1, Name: "Buff One"},
				2: {BuffId: 2, Name: "Buff Two"},
				3: {BuffId: 3, Name: "Buff Three"},
			},
			wantIds: []int{1, 2, 3},
		},
		{
			name: "Buffs with non-sequential IDs",
			setupBuffs: map[int]*BuffSpec{
				100: {BuffId: 100, Name: "Buff 100"},
				5:   {BuffId: 5, Name: "Buff 5"},
				42:  {BuffId: 42, Name: "Buff 42"},
			},
			wantIds: []int{100, 5, 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffs = tt.setupBuffs
			got := GetAllBuffIds()
			assert.ElementsMatch(t, tt.wantIds, got)
		})
	}
}
func TestSearchBuffs(t *testing.T) {
	origBuffs := buffs
	defer func() { buffs = origBuffs }()

	buffs = map[int]*BuffSpec{
		1: {BuffId: 1, Name: "Fast Healing", Description: "Increases health recovery"},
		2: {BuffId: 2, Name: "Poison", Description: "Deals damage over time"},
		3: {BuffId: 3, Name: "Night Vision", Description: "See in the dark"},
		4: {BuffId: 4, Name: "Hidden", Description: "Invisible to others"},
		5: {BuffId: 5, Name: "Hydrated", Description: "Reduces thirst"},
	}

	tests := []struct {
		name       string
		searchTerm string
		wantIds    []int
	}{
		{
			name:       "Exact match on name",
			searchTerm: "Poison",
			wantIds:    []int{2},
		},
		{
			name:       "Partial match on name (case-insensitive)",
			searchTerm: "heal",
			wantIds:    []int{1},
		},
		{
			name:       "Partial match on description",
			searchTerm: "damage",
			wantIds:    []int{2},
		},
		{
			name:       "Match multiple buffs by description",
			searchTerm: "see",
			wantIds:    []int{3},
		},
		{
			name:       "Match multiple buffs by name",
			searchTerm: "hid",
			wantIds:    []int{4},
		},
		{
			name:       "No matches",
			searchTerm: "fire",
			wantIds:    []int{},
		},
		{
			name:       "Whitespace and case-insensitive",
			searchTerm: "   nIgHt   ",
			wantIds:    []int{3},
		},
		{
			name:       "Empty search term returns all buffs",
			searchTerm: "",
			wantIds:    []int{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SearchBuffs(tt.searchTerm)
			assert.ElementsMatch(t, tt.wantIds, got)
		})
	}
}
func TestBuffSpec_Id(t *testing.T) {
	tests := []struct {
		name   string
		spec   BuffSpec
		wantId int
	}{
		{
			name:   "Positive BuffId",
			spec:   BuffSpec{BuffId: 42},
			wantId: 42,
		},
		{
			name:   "Zero BuffId",
			spec:   BuffSpec{BuffId: 0},
			wantId: 0,
		},
		{
			name:   "Negative BuffId",
			spec:   BuffSpec{BuffId: -7},
			wantId: -7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.Id()
			assert.Equal(t, tt.wantId, got)
		})
	}
}
func TestBuffSpec_Filename(t *testing.T) {
	tests := []struct {
		name     string
		spec     BuffSpec
		expected string
	}{
		{
			name:     "Simple name",
			spec:     BuffSpec{BuffId: 1, Name: "Fast Healing"},
			expected: "1-fast_healing.yaml",
		},
		{
			name:     "Name with special characters",
			spec:     BuffSpec{BuffId: 42, Name: "Poison!@#"},
			expected: "42-poison___.yaml",
		},
		{
			name:     "Name with spaces and mixed case",
			spec:     BuffSpec{BuffId: 7, Name: "Night Vision"},
			expected: "7-night_vision.yaml",
		},
		{
			name:     "Name with underscores and dashes",
			spec:     BuffSpec{BuffId: 100, Name: "Hydrated_buff-test"},
			expected: "100-hydrated_buff_test.yaml",
		},
		{
			name:     "Empty name",
			spec:     BuffSpec{BuffId: 5, Name: ""},
			expected: "5-.yaml",
		},
		{
			name:     "Name with leading/trailing spaces",
			spec:     BuffSpec{BuffId: 8, Name: "  Hidden Buff  "},
			expected: "8-__hidden_buff__.yaml",
		},
		{
			name:     "Name with multiple spaces",
			spec:     BuffSpec{BuffId: 9, Name: "Buff    With   Spaces"},
			expected: "9-buff____with___spaces.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.Filename()
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestBuffSpec_Filepath(t *testing.T) {
	tests := []struct {
		name     string
		spec     BuffSpec
		expected string
	}{
		{
			name:     "Simple name",
			spec:     BuffSpec{BuffId: 1, Name: "Fast Healing"},
			expected: "1-fast_healing.yaml",
		},
		{
			name:     "Name with special characters",
			spec:     BuffSpec{BuffId: 42, Name: "Poison!@#"},
			expected: "42-poison___.yaml",
		},
		{
			name:     "Name with spaces and mixed case",
			spec:     BuffSpec{BuffId: 7, Name: "Night Vision"},
			expected: "7-night_vision.yaml",
		},
		{
			name:     "Name with underscores and dashes",
			spec:     BuffSpec{BuffId: 100, Name: "Hydrated_buff-test"},
			expected: "100-hydrated_buff_test.yaml",
		},
		{
			name:     "Empty name",
			spec:     BuffSpec{BuffId: 5, Name: ""},
			expected: "5-.yaml",
		},
		{
			name:     "Name with leading/trailing spaces",
			spec:     BuffSpec{BuffId: 8, Name: "  Hidden Buff  "},
			expected: "8-__hidden_buff__.yaml",
		},
		{
			name:     "Name with multiple spaces",
			spec:     BuffSpec{BuffId: 9, Name: "Buff    With   Spaces"},
			expected: "9-buff____with___spaces.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.Filepath()
			assert.Equal(t, tt.expected, got)
		})
	}
}
