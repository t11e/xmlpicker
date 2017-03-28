package xmlpicker

import "fmt"

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
