package xmlpicker

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

type XMLExporter struct {
	Encoder *xml.Encoder
	hasNS   bool
}

func (e *XMLExporter) EncodeNode(node *Node) error {
	if text, ok := node.Text(); ok {
		return e.encodeText(text)
	}
	if err := e.encodeStartElement(node); err != nil {
		return err
	}
	for _, child := range node.Children {
		if err := e.EncodeNode(child); err != nil {
			return err
		}
	}
	return e.encodeEndElement(node)
}

func (e *XMLExporter) StartPath(node *Node) error {
	e.hasNS = false
	if node.Parent == nil {
		return nil
	}
	if err := e.StartPath(node.Parent); err != nil {
		return err
	}
	return e.encodeStartElement(node)
}

func (e *XMLExporter) EndPath(node *Node) error {
	if node.Parent == nil {
		return nil
	}
	if err := e.encodeEndElement(node); err != nil {
		return err
	}
	return e.EndPath(node.Parent)
}

func (e *XMLExporter) encodeStartElement(node *Node) error {
	if node.Namespaces != nil {
		e.hasNS = true
	}
	attr, err := e.fixAttributes(node)
	if err != nil {
		return err
	}
	token := xml.StartElement{Name: node.StartElement.Name, Attr: attr}
	if err := e.fixElementName(&token.Name, node); err != nil {
		return err
	}
	return e.Encoder.EncodeToken(token)
}

func (e *XMLExporter) encodeEndElement(node *Node) error {
	token := xml.EndElement{Name: node.StartElement.Name}
	if err := e.fixElementName(&token.Name, node); err != nil {
		return err
	}
	return e.Encoder.EncodeToken(token)
}

func (e *XMLExporter) fixAttributes(node *Node) ([]xml.Attr, error) {
	if !e.hasNS {
		return node.StartElement.Attr, nil
	}
	attr := make([]xml.Attr, 0, len(node.Namespaces)+len(node.StartElement.Attr))
	for _, a := range node.StartElement.Attr {
		if a.Name.Space != "" {
			if err := e.validatePrefix(node, a.Name.Space); err != nil {
				return nil, err
			}
			a.Name.Local = a.Name.Space + ":" + a.Name.Local
			a.Name.Space = ""
		}
		attr = append(attr, a)
	}
	if len(node.Namespaces) != 0 {
		ks := make([]string, 0, len(node.Namespaces))
		for k, v := range node.Namespaces {
			if prev, ok := node.Parent.LookupPrefix(k); ok && prev == v {
				continue // prefix:ns combination already in place
			}
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
	return attr, nil
}

func (e *XMLExporter) fixElementName(name *xml.Name, node *Node) error {
	if name.Space != "" {
		if e.hasNS && name.Space != "" {
			if err := e.validatePrefix(node, name.Space); err != nil {
				return err
			}
			name.Local = name.Space + ":" + name.Local
			name.Space = ""
		}
		if name.Space == node.Parent.StartElement.Name.Space {
			name.Space = ""
		}
	}
	return nil
}

func (e *XMLExporter) validatePrefix(node *Node, prefix string) error {
	if !e.hasNS || prefix == "" || prefix == "xml" {
		return nil
	}
	if _, ok := node.LookupPrefix(prefix); !ok {
		return fmt.Errorf("xmlpicker: undeclared prefix %s at %s", prefix, (*FormatNodePath)(node))
	}
	return nil
}

func (e *XMLExporter) encodeText(text string) error {
	text = strings.Replace(text, "\n", "&#10;", -1)
	text = strings.Replace(text, "\r", "&#13;", -1)
	return e.Encoder.EncodeToken(xml.CharData([]byte(text)))
}
