package xmlpicker

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Node struct {
	StartElement xml.StartElement
	Parent       *Node
	Namespaces   Namespaces
	Children     []*Node
}

type Namespaces map[string]string

type Path []*Node

func (node *Node) Text() (string, bool) {
	return decodeText(&node.StartElement)
}
func (node *Node) SetText(text string) {
	encodeText(&node.StartElement, text)
}

func decodeText(e *xml.StartElement) (string, bool) {
	if e.Name.Local != "" || e.Name.Space != "" {
		return "", false
	}
	if len(e.Attr) != 1 {
		return "", false
	}
	if e.Attr[0].Name.Local != "" || e.Attr[0].Name.Space != "" {
		return "", false
	}
	return e.Attr[0].Value, true
}

func encodeText(e *xml.StartElement, text string) {
	e.Name.Local = ""
	e.Name.Space = ""
	e.Attr = []xml.Attr{{Value: text}}
}

func (node *Node) Depth() int {
	d := 0
	for n := node; n != nil && n.Parent != nil; n = n.Parent {
		d = d + 1
	}
	return d
}

func (node *Node) LookupPrefix(prefix string) (string, bool) {
	for n := node; n != nil; n = n.Parent {
		if ns, ok := n.Namespaces[prefix]; ok {
			return ns, ok
		}
	}
	return prefix, false
}

type Selector interface {
	Matches(node *Node) bool
}

func PathSelector(path string) Selector {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	parts := strings.Split(path, "/")
	for i, v := range parts {
		parts[i] = strings.TrimSpace(v)
	}
	for i, v := range parts {
		if i != 0 && v == "" {
			parts[i] = "*"
		}
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return pathSelector(parts)
}

type pathSelector []string

func (s pathSelector) Matches(node *Node) bool {
	i := 0
	for n := node; n != nil && i < len(s); n = n.Parent {
		p := s[i]
		if p != "*" && p != n.StartElement.Name.Local {
			return false
		}
		i = i + 1
	}
	return i == len(s)
}

func NewParser(decoder *xml.Decoder, selector Selector) *Parser {
	p := &Parser{
		MaxDepth:    1000,
		MaxChildren: 1000,
		MaxTokens:   -1,
		decoder:     decoder,
		selector:    selector,
		node:        &Node{},
	}
	return p
}

type Parser struct {
	NSFlag      NSFlag
	MaxDepth    int
	MaxChildren int
	MaxTokens   int

	decoder    *xml.Decoder
	selector   Selector
	tokenCount int
	node       *Node
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

var UnexpectedEOF = errors.New("xmlpicker: unexpected EOF")

func (p *Parser) Next() (*Node, error) {
	if p.node == nil {
		return nil, errors.New("xmlpicker: will no longer consume tokens, Next() called after error")
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
			if err == io.EOF && p.node.Children != nil {
				return nil, UnexpectedEOF
			}
			return nil, err
		}
		p.tokenCount = p.tokenCount + 1
		if p.MaxTokens != -1 && p.tokenCount > p.MaxTokens {
			p.node = nil
			return nil, fmt.Errorf("xmlpicker: token limit reached %d", p.MaxTokens)
		}
		switch t := t.(type) {
		case xml.StartElement:
			p.push(t)
			if p.node.Depth() > p.MaxDepth {
				p.node = nil
				return nil, fmt.Errorf("xmlpicker: depth limit reached %d", p.MaxDepth)
			}
			if p.node.Parent.Children == nil {
				if p.selector.Matches(p.node) {
					p.node.Children = make([]*Node, 0)
				}
				continue
			}
			p.node.Children = make([]*Node, 0)
			p.node.Parent.Children = append(p.node.Parent.Children, p.node)
			if len(p.node.Parent.Children) > p.MaxChildren {
				return nil, fmt.Errorf("xmlpicker: maximum node child limit reached %d", p.MaxChildren)
			}
		case xml.EndElement:
			prev, err := p.pop(t)
			if err != nil {
				p.node = nil
				return nil, err
			}
			if prev.Children != nil && p.node.Children == nil {
				return prev, nil
			}
		case xml.CharData:
			if p.node.Children == nil {
				continue
			}
			s := strings.TrimSpace(string(t.Copy()))
			if len(s) == 0 {
				continue
			}
			node := &Node{Parent: p.node}
			node.SetText(s)
			p.node.Children = append(p.node.Children, node)
			if len(p.node.Children) > p.MaxChildren {
				return nil, fmt.Errorf("xmlpicker: maximum node child limit reached %d", p.MaxChildren)
			}
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
			return nil, fmt.Errorf("xmlpicker: unexpected xml token %+v", t)
		}
	}
}

// push adds start to the path.
// Namespace handling is similar to xml.Token().
func (p *Parser) push(start xml.StartElement) *Node {
	element := xml.StartElement{Name: start.Name}
	if p.NSFlag == NSStrip {
		element.Name.Space = ""
	}
	update := false
	for _, a := range start.Attr {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			update = true
			break
		}
		if p.NSFlag == NSStrip && a.Name.Space != "" {
			update = true
			break
		}
	}
	var ns Namespaces
	if !update {
		element.Attr = make([]xml.Attr, len(start.Attr))
		copy(element.Attr, start.Attr)
	} else {
		if p.NSFlag == NSPrefix {
			ns = make(Namespaces)
		}
		element.Attr = make([]xml.Attr, 0, len(start.Attr))
		for _, a := range start.Attr {
			if a.Name.Space == "xmlns" {
				if ns != nil {
					ns[a.Name.Local] = a.Value
				}
				continue
			}
			if a.Name.Space == "" && a.Name.Local == "xmlns" { // default space for untagged names
				if ns != nil {
					ns[""] = a.Value
				}
				continue
			}
			if p.NSFlag == NSStrip {
				a.Name.Space = ""
			}
			element.Attr = append(element.Attr, a)
		}
	}
	pushed := &Node{
		StartElement: element,
		Namespaces:   ns,
		Parent:       p.node,
	}
	// TODO needed?
	//if p.NSFlag == NSPrefix && pushed.StartElement.Name.Space != "" {
	//	if defaultSpace, ok := pushed.LookupPrefix(""); ok && defaultSpace == pushed.StartElement.Name.Space {
	//		pushed.StartElement.Name.Space = ""
	//	}
	//}
	p.node = pushed
	return pushed
}

