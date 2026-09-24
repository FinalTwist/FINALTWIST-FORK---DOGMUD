package markdown

import (
	"regexp"
	"strings"
)

const (
	lineBreakString = "  "
)

var tableSep = regexp.MustCompile(`^\s*\|?[-: ]+\|?([-: ]*\|?)*\s*$`)

type Parser struct {
	lines []string
	pos   int
}

func NewParser(input string) *Parser {
	return &Parser{
		lines: strings.Split(input, "\n"),
	}
}

func (p *Parser) Parse() Node {
	doc := &baseNode{nodeType: DocumentNode}
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]

		switch {
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "==="), strings.HasPrefix(line, ":::"):
			doc.nodeChildren = append(doc.nodeChildren, p.parseHorizontalLine())
		case strings.HasPrefix(line, "#"):
			doc.nodeChildren = append(doc.nodeChildren, p.parseHeading())
		case strings.HasPrefix(strings.TrimSpace(line), "- "):
			// compute leading-space indent
			indent := len(line) - len(strings.TrimLeft(line, " "))
			doc.nodeChildren = append(doc.nodeChildren, p.parseList(indent))
		case strings.TrimSpace(line) == "":
			p.pos++ // skip blank
		default:
			// instead of a single node, grab a slice
			for _, node := range p.parseParagraphNodes() {
				doc.nodeChildren = append(doc.nodeChildren, node)
			}
		}
	}
	return doc
}

func (p *Parser) parseHorizontalLine() *baseNode {
	line := p.lines[p.pos]
	level := 0

	lineType := line[level]
	for level < len(line) && line[level] == lineType {
		level++
	}
	// skip "# " prefix
	content := ""
	if len(line) > level+1 {
		content = line[level+1:]
	}
	p.pos++

	h := &baseNode{
		nodeType:     HorizontalLineNode,
		nodeChildren: p.parseInline(content),
		level:        level,
		content:      strings.Repeat(string(lineType), 3),
	}
	return h
}

func (p *Parser) parseHeading() *baseNode {
	line := p.lines[p.pos]
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	// skip "# " prefix
	content := ""
	if len(line) > level+1 {
		content = line[level+1:]
	}
	p.pos++

	h := &baseNode{
		nodeType:     HeadingNode,
		nodeChildren: p.parseInline(content),
		level:        level,
	}
	return h
}

// parseList now takes the indent level of its bullets
func (p *Parser) parseList(baseIndent int) *baseNode {
	list := &baseNode{nodeType: ListNode}

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		// count leading spaces
		currIndent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)

		// if it’s not a bullet or we’ve de-indented, this list is done
		if !strings.HasPrefix(trimmed, "- ") || currIndent < baseIndent {
			break
		}

		if currIndent > baseIndent {
			// nested list: recurse, attach to last ListItem
			nested := p.parseList(currIndent)
			if len(list.nodeChildren) > 0 {
				lastItem := list.nodeChildren[len(list.nodeChildren)-1].(*baseNode)
				lastItem.nodeChildren = append(lastItem.nodeChildren, nested)
			}
			continue
		}

		// same-level item
		itemText := trimmed[2:] // drop "- "
		item := &baseNode{nodeType: ListItemNode}
		item.nodeChildren = p.parseInline(itemText)
		list.nodeChildren = append(list.nodeChildren, item)
		p.pos++
	}

	return list
}

func (p *Parser) parseParagraphNodes() []Node {
	// 1) collect until blank line
	var lines []string
	for p.pos < len(p.lines) && strings.TrimSpace(p.lines[p.pos]) != "" && !strings.HasPrefix(strings.TrimSpace(p.lines[p.pos]), `---`) {
		lines = append(lines, p.lines[p.pos])
		p.pos++
	}

	var nodes []Node
	start := 0

	// 2) whenever we see a line ending in "  ",
	//    that's a hard break point.
	para := &baseNode{nodeType: ParagraphNode}

	for i, line := range lines {
		if strings.HasSuffix(line, lineBreakString) {
			// lines[start..i] form one paragraph
			seg := append([]string{}, lines[start:i+1]...)
			seg[len(seg)-1] = strings.TrimSuffix(seg[len(seg)-1], lineBreakString)

			// Parse the contents thus far, converting new lines into spaces
			newChildren := p.parseInline(strings.Join(seg, " "))
			// add to paragraph
			para.nodeChildren = append(para.nodeChildren, newChildren...)
			// now add a line break
			para.nodeChildren = append(para.nodeChildren, &baseNode{nodeType: HardBreakNode})

			// now skip ahead
			start = i + 1
			continue
		}
	}

	// 3) whatever remains after the last hard-break
	if start < len(lines) {
		seg := lines[start:]
		newChildren := p.parseInline(strings.Join(seg, " "))
		para.nodeChildren = append(para.nodeChildren, newChildren...)
	}

	nodes = append(nodes, para)
	return nodes
}

func (p *Parser) parseInline(text string) []Node {
	var nodes []Node
	for i := 0; i < len(text); {
		// —— special: ~…~
		if text[i] == '~' {
			start := i
			for i < len(text) && text[i] == '~' {
				i++
			}
			count := i - start
			delim := strings.Repeat("~", count)

			if j := strings.Index(text[i:], delim); j >= 0 {
				inner := text[i : i+j]
				childNodes := p.parseInline(inner)
				n := &baseNode{
					nodeType:     SpecialNode,
					nodeChildren: childNodes,
					level:        count,
				}
				nodes = append(nodes, n)
				i += j + count
				continue
			}

			// no closing run → literal dollars
			nodes = append(nodes, &baseNode{
				nodeType: TextNode,
				content:  text[start:i],
			})
			continue
		}

		// —— strong: **bold**
		if strings.HasPrefix(text[i:], "**") && i+2 < len(text) && text[i+2] != ' ' {
			if j := strings.Index(text[i+2:], "**"); j >= 0 {
				inner := text[i+2 : i+2+j]
				n := &baseNode{nodeType: StrongNode}
				n.nodeChildren = p.parseInline(inner)
				nodes = append(nodes, n)
				i += 4 + j
				continue
			}
		}

		// —— emphasis: *em*
		if text[i] == '*' && i+1 < len(text) && text[i+1] != ' ' {
			if j := strings.Index(text[i+1:], "*"); j >= 0 {
				inner := text[i+1 : i+1+j]
				n := &baseNode{nodeType: EmphasisNode}
				n.nodeChildren = p.parseInline(inner)
				nodes = append(nodes, n)
				i += 2 + j
				continue
			}
		}

		// —— plain text fallback
		j := i
		for j < len(text) && text[j] != '*' && text[j] != '~' {
			j++
		}
		if j == i {
			// unmatched '*' or '~', consume one char
			nodes = append(nodes, &baseNode{
				nodeType: TextNode,
				content:  text[i : i+1],
			})
			i++
		} else {
			// real plain text
			nodes = append(nodes, &baseNode{
				nodeType: TextNode,
				content:  text[i:j],
			})
			i = j
		}
	}
	return nodes
}
