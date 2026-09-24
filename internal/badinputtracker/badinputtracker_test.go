package badinputtracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackBadCommand(t *testing.T) {
	tests := []struct {
		name     string
		commands [][2]string
		expected map[string]int
	}{
		{
			name: "single command",
			commands: [][2]string{
				{"foo", "bar"},
			},
			expected: map[string]int{
				"foo bar": 1,
			},
		},
		{
			name: "multiple commands same cmd/rest",
			commands: [][2]string{
				{"foo", "bar"},
				{"foo", "bar"},
			},
			expected: map[string]int{
				"foo bar": 2,
			},
		},
		{
			name: "multiple commands different rest",
			commands: [][2]string{
				{"foo", "bar"},
				{"foo", "baz"},
			},
			expected: map[string]int{
				"foo bar": 1,
				"foo baz": 1,
			},
		},
		{
			name: "multiple commands different cmd",
			commands: [][2]string{
				{"foo", "bar"},
				{"baz", "qux"},
			},
			expected: map[string]int{
				"foo bar": 1,
				"baz qux": 1,
			},
		},
		{
			name: "complex mix",
			commands: [][2]string{
				{"foo", "bar"},
				{"foo", "bar"},
				{"foo", "baz"},
				{"baz", "qux"},
				{"baz", "qux"},
				{"baz", "qux"},
			},
			expected: map[string]int{
				"foo bar": 2,
				"foo baz": 1,
				"baz qux": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Clear()
			for _, pair := range tt.commands {
				TrackBadCommand(pair[0], pair[1])
			}
			got := GetBadCommands()
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestGetBadCommands(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected map[string]int
	}{
		{
			name: "no commands tracked",
			setup: func() {
				Clear()
			},
			expected: map[string]int{},
		},
		{
			name: "single command tracked",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
			},
			expected: map[string]int{
				"foo bar": 1,
			},
		},
		{
			name: "multiple commands tracked",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "baz")
				TrackBadCommand("baz", "qux")
			},
			expected: map[string]int{
				"foo bar": 1,
				"foo baz": 1,
				"baz qux": 1,
			},
		},
		{
			name: "increment counts for same command/rest",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "bar")
			},
			expected: map[string]int{
				"foo bar": 3,
			},
		},
		{
			name: "mixed increment and new commands",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "baz")
				TrackBadCommand("baz", "qux")
				TrackBadCommand("baz", "qux")
			},
			expected: map[string]int{
				"foo bar": 2,
				"foo baz": 1,
				"baz qux": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := GetBadCommands()
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestClear(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		expectLen int
	}{
		{
			name: "clear after tracking commands",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				TrackBadCommand("baz", "qux")
			},
			expectLen: 0,
		},
		{
			name: "clear with no commands tracked",
			setup: func() {
				Clear()
			},
			expectLen: 0,
		},
		{
			name: "clear after multiple clears",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				Clear()
				Clear()
			},
			expectLen: 0,
		},
		{
			name: "clear after incrementing same command",
			setup: func() {
				Clear()
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "bar")
				TrackBadCommand("foo", "bar")
			},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			Clear()
			got := GetBadCommands()
			assert.Equal(t, tt.expectLen, len(got))
			assert.Equal(t, map[string]int{}, got)
		})
	}
}