// pop removes the end element from the path and returns an error if it does not match the appropriate start element.
// Normally xml.Decoder.Token() would do this for us but we are using xml.Decoder.RawToken() instead to allow for
// access of the XML namespace prefixes.
// Syntax errors handling is similar to xml.popElement().
func (p *Parser) pop(end xml.EndElement) (*Node, error) {
	if p.node.Parent == nil {
		return nil, fmt.Errorf("xmlpicker: unexpected end element </%s>", end.Name.Local)
	}
	popped := p.node
	start := popped.StartElement
	if start.Name.Local != end.Name.Local {
		return nil, fmt.Errorf("xmlpicker: element <%s> closed by </%s>", start.Name.Local, end.Name.Local)
	}
	if p.NSFlag != NSStrip && start.Name.Space != end.Name.Space {
		return nil, fmt.Errorf("xmlpicker: element <%s> in space %s closed by </%s> in space %s", start.Name.Local, start.Name.Space, end.Name.Local, end.Name.Space)
	}
	p.node = popped.Parent
	return popped, nil
}

type Mapper interface {
	FromNode(node *Node) (map[string]interface{}, error)
}

type SimpleMapper struct {
}

func (m SimpleMapper) FromNode(n *Node) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	return m.fromNodeImpl(out, n, 0)
}

func (m SimpleMapper) fromNodeImpl(out map[string]interface{}, n *Node, depth int) (map[string]interface{}, error) {
	if text, ok := n.Text(); ok {
		out["#text"] = []string{text}
		return out, nil
	}
	if depth == 0 {
		out["_name"] = n.StartElement.Name.Local
	}
	for _, a := range n.StartElement.Attr {
		out[fmt.Sprintf("@%s", a.Name.Local)] = a.Value
	}
	for _, c := range n.Children {
		var key string
		var value interface{}
		if text, ok := c.Text(); ok {
			key = "#text"
			value = text
		} else {
			key = c.StartElement.Name.Local
			var err error
			value, err = m.fromNodeImpl(make(map[string]interface{}), c, depth+1)
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
