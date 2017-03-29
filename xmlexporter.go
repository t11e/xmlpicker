package xmlpicker

import (
	"encoding/xml"
	"sort"
	"strings"
)

type XMLExporter struct {
	Encoder *xml.Encoder
	hasNS   bool
}

func (e *XMLExporter) EncodeNode(node *Node) error {
	e.hasNS = false
	if err := e.startPath(node); err != nil {
		return err
	}
	if text, ok := node.Text(); ok {
		if err := e.encodeText(text); err != nil {
			return err
		}
	} else {
		for _, child := range node.Children {
			if err := e.encodeNode(child); err != nil {
				return err
			}
		}
	}
	return e.endPath(node)
}

func (e *XMLExporter) encodeNode(n *Node) error {
	if text, ok := n.Text(); ok {
		return e.encodeText(text)
	}
	if err := e.encodeStartElement(n); err != nil {
		return err
	}
	for _, child := range n.Children {
		if err := e.encodeNode(child); err != nil {
			return err
		}
	}
	if err := e.encodeEndElement(n); err != nil {
		return err
	}
	return nil
}

func (e *XMLExporter) startPath(node *Node) error {
	if node.Parent == nil {
		return nil
	}
	if err := e.startPath(node.Parent); err != nil {
		return err
	}
	return e.encodeStartElement(node)
}

func (e *XMLExporter) endPath(node *Node) error {
	if node.Parent == nil {
		return nil
	}
	if err := e.encodeEndElement(node); err != nil {
		return err
	}
	return e.endPath(node.Parent)
}

func (e *XMLExporter) encodeStartElement(node *Node) error {
	if node.Namespaces != nil {
		e.hasNS = true
	}
	token := xml.StartElement{Name: node.StartElement.Name, Attr: e.fixAttributes(node)}
	e.fixElementName(&token.Name, node.Parent)
	return e.Encoder.EncodeToken(token)
}

func (e *XMLExporter) encodeEndElement(node *Node) error {
	token := xml.EndElement{Name: node.StartElement.Name}
	e.fixElementName(&token.Name, node.Parent)
	return e.Encoder.EncodeToken(token)
}

func (e *XMLExporter) fixAttributes(node *Node) []xml.Attr {
	if !e.hasNS {
		return node.StartElement.Attr
	}
	attr := make([]xml.Attr, 0, len(node.Namespaces)+len(node.StartElement.Attr))
	for _, a := range node.StartElement.Attr {
		if a.Name.Space != "" {
			a.Name.Local = a.Name.Space + ":" + a.Name.Local
			a.Name.Space = ""
		}
		attr = append(attr, a)
	}
	if len(node.Namespaces) != 0 {
		ks := make([]string, 0, len(node.Namespaces))
		for k := range node.Namespaces {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			var name string
			if k == "" {
				name = "xmlns"
			} else {
				name = "xmlns:" + k
			}
			attr = append(attr, xml.Attr{
				Name:  xml.Name{Local: name},
				Value: node.Namespaces[k],
			})
		}
	}
	return attr
}

func (e *XMLExporter) fixElementName(name *xml.Name, parent *Node) {
	if name.Space != "" {
		if e.hasNS && name.Space != "" {
			name.Local = name.Space + ":" + name.Local
			name.Space = ""
		}
		if name.Space == parent.StartElement.Name.Space {
			name.Space = ""
		}
	}
}

func (e *XMLExporter) encodeText(text string) error {
	text = strings.Replace(text, "\n", "&#10;", -1)
	text = strings.Replace(text, "\r", "&#13;", -1)
	return e.Encoder.EncodeToken(xml.CharData([]byte(text)))
}
