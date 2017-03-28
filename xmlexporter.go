package xmlpicker

import (
	"encoding/xml"
)

// TODO Productize this functionality and move its test file to the _test module

func startElement(e *xml.Encoder, node *Node, nsFlag NSFlag) xml.StartElement {
	if nsFlag != NSPrefix {
		return node.StartElement
	}
	var attr []xml.Attr
	if node.StartElement.Attr != nil {
		i := len(node.Namespaces) + len(node.StartElement.Attr)
		attr = make([]xml.Attr, i, i)
		i = i - 1
		for _, a := range node.StartElement.Attr {
			if a.Name.Space != "" {
				a.Name.Local = a.Name.Space + ":" + a.Name.Local
				a.Name.Space = ""
			}
			attr[i] = a
			i = i - 1
		}
		for k, v := range node.Namespaces {
			var name string
			if k == "" {
				name = "xmlns"
			} else {
				name = "xmlns:" + k
			}
			attr[i] = xml.Attr{
				Name:  xml.Name{Local: name},
				Value: v,
			}
			i = i - 1
		}
	}
	name := node.StartElement.Name.Local
	if node.StartElement.Name.Space != "" {
		name = node.StartElement.Name.Space + ":" + name
	}
	return xml.StartElement{Name: xml.Name{Local: name}, Attr: attr}
}

func endElement(e *xml.Encoder, node *Node, nsFlag NSFlag) xml.EndElement {
	if nsFlag != NSPrefix {
		return xml.EndElement{Name: node.StartElement.Name}
	}
	name := node.StartElement.Name.Local
	if node.StartElement.Name.Space != "" {
		name = node.StartElement.Name.Space + ":" + name
	}
	return xml.EndElement{Name: xml.Name{Local: name}}
}

func startNode(e *xml.Encoder, node *Node, nsFlag NSFlag) error {
	if node.Parent == nil {
		return nil
	}
	if err := startNode(e, node.Parent, nsFlag); err != nil {
		return err
	}
	return e.EncodeToken(startElement(e, node, nsFlag))
}

func endNode(e *xml.Encoder, node *Node, nsFlag NSFlag) error {
	if node.Parent == nil {
		return nil
	}
	if err := e.EncodeToken(endElement(e, node, nsFlag)); err != nil {
		return err
	}
	return endNode(e, node.Parent, nsFlag)
}

func XMLExport(e *xml.Encoder, node *Node, nsFlag NSFlag) error {
	if err := startNode(e, node, nsFlag); err != nil {
		return err
	}
	if text, ok := node.Text(); ok {
		if err := e.EncodeToken([]byte(text)); err != nil {
			return err
		}
	} else {
		for _, child := range node.Children {
			if err := isolateNodeImpl(e, child, nsFlag); err != nil {
				return err
			}
		}
	}
	return endNode(e, node, nsFlag)
}

func isolateNodeImpl(e *xml.Encoder, n *Node, nsFlag NSFlag) error {
	if text, ok := n.Text(); ok {
		return e.EncodeToken(xml.CharData([]byte(text)))
	}
	if err := e.EncodeToken(startElement(e, n, nsFlag)); err != nil {
		return err
	}
	for _, child := range n.Children {
		if err := isolateNodeImpl(e, child, nsFlag); err != nil {
			return err
		}
	}
	if err := e.EncodeToken(endElement(e, n, nsFlag)); err != nil {
		return err
	}

	return nil
}
