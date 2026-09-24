package markdown

type Formatter interface {
	Document(string, int) string
	Paragraph(string, int) string
	HorizontalLine(string, int) string
	HardBreak(string, int) string
	Heading(string, int) string
	List(string, int) string
	ListItem(string, int) string
	Text(string, int) string
	Strong(string, int) string
	Emphasis(string, int) string
	Special(string, int) string
}
