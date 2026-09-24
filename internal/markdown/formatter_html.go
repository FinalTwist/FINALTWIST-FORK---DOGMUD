package markdown

import (
	"strconv"
	"strings"
)

//
// Formats into HTML tags
//

type HTML struct{}

func (HTML) Document(contents string, depth int) string {
	return strings.TrimLeft(contents, "\n ")
}
func (HTML) Paragraph(contents string, depth int) string      { return "\n<p>\n" + contents + "\n</p>" }
func (HTML) HorizontalLine(contents string, depth int) string { return "\n<hr />\n" }
func (HTML) HardBreak(contents string, depth int) string      { return "\n<br />\n" }
func (HTML) Heading(contents string, depth int) string {
	return "\n<h" + strconv.Itoa(depth) + ">" + contents + "</h" + strconv.Itoa(depth) + ">"
}
func (HTML) List(contents string, depth int) string {
	return "\n" + strings.Repeat("\t", depth) + "<ul>" + contents + "\n" + strings.Repeat("\t", depth) + "</ul>"
}
func (HTML) ListItem(contents string, depth int) string {
	return "\n" + strings.Repeat("\t", depth) + "<li>" + contents + "\n" + strings.Repeat("\t", depth) + "</li>"
}
func (HTML) Text(contents string, depth int) string {
	return contents
}
func (HTML) Strong(contents string, depth int) string   { return "<strong>" + contents + "</strong>" }
func (HTML) Emphasis(contents string, depth int) string { return "<em>" + contents + "</em>" }
func (HTML) Special(contents string, depth int) string {
	return "<span data-special=\"" + strconv.Itoa(depth) + "\">" + contents + "</span>"
}
