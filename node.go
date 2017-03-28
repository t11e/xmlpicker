package xmlpicker

import "encoding/xml"

type Node struct {
	StartElement xml.StartElement
	Parent       *Node
	Namespaces   Namespaces
	Children     []*Node
}

type Namespaces map[string]string

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
