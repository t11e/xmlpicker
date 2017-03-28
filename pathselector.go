package xmlpicker

import "strings"

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
