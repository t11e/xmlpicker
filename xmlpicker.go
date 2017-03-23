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

type Path []xml.StartElement

func (p Path) String() string {
	var b bytes.Buffer
	b.WriteRune('/')
	for i, t := range p {
		if i > 0 {
			b.WriteRune('/')
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
	Parts    []string
	MatchLen int
}

func (s simpleSelector) Matches(path Path) bool {
	if len(path) != s.MatchLen {
		return false
	}
	for i, part := range s.Parts {
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

type Parser struct {
	MaxDepth    int
	MaxChildren int
	MaxTokens   int

	decoder    *xml.Decoder
	selector   Selector
	path       Path
	tokenCount int
	node       *Node
}

var EOF = errors.New("EOF")

func (p *Parser) Next() (Path, *Node, error) {
	if p.path == nil {
		return nil, nil, errors.New("xmlpicker: will no longer consume tokens, Next() called after error")
	}
	for {
		t, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF && len(p.path) == 0 && p.node == nil {
				return nil, nil, EOF
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
			p.path = append(p.path, t)
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
				return nil, nil, errors.New("xmlpicker: negative depth detected!")
			}
			var resultPath Path
			var resultNode *Node
			if p.node != nil && p.node.Parent == nil {
				resultPath = p.path
				resultNode = p.node
			}
			p.path = p.path[:len(p.path)-1]
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
