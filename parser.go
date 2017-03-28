package xmlpicker

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

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

type Selector interface {
	Matches(node *Node) bool
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
