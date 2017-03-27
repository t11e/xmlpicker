package xmlpicker

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Node struct {
	Element  *xml.StartElement
	Text     string
	Parent   *Node
	Children []*Node
}

type Path []*PathElement

type Namespaces map[string]string

type PathElement struct {
	Name       xml.Name
	Namespaces Namespaces
}

func (p *Path) String() string {
	var b bytes.Buffer
	b.WriteRune('/')
	for i, t := range *p {
		if i > 0 {
			b.WriteRune('/')
		}
		if t.Name.Space != "" {
			s, ok := t.Namespaces[t.Name.Space]
			if !ok {
				s = "!" + t.Name.Space
			}
			b.WriteString(s)
			b.WriteRune(':')
		}
		b.WriteString(t.Name.Local)
	}
	return b.String()
}

type Selector interface {
	Matches(path Path) bool
}

func SimpleSelector(selector string) Selector {
	parts := strings.Split(selector, "/")
	if len(parts) > 1 && parts[0] == "" {
		parts = parts[1:]
	}
	matchLen := len(parts)
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return simpleSelector{parts, matchLen}
}

type simpleSelector struct {
	parts    []string
	matchLen int
}

func (s simpleSelector) Matches(path Path) bool {
	if len(path) != s.matchLen {
		return false
	}
	for i, part := range s.parts {
		p := path[i]
		if part != p.Name.Local {
			return false
		}
	}
	return true
}

func NewParser(decoder *xml.Decoder, selector Selector) *Parser {
	return &Parser{
		MaxDepth:    1000,
		MaxChildren: 1000,
		MaxTokens:   -1,
		decoder:     decoder,
		selector:    selector,
		path:        make(Path, 0),
	}
}

type NSFlag int

const (
	NSExpand NSFlag = iota
	NSPrefix
	NSStrip
)

func (f NSFlag) String() string {
	switch f {
	case NSExpand:
		return "NSExpand"
	case NSPrefix:
		return "NSPrefix"
	case NSStrip:
		return "NSStrip"
	default:
		return fmt.Sprintf("!NSFLAG(%d)", f)
	}
}

type Parser struct {
	NSFlag      NSFlag
	MaxDepth    int
	MaxChildren int
	MaxTokens   int

	decoder    *xml.Decoder
	selector   Selector
	path       Path
	tokenCount int
	node       *Node
}

var UnexpectedEOF = errors.New("xmlpicker: unexpected EOF")

// push adds start to the path.
// Namespace handling is similar to xml.Token().
func (path *Path) push(start *xml.StartElement, mapNamespaces bool) {
	if !mapNamespaces {
		*path = append(*path, &PathElement{
			Name: start.Name,
		})
		return

	}
	var ns Namespaces
	if len(*path) != 0 {
		ns = (*path)[len(*path)-1].Namespaces
	}
	modifies := false
	for _, a := range start.Attr {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			modifies = true
			break
		}
	}
	if modifies {
		c := make(Namespaces, len(ns))
		for k, v := range ns {
			c[k] = v
		}
		ns = c
		for _, a := range start.Attr {
			if a.Name.Space == "xmlns" {
				ns[a.Name.Local] = a.Value
			}
			if a.Name.Space == "" && a.Name.Local == "xmlns" { // default space for untagged names
				ns[""] = a.Value
			}
		}

	}
	*path = append(*path, &PathElement{
		Name:       start.Name,
		Namespaces: ns,
	})
}

// pop removes the end element from the path and returns an error if it does not match the appropriate start element.
// Normally xml.Decoder.Token() would do this for us but we are using xml.Decoder.RawToken() instead to allow for
// access of the XML namespace prefixes.
// Syntax errors handling is similar to xml.popElement().
func (path *Path) pop(end *xml.EndElement, checkSpace bool) (*PathElement, error) {
	i := len(*path) - 1
	if i < 0 {
		return nil, fmt.Errorf("xmlpicker: unexpected end element </%s>", end.Name.Local)
	}
	start := (*path)[i]
	if start.Name.Local != end.Name.Local {
		return nil, fmt.Errorf("xmlpicker: element <%s> closed by </%s>", start.Name.Local, end.Name.Local)
	}
	if checkSpace && start.Name.Space != end.Name.Space {
		return nil, fmt.Errorf("xmlpicker: element <%s> in space %s closed by </%s> in space %s", start.Name.Local, start.Name.Space, end.Name.Local, end.Name.Space)
	}
	*path = (*path)[:i]
	return start, nil
}

