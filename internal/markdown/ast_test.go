package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	// Use ReMarkdown for consistent output
	SetFormatter(ReMarkdown{})
}

func TestParser(t *testing.T) {
	t.Run("Heading", func(t *testing.T) {
		doc := NewParser("# Hello").Parse()
		require.Equal(t, DocumentNode, doc.Type())
		require.Len(t, doc.Children(), 1)
		heading := doc.Children()[0]
		require.Equal(t, HeadingNode, heading.Type())
		require.Equal(t, "# Hello", doc.String(0))
	})

	t.Run("Paragraph", func(t *testing.T) {
		doc := NewParser("Hello world").Parse()
		require.Len(t, doc.Children(), 1)
		para := doc.Children()[0]
		require.Equal(t, ParagraphNode, para.Type())
		require.Equal(t, "Hello world", doc.String(0))
	})

	t.Run("HorizontalLine", func(t *testing.T) {
		doc := NewParser("---").Parse()
		require.Len(t, doc.Children(), 1)
		sep := doc.Children()[0]
		require.Equal(t, HorizontalLineNode, sep.Type())
		require.Equal(t, "---", doc.String(0))
	})

	t.Run("HardBreak", func(t *testing.T) {
		doc := NewParser("Line one  \nLine two").Parse()
		require.Equal(t, "Line one\nLine two", doc.String(0))
	})

	t.Run("List", func(t *testing.T) {
		doc := NewParser("- one\n- two").Parse()
		require.Len(t, doc.Children(), 1)
		list := doc.Children()[0]
		require.Equal(t, ListNode, list.Type())
		require.Equal(t, "- one\n- two", doc.String(0))
	})

	t.Run("InlineFormatting", func(t *testing.T) {
		t.Run("Emphasis", func(t *testing.T) {
			doc := NewParser("*em*").Parse()
			require.Len(t, doc.Children(), 1)
			para := doc.Children()[0].(*baseNode)
			require.Equal(t, ParagraphNode, para.Type())
			children := para.Children()
			require.Len(t, children, 1)
			require.Equal(t, EmphasisNode, children[0].Type())
			require.Equal(t, "*em*", doc.String(0))
		})

		t.Run("Strong", func(t *testing.T) {
			doc := NewParser("**bold**").Parse()
			require.Len(t, doc.Children(), 1)
			para := doc.Children()[0].(*baseNode)
			require.Equal(t, ParagraphNode, para.Type())
			children := para.Children()
			require.Len(t, children, 1)
			require.Equal(t, StrongNode, children[0].Type())
			require.Equal(t, "**bold**", doc.String(0))
		})

		t.Run("Special", func(t *testing.T) {
			doc := NewParser("~sp~").Parse()
			require.Len(t, doc.Children(), 1)
			para := doc.Children()[0].(*baseNode)
			require.Equal(t, ParagraphNode, para.Type())
			children := para.Children()
			require.Len(t, children, 1)
			require.Equal(t, SpecialNode, children[0].Type())
			require.Equal(t, "~sp~", doc.String(0))
		})
	})

	t.Run("InvalidNodeType", func(t *testing.T) {
		n := &baseNode{nodeType: NodeType("Unknown"), content: "xyz"}
		str := n.String(5)
		require.Contains(t, str, "INVALID Node")
		require.Contains(t, str, "xyz")
	})
}
