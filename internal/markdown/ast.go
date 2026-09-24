package markdown

import (
	"fmt"
)

// NodeType identifies the kind of AST node.
type NodeType string

const (
	DocumentNode       NodeType = "Document"
	HeadingNode        NodeType = "Heading"
	ParagraphNode      NodeType = "Paragraph"
	HorizontalLineNode NodeType = "HorizontalLine"
	HardBreakNode      NodeType = "HardBreak"
	ListNode           NodeType = "List"
	ListItemNode       NodeType = "ListItem"
	TextNode           NodeType = "Text"
	StrongNode         NodeType = "Strong"
	EmphasisNode       NodeType = "Emphasis"
	SpecialNode        NodeType = "Special"
)

var (
	activeFormatter Formatter = ReMarkdown{}
)

func SetFormatter(newFormatter Formatter) {
	activeFormatter = newFormatter
}

// Node is an element in the AST.
type Node interface {
	Type() NodeType
	Children() []Node
	String(int) string
}

// baseNode provides common fields.
type baseNode struct {
	nodeType     NodeType
	nodeChildren []Node
	level        int
	content      string
}

func (n *baseNode) Type() NodeType   { return n.nodeType }
func (n *baseNode) Children() []Node { return n.nodeChildren }
func (n *baseNode) String(depth int) string {
	ret := ``
	for _, c := range n.Children() {
		if n.Type() == ListNode {
			ret += c.String(depth - 1)
		} else {
			ret += c.String(depth + 1)
		}

	}

	switch n.Type() {
	case DocumentNode:
		return activeFormatter.Document(ret, depth)
	case HeadingNode:
		return activeFormatter.Heading(ret, n.level)
	case ParagraphNode:
		return activeFormatter.Paragraph(ret, depth)
	case HorizontalLineNode:
		return activeFormatter.HorizontalLine(n.content, depth)
	case HardBreakNode:
		return activeFormatter.HardBreak(ret, depth)
	case ListNode:
		return activeFormatter.List(ret, depth)
	case ListItemNode:
		return activeFormatter.ListItem(ret, depth)
	case TextNode:
		return activeFormatter.Text(n.content+ret, depth)
	case StrongNode:
		return activeFormatter.Strong(ret, depth)
	case EmphasisNode:
		return activeFormatter.Emphasis(ret, depth)
	case SpecialNode:
		return activeFormatter.Special(ret, n.level)
	default:
		return fmt.Sprintf(`[INVALID Node: type=%s depth=%d text=%s]`, n.Type(), depth, n.content+"/"+ret)
	}
}