func (p *Parser) Next() (Path, *Node, error) {
	if p.path == nil {
		return nil, nil, errors.New("xmlpicker: will no longer consume tokens, Next() called after error")
	}
	for {
		var t xml.Token
		var err error
		if p.NSFlag == NSPrefix {
			t, err = p.decoder.RawToken()
		} else {
			t, err = p.decoder.Token()
		}
		if err != nil {
			if err == io.EOF && (len(p.path) != 0 || p.node != nil) {
				return nil, nil, UnexpectedEOF
			}
			return nil, nil, err
		}
		p.tokenCount = p.tokenCount + 1
		if p.MaxTokens != -1 && p.tokenCount > p.MaxTokens {
			p.path = nil
			return nil, nil, fmt.Errorf("xmlpicker: token limit reached %d", p.MaxTokens)
		}
		switch t := t.(type) {
		case xml.StartElement:
			t = t.Copy()
			if p.NSFlag == NSStrip {
				t.Name.Space = ""
				for i := 0; i < len(t.Attr); i = i + 1 {
					t.Attr[i].Name.Space = ""
				}
			}
			p.path.push(&t, p.NSFlag == NSPrefix)
			if len(p.path) > p.MaxDepth {
				p.path = nil
				return nil, nil, fmt.Errorf("xmlpicker: depth limit reached %d", p.MaxDepth)
			}
			if p.node == nil {
				if p.selector.Matches(p.path) {
					p.node = &Node{Element: &t}
				}
				continue
			}
			c := &Node{Element: &t}
			c.Parent = p.node
			p.node.Children = append(p.node.Children, c)
			if len(p.node.Children) > p.MaxChildren {
				return nil, nil, fmt.Errorf("xmlpicker: maximum node child limit reached %d", p.MaxChildren)
			}
			p.node = c
		case xml.EndElement:
			if len(p.path) == 0 {
				p.path = nil
				p.node = nil
				return nil, nil, fmt.Errorf("xmlpicker: unexpected end element </%s>", t.Name.Local)
			}
			var resultPath Path
			var resultNode *Node
			if p.node != nil && p.node.Parent == nil {
				resultPath = p.path
				resultNode = p.node
			}
			_, err := p.path.pop(&t, p.NSFlag != NSStrip)
			if err != nil {
				p.path = nil
				p.node = nil
				return nil, nil, err
			}
			if p.node != nil {
				p.node = p.node.Parent
			}
			if resultPath != nil || resultNode != nil {
				return resultPath, resultNode, nil
			}

		case xml.CharData:
			if p.node == nil {
				continue
			}
			t = t.Copy()
			s := string(t)
			s = strings.TrimSpace(s)
			if len(s) == 0 {
				continue
			}
			c := &Node{Text: s}
			c.Parent = p.node
			p.node.Children = append(p.node.Children, c)
			if len(p.node.Children) > p.MaxChildren {
				return nil, nil, fmt.Errorf("xmlpicker: maximum node child limit reached %d", p.MaxChildren)
			}
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
			return nil, nil, fmt.Errorf("xmlpicker: unexpected xml token %+v", t)
		}
	}
}

type Mapper interface {
	FromNode(node *Node) (map[string]interface{}, error)
}

type SimpleMapper struct {
}

func (xi SimpleMapper) FromNode(n *Node) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	if n.Element == nil {
		out["#text"] = []string{n.Text}
		return out, nil
	}
	for _, a := range n.Element.Attr {
		out[fmt.Sprintf("@%s", a.Name.Local)] = a.Value
	}
	for _, c := range n.Children {
		var key string
		var value interface{}
		if c.Element == nil {
			key = "#text"
			value = c.Text
		} else {
			key = c.Element.Name.Local
			var err error
			value, err = xi.FromNode(c)
			if err != nil {
				return nil, err
			}
		}
		var values []interface{}
		if prev, ok := out[key]; ok {
			values = prev.([]interface{})
		} else {
			values = make([]interface{}, 0)
			out[key] = values
		}
		out[key] = append(values, value)
	}
	return out, nil
}
