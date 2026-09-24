package markdown

import (
	"strconv"
	"strings"
)

// Formats into HTML tags
//
// Expected ansitags color aliases:
// md
// md-p
// md-h1-prefix
// md-h1, md-h2, md-h3 etc.
// md-li
// md-bold
// md-em
// md-sp1, md-sp2, md-sp3, etc.
// md-tbl-hdr
// md-tbl-row
// md-tbl-cell
// md-hr1
// md-hr2
//
// All have bg classes named the same with "-bg" at the end.
// Example: md-li-bg

var dividers = map[string]string{
	"---": "\n<ansi fg=\"md-hr1\" bg=\"md-hr1-bg\">--------------------------------------------------------------------------------</ansi>",
	"===": "\n<ansi fg=\"md-hr2\" bg=\"md-hr2-bg\">================================================================================</ansi>",
	":::": "\n<ansi fg=\"6\">  .--.      .-'.      .--.      .--.      .--.      .--.      .`-.      .--.    \n" +
		"<ansi fg=\"187\">:::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">::::::::.</ansi>\\<ansi fg=\"187\">:::</ansi>\n" +
		"'      `--'      `.-'      `--'      `--'      `--'      `-.'      `--'      `--</ansi>",
}

type ANSITags struct{}

func (ANSITags) Document(contents string, depth int) string {
	return "<ansi fg=\"md\" bg=\"md-bg\">" + strings.TrimLeft(contents, "\n ") + "</ansi>"
}
func (ANSITags) Paragraph(contents string, depth int) string {
	return "\n\n<ansi fg=\"md-p\" bg=\"md-p-bg\">" + contents + "</ansi>"
}
func (ANSITags) HorizontalLine(contents string, depth int) string {
	return "\n" + dividers[contents]
}
func (ANSITags) HardBreak(contents string, depth int) string { return "\n" }
func (ANSITags) Heading(contents string, depth int) string {
	if depth == 1 {
		contents = "<ansi fg=\"md-h1-prefix\" bg=\"md-h1-prefix-bg\">.:</ansi> " + contents
	}
	return "\n\n<ansi fg=\"md-h" + strconv.Itoa(depth) + "\" bg=\"md-h" + strconv.Itoa(depth) + "-bg\">" + contents + "</ansi>"
}
func (ANSITags) List(contents string, depth int) string {
	if depth == 0 {
		return "\n\n" + contents
	}
	return strings.Repeat(` `, depth) + contents
}
func (ANSITags) ListItem(contents string, depth int) string {
	return "\n" + strings.Repeat(` `, depth) + "<ansi fg=\"md-li\" bg=\"md-li-bg\">- " + contents + "</ansi>"
}
func (ANSITags) Text(contents string, depth int) string {
	//return strings.TrimSpace(contents)
	return contents
}
func (ANSITags) Strong(contents string, depth int) string {
	return "<ansi fg=\"md-bold\" bg=\"md-bold-bg\">" + contents + "</ansi>"
}
func (ANSITags) Emphasis(contents string, depth int) string {
	return "<ansi fg=\"md-em\" bg=\"md-em-bg\">" + contents + "</ansi>"
}
func (ANSITags) Special(contents string, depth int) string {
	return "<ansi fg=\"md-sp" + strconv.Itoa(depth) + "\" bg=\"md-sp" + strconv.Itoa(depth) + "-bg\">" + contents + "</ansi>"
}
