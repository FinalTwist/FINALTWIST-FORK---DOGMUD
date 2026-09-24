package templates_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/stretchr/testify/assert"
)

func TestDynamicList(t *testing.T) {
	tests := []struct {
		name           string
		itmNames       []templates.NameDescription
		colWidth       int
		sw             int
		numWidth       int
		longestName    int
		expectedOutput string
	}{
		{
			name:           "Empty list",
			itmNames:       []templates.NameDescription{},
			colWidth:       20,
			sw:             80,
			numWidth:       2,
			longestName:    10,
			expectedOutput: `<ansi fg="202">No items to display.</ansi>`,
		},
		{
			name: "Single item",
			itmNames: []templates.NameDescription{
				{Name: "Item1", Marked: false},
			},
			colWidth:       20,
			sw:             80,
			numWidth:       2,
			longestName:    10,
			expectedOutput: `  <ansi fg="red-bold">1</ansi>. <ansi fg="yellow-bold">Item1</ansi>      `,
		},
		{
			name: "Multiple items with wrapping",
			itmNames: []templates.NameDescription{
				{Name: "Item1", Marked: false},
				{Name: "Item2", Marked: true},
				{Name: "Item3", Marked: false},
			},
			colWidth:       20,
			sw:             40,
			numWidth:       2,
			longestName:    10,
			expectedOutput: `  <ansi fg="red-bold">1</ansi>. <ansi fg="yellow-bold">Item1</ansi>       <ansi fg="white-bold" bg="059"> <ansi fg="red-bold">2</ansi>. <ansi fg="yellow-bold">Item2</ansi>     </ansi> ` + term.CRLFStr + `  <ansi fg="red-bold">3</ansi>. <ansi fg="yellow-bold">Item3</ansi>      `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := templates.DynamicList(tt.itmNames, tt.colWidth, tt.sw, tt.numWidth, tt.longestName)
			assert.Equal(t, tt.expectedOutput, output, "DynamicList() output mismatch")
		})
	}
}
