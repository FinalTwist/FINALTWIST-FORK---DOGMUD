package markdown

import "strings"

//
// Formats into a clean version of supported markdown
//

type ReMarkdown struct{}

func (ReMarkdown) Document(contents string, depth int) string {
	return strings.TrimLeft(contents, "\n ")
}
func (ReMarkdown) Paragraph(contents string, depth int) string      { return "\n\n" + contents }
func (ReMarkdown) HardBreak(contents string, depth int) string      { return "\n" }
func (ReMarkdown) HorizontalLine(contents string, depth int) string { return "\n\n---" }
func (ReMarkdown) Heading(contents string, depth int) string {
	return "\n\n" + strings.Repeat(`#`, depth) + " " + contents
}
func (ReMarkdown) List(contents string, depth int) string {
	if depth == 0 {
		return "\n\n" + contents
	}
	return strings.Repeat(` `, depth) + contents
}
func (ReMarkdown) ListItem(contents string, depth int) string {
	return "\n" + strings.Repeat(` `, depth) + "- " + contents
}
func (ReMarkdown) Text(contents string, depth int) string {
	//return strings.TrimSpace(contents)
	return contents
}
func (ReMarkdown) Strong(contents string, depth int) string   { return "**" + contents + "**" }
func (ReMarkdown) Emphasis(contents string, depth int) string { return "*" + contents + "*" }
func (ReMarkdown) Special(contents string, depth int) string {
	return strings.Repeat(`~`, depth) + contents + strings.Repeat(`~`, depth)
}
