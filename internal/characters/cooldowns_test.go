package characters

import (
	"testing"

	"maps"

	"github.com/stretchr/testify/assert"
)

func TestCooldowns_RoundTick(t *testing.T) {
	tests := []struct {
		name     string
		input    Cooldowns
		expected Cooldowns
	}{
		{
			name:     "Single cooldown decrements",
			input:    Cooldowns{"test": 3},
			expected: Cooldowns{"test": 2},
		},
		{
			name:     "Multiple cooldowns decrement",
			input:    Cooldowns{"a": 5, "b": 2, "c": 0},
			expected: Cooldowns{"a": 4, "b": 1, "c": -1},
		},
		{
			name:     "Empty cooldowns map",
			input:    Cooldowns{},
			expected: Cooldowns{},
		},
		{
			name:     "Negative values decrement",
			input:    Cooldowns{"neg": -2},
			expected: Cooldowns{"neg": -3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := Cooldowns{}
			maps.Copy(cd, tt.input)
			cd.RoundTick()
			assert.Equal(t, tt.expected, cd)
		})
	}
}
func TestCooldowns_Prune(t *testing.T) {
	tests := []struct {
		name     string
		input    Cooldowns
		expected Cooldowns
	}{
		{
			name:     "Removes zero value",
			input:    Cooldowns{"a": 0, "b": 2, "c": 1},
			expected: Cooldowns{"b": 2, "c": 1},
		},
		{
			name:     "Removes negative value",
			input:    Cooldowns{"x": -1, "y": 3},
			expected: Cooldowns{"y": 3},
		},
		{
			name:     "Removes multiple zero and negative values",
			input:    Cooldowns{"a": 0, "b": -2, "c": 5},
			expected: Cooldowns{"c": 5},
		},
		{
			name:     "Keeps all positive values",
			input:    Cooldowns{"a": 1, "b": 2},
			expected: Cooldowns{"a": 1, "b": 2},
		},
		{
			name:     "Empty map remains empty",
			input:    Cooldowns{},
			expected: Cooldowns{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := Cooldowns{}
			maps.Copy(cd, tt.input)
			cd.Prune()
			assert.Equal(t, tt.expected, cd)
		})
	}
}
