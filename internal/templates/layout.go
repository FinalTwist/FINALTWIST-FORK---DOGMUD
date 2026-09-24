package templates

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/term"
)

// DynamicList takes a slice of NameDescription items and formats them into a string
// NOTE: This is a first step to moving dynamic lists into a common function.
func DynamicList(itmNames []NameDescription, colWidth, sw, numWidth, longestName int) string {
	if len(itmNames) == 0 {
		return `<ansi fg="202">No items to display.</ansi>`
	}

	strOut := ``
	totalLen := 0
	for idx, itm := range itmNames {

		if totalLen+colWidth > sw {
			strOut += term.CRLFStr
			totalLen = 0
		}

		numStr := strconv.Itoa(idx + 1)

		strOut += ` `

		if itm.Marked {
			strOut += `<ansi fg="white-bold" bg="059">`
		}

		strOut += strings.Repeat(` `, numWidth-len(numStr)) + fmt.Sprintf(`<ansi fg="red-bold">%s</ansi>`, strconv.Itoa(idx+1)) + `. ` +
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, itm.Name) + strings.Repeat(` `, longestName-len(itm.Name))

		if itm.Marked {
			strOut += `</ansi>`
		}

		strOut += ` `

		totalLen += colWidth

	}
	return strOut
}
