package xmlpicker

import (
	"encoding/xml"
	"sort"
)

// TODO Productize this functionality and move its test file to the _test module

func startElement(e *xml.Encoder, node *Node, nsFlag NSFlag) xml.StartElement {
	var attr []xml.Attr
	if nsFlag != NSPrefix || (node.StartElement.Attr == nil && node.Namespaces == nil) {
		attr = node.StartElement.Attr
	} else {
		attr = make([]xml.Attr, 0, len(node.Namespaces)+len(node.StartElement.Attr))
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
	}
	result := xml.StartElement{Name: node.StartElement.Name, Attr: attr}
	if result.Name.Space != "" {
		if nsFlag == NSPrefix {
			result.Name.Local = result.Name.Space + ":" + result.Name.Local
			result.Name.Space = ""
		}
		if nsFlag == NSExpand && result.Name.Space == node.Parent.StartElement.Name.Space {
			result.Name.Space = ""
		}
	}
	return result
}

func endElement(e *xml.Encoder, node *Node, nsFlag NSFlag) xml.EndElement {
	result := xml.EndElement{Name: node.StartElement.Name}
	if result.Name.Space != "" {
		if nsFlag == NSPrefix && result.Name.Space != "" {
			result.Name.Local = result.Name.Space + ":" + result.Name.Local
			result.Name.Space = ""
		}
		if nsFlag == NSExpand && result.Name.Space == node.Parent.StartElement.Name.Space {
			result.Name.Space = ""
		}
	}
	return result
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
