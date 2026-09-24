package markdown

import (
	"fmt"
	"strings"
	"testing"
)

// Output oriented tests, for development

// This test should be ran just as a way to verify content visually.
// Prefixed with "x" when not being used.
func xTest_Printing(t *testing.T) {
	src := `# This is a **HEADING**


This is a *NEW PARAGRAPH*.
Paragraph a preceded by two new lines.  
This is a **line break**.  
Line breaks happen when the previous line ended.  
with two spaces: "  ".  
They are preceded by only a single new line.

This is another paragraph.
 
- item one
  - item one >> sub one
  - item one >> sub two
    - item one >> sub two >> sub one
- **bold** item two

## This is a ~~SUB HEADING~~

        That ~is~ all.

Some text
---
===
:::
Some more text

        That ~is~ all.
`

	parser := NewParser(src)
	ast := parser.Parse()

	fmt.Println("------------------- DUMP -------------------")
	fmt.Println()
	Dump(ast, 3)
	fmt.Println()
	fmt.Println("----------------- REFORMAT -----------------")
	fmt.Println()
	fmt.Println(ast.String(0))
	fmt.Println()

	fmt.Println("------------------- HTML -------------------")
	fmt.Println()
	SetFormatter(HTML{})
	fmt.Println(ast.String(0))
	fmt.Println()
	fmt.Println("------------------- ANSI -------------------")
	fmt.Println()
	SetFormatter(ANSITags{})
	fmt.Println(ast.String(0))
	fmt.Println()

	fmt.Println("------------------- DONE -------------------")
}

// Useful for printing out stuff
func Dump(n Node, indentSpaces int, currentIndent ...int) {

	if len(currentIndent) == 0 {
		currentIndent = []int{0}
	}

	if indentSpaces == 0 {
		indentSpaces = 1
	}

	fmt.Printf("%s- %s", strings.Repeat(" ", currentIndent[0]*indentSpaces), n.Type())

	bNode := n.(*baseNode)
	switch n.Type() {
	case HeadingNode:
		fmt.Printf(" (level=%d)\n", bNode.level)
	case TextNode:
		fmt.Printf(": %q\n", bNode.content)
	default:
		fmt.Printf(" (%d)", len(n.Children()))
		fmt.Println()
	}
	for _, c := range n.Children() {
		Dump(c, indentSpaces, currentIndent[0]+1)
	}
}
