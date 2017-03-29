package xmlpicker

type Mapper interface {
	FromNode(node *Node) (map[string]interface{}, error)
}

type SimpleMapper struct {
	hasNS bool
}

func (m SimpleMapper) FromNode(node *Node) (map[string]interface{}, error) {
	m.hasNS = false
	for n := node; n != nil; n = n.Parent {
		if n.Namespaces != nil {
			m.hasNS = true
			break
		}
	}
	out := make(map[string]interface{})
	return m.fromNodeImpl(out, node, 0)
}

func (m SimpleMapper) fromNodeImpl(out map[string]interface{}, node *Node, depth int) (map[string]interface{}, error) {
	if text, ok := node.Text(); ok {
		out["#text"] = []string{text}
		return out, nil
	}
	if depth == 0 {
		out["_name"] = node.StartElement.Name.Local
		if node.StartElement.Name.Space != "" {
			out["_namespace"] = node.StartElement.Name.Space
		}
	}
	if node.Namespaces != nil {
		m.hasNS = true
		out["_namespaces"] = node.Namespaces
	}
	for _, a := range node.StartElement.Attr {
		var key string
		if a.Name.Space == "" {
			key = "@" + a.Name.Local
		} else if m.hasNS {
			key = "@" + a.Name.Space + ":" + a.Name.Local
		} else {
			key = "@" + a.Name.Local + " " + a.Name.Space
		}
		out[key] = a.Value
	}
	for _, c := range node.Children {
		var key string
		var value interface{}
		if text, ok := c.Text(); ok {
			key = "#text"
			value = text
		} else {
			if c.StartElement.Name.Space == "" {
				key = c.StartElement.Name.Local
			} else if m.hasNS {
				key = c.StartElement.Name.Space + ":" + c.StartElement.Name.Local
			} else {
				key = c.StartElement.Name.Local + " " + c.StartElement.Name.Space
			}
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
