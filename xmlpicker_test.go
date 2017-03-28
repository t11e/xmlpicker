package xmlpicker_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t11e/xmlpicker"
)

func TestSimpleSelector(t *testing.T) {
	for idx, test := range []struct {
		selector       string
		xml            string
		nsFlag         xmlpicker.NSFlag
		expandPrefixes bool
		expected       []string
	}{
		{
			xml:      `<a><b/><c/></a>`,
			expected: []string{"/a"},
		},
		{
			selector: "*",
			xml:      `<a><b/><c/></a>`,
			expected: []string{"/a"},
		},
		{
			selector: "/",
			xml:      `<a><b/><c/></a>`,
			expected: []string{"/a"},
		},
		{
			selector: "/*",
			xml:      `<a><b/><c/></a>`,
			expected: []string{"/a"},
		},
		{
			selector: "/a",
			xml:      `<a><b/><c/></a>`,
			expected: []string{"/a"},
		},
		{
			selector: "/a/",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/c", "/a/b"},
		},
		{
			selector: "/a/*",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/c", "/a/b"},
		},
		{
			selector: "/*/",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/c", "/a/b"},
		},
		{
			selector: "/a/b",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/b"},
		},
		{
			selector: "/*/b",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/b"},
		},
		{
			selector: "/a/b/c",
			xml:      `<a><b><c/></b><c/><b><c/></b><b><d/></b></a>`,
			expected: []string{"/a/b/c", "/a/b/c"},
		},
		{
			selector: "/a/*/c",
			xml:      `<a><b><c/></b><c/><b><c/></b><b><d/></b></a>`,
			expected: []string{"/a/b/c", "/a/b/c"},
		},
		{
			selector: "/*/b/c",
			xml:      `<a><b><c/></b><c/><b><c/></b><b><d/></b></a>`,
			expected: []string{"/a/b/c", "/a/b/c"},
		},
		{
			selector: "/*/*/c",
			xml:      `<a><b><c/></b><c/><b><c/></b><b><d/></b></a>`,
			expected: []string{"/a/b/c", "/a/b/c"},
		},

		{
			selector: "/root/",
			xml:      `<root xmlns:x="X" xmlns:y="Y"><x:a/><y:a/><x:a/></root>`,
			expected: []string{"/root/X:a", "/root/Y:a", "/root/X:a"},
		},
		{
			selector: "/root/",
			xml:      `<root xmlns:x="X" xmlns:y="Y"><x:a/><y:a/><x:a/></root>`,
			nsFlag:   xmlpicker.NSStrip,
			expected: []string{"/root/a", "/root/a", "/root/a"},
		},
		{
			selector: "/root/",
			xml:      `<root xmlns:x="X" xmlns:y="Y"><x:a/><y:a/><x:a/></root>`,
			nsFlag:   xmlpicker.NSPrefix,
			expected: []string{"/root/x:a", "/root/y:a", "/root/x:a"},
		},
		{
			selector:       "/root/",
			xml:            `<root xmlns:x="X" xmlns:y="Y"><x:a/><y:a/><x:a/></root>`,
			nsFlag:         xmlpicker.NSPrefix,
			expandPrefixes: true,
			expected:       []string{"/root/X:a", "/root/Y:a", "/root/X:a"},
		},

		{
			selector: "/root/",
			xml:      `<root xmlns:x="X"><x:a xmlns:x="X2"></x:a><x:b/></root>`,
			expected: []string{"/root/X2:a", "/root/X:b"},
		},
		{
			selector: "/root/",
			xml:      `<root xmlns:x="X"><x:a xmlns:x="X2"></x:a><x:b/></root>`,
			nsFlag:   xmlpicker.NSStrip,
			expected: []string{"/root/a", "/root/b"},
		},
		{
			selector: "/root/",
			xml:      `<root xmlns:x="X"><x:a xmlns:x="X2"></x:a><x:b/></root>`,
			nsFlag:   xmlpicker.NSPrefix,
			expected: []string{"/root/x:a", "/root/x:b"},
		},
		{
			selector:       "/root/",
			xml:            `<root xmlns:x="X"><x:a xmlns:x="X2"></x:a><x:b/></root>`,
			nsFlag:         xmlpicker.NSPrefix,
			expandPrefixes: true,
			expected:       []string{"/root/X2:a", "/root/X:b"},
		},
	} {
		var variant string
		if test.expandPrefixes {
			variant = " expandPrefixes"
		}
		name := fmt.Sprintf("%d %s %s%s", idx, test.selector, test.nsFlag, variant)
		t.Run(name, func(t *testing.T) {
			actual := make([]string, 0)
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector(test.selector))
			parser.NSFlag = test.nsFlag
			for {
				node, err := parser.Next()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err, "%s\nXML:\n%s\n", name, test.xml) {
					return
				}
				i := node.Depth() + 1
				parts := make([]string, i, i)
				for n := node; n.Parent != nil; n = n.Parent {
					name := n.StartElement.Name
					space := name.Space
					if space != "" && test.expandPrefixes {
						var ok bool
						if space, ok = n.LookupPrefix(name.Space); !ok {
							space = fmt.Sprintf("!{%space}MISSING", name.Space)
						}
					}
					var part string
					if space != "" {
						part = fmt.Sprintf("%s:%s", space, name.Local)
					} else {
						part = name.Local
					}
					i = i - 1
					parts[i] = part
				}
				actual = append(actual, strings.Join(parts, "/"))
			}
			assert.Equal(t, test.expected, actual, "%s\nXML:\n%s\n", name, test.xml)
		})
	}
}

func TestParserNext(t *testing.T) {
	for idx, test := range []struct {
		name        string
		xml         string
		nsFlag      xmlpicker.NSFlag
		expected    int
		expectedErr string
	}{
		{
			name:     "control",
			xml:      `<a/>`,
			expected: 1,
		},
		{
			name:     "control",
			xml:      `<a/>`,
			nsFlag:   xmlpicker.NSStrip,
			expected: 1,
		},
		{
			name:     "control",
			xml:      `<a/>`,
			nsFlag:   xmlpicker.NSPrefix,
			expected: 1,
		},

		{
			name:     "empty",
			xml:      ``,
			expected: 0,
		},
		{
			name:     "empty",
			xml:      ``,
			nsFlag:   xmlpicker.NSStrip,
			expected: 0,
		},
		{
			name:     "empty",
			xml:      ``,
			nsFlag:   xmlpicker.NSPrefix,
			expected: 0,
		},

		{
			name:     "junk",
			xml:      `   abc>@;:&#38;""''!-123 `,
			expected: 0,
		},
		{
			name:     "junk",
			xml:      `   abc>@;:&#38;""''!-123 `,
			nsFlag:   xmlpicker.NSStrip,
			expected: 0,
		},
		{
			name:     "junk",
			xml:      `   abc>@;:&#38;""''!-123 `,
			nsFlag:   xmlpicker.NSPrefix,
			expected: 0,
		},

		{
			name:        "eof",
			xml:         `<a>`,
			expectedErr: "XML syntax error on line 1: unexpected EOF",
		},
		{
			name:        "eof",
			xml:         `<a>`,
			nsFlag:      xmlpicker.NSStrip,
			expectedErr: "XML syntax error on line 1: unexpected EOF",
		},
		{
			name:        "eof",
			xml:         `<a>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "xmlpicker: unexpected EOF",
		},

		{
			name:        "invalid just end element",
			xml:         `</a>`,
			expectedErr: "XML syntax error on line 1: unexpected end element </a>",
		},
		{
			name:        "invalid just end element",
			xml:         `</a>`,
			nsFlag:      xmlpicker.NSStrip,
			expectedErr: "XML syntax error on line 1: unexpected end element </a>",
		},
		{
			name:        "invalid just end element",
			xml:         `</a>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "xmlpicker: unexpected end element </a>",
		},

		{
			name:        "invalid element name",
			xml:         `<*123/>`,
			expectedErr: "XML syntax error on line 1: expected element name after <",
		},
		{
			name:        "invalid element name",
			xml:         `<*123/>`,
			nsFlag:      xmlpicker.NSStrip,
			expectedErr: "XML syntax error on line 1: expected element name after <",
		},
		{
			name:        "invalid element name",
			xml:         `<*123/>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "XML syntax error on line 1: expected element name after <",
		},

		{
			name:        "mismatched element local",
			xml:         `<a></b>`,
			expectedErr: "XML syntax error on line 1: element <a> closed by </b>",
		},
		{
			name:        "mismatched element local",
			xml:         `<a></b>`,
			nsFlag:      xmlpicker.NSStrip,
			expectedErr: "XML syntax error on line 1: element <a> closed by </b>",
		},
		{
			name:        "mismatched element local",
			xml:         `<a></b>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "xmlpicker: element <a> closed by </b>",
		},

		{
			name:        "mismatched element space",
			xml:         `<x:a></y:a>`,
			expectedErr: "XML syntax error on line 1: element <a> in space xclosed by </a> in space y",
		},
		{
			name:        "mismatched element space",
			xml:         `<x:a></y:a>`,
			nsFlag:      xmlpicker.NSStrip,
			expectedErr: "XML syntax error on line 1: element <a> in space xclosed by </a> in space y",
		},
		{
			name:        "mismatched element space",
			xml:         `<x:a></y:a>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "xmlpicker: element <a> in space x closed by </a> in space y",
		},

		{
			name:     "different space prefix, valid xml",
			xml:      `<root xmlns:x1="http://example.com/x" xmlns:x2="http://example.com/x"><x1:a></x2:a></root>`,
			expected: 1,
		},
		{
			name:     "different space prefix, valid xml",
			xml:      `<root xmlns:x1="http://example.com/x" xmlns:x2="http://example.com/x"><x1:a></x2:a></root>`,
			nsFlag:   xmlpicker.NSStrip,
			expected: 1,
		},
		{
			name:        "different space prefix, valid xml",
			xml:         `<root xmlns:x1="http://example.com/x" xmlns:x2="http://example.com/x"><x1:a></x2:a></root>`,
			nsFlag:      xmlpicker.NSPrefix,
			expectedErr: "xmlpicker: element <a> in space x1 closed by </a> in space x2",
		},
	} {
		name := fmt.Sprintf("%d %s %s", idx, test.name, test.nsFlag)
		t.Run(name, func(t *testing.T) {
			actual := 0
			var actualErr error
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector("/"))
			parser.NSFlag = test.nsFlag
			for {
				_, err := parser.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					actualErr = err
					break
				}
				actual = actual + 1
			}
			if test.expectedErr != "" {
				assert.EqualError(t, actualErr, test.expectedErr, "%s\nXML:\n%s\n", name, test.xml)
			} else {
				assert.NoError(t, actualErr, "%s\nXML:\n%s\n", name, test.xml)
			}
			assert.Equal(t, test.expected, actual, "%s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", name, test.xml, test.expected, actual)
		})
	}
}

func startElement(e *xml.Encoder, node *xmlpicker.Node, nsFlag xmlpicker.NSFlag) xml.StartElement {
	if nsFlag != xmlpicker.NSPrefix {
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

func endElement(e *xml.Encoder, node *xmlpicker.Node, nsFlag xmlpicker.NSFlag) xml.EndElement {
	if nsFlag != xmlpicker.NSPrefix {
		return xml.EndElement{Name: node.StartElement.Name}
	}
	name := node.StartElement.Name.Local
	if node.StartElement.Name.Space != "" {
		name = node.StartElement.Name.Space + ":" + name
	}
	return xml.EndElement{Name: xml.Name{Local: name}}
}

func startNode(e *xml.Encoder, node *xmlpicker.Node, nsFlag xmlpicker.NSFlag) error {
	if node.Parent == nil {
		return nil
	}
	if err := startNode(e, node.Parent, nsFlag); err != nil {
		return err
	}
	return e.EncodeToken(startElement(e, node, nsFlag))
}

func endNode(e *xml.Encoder, node *xmlpicker.Node, nsFlag xmlpicker.NSFlag) error {
	if node.Parent == nil {
		return nil
	}
	if err := e.EncodeToken(endElement(e, node, nsFlag)); err != nil {
		return err
	}
	return endNode(e, node.Parent, nsFlag)
}

func isolateNode(e *xml.Encoder, node *xmlpicker.Node, nsFlag xmlpicker.NSFlag) error {
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

func isolateNodeImpl(e *xml.Encoder, n *xmlpicker.Node, nsFlag xmlpicker.NSFlag) error {
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

func TestXMLExporter(t *testing.T) {
	for idx, test := range []struct {
		name        string
		selector    string
		xml         string
		nsFlag      xmlpicker.NSFlag
		expected    string
		expectedErr string
	}{
		{
			name:     "control",
			xml:      `<a/>`,
			selector: "/",
			expected: `<a></a>`,
		},
		{
			name:     "simple",
			xml:      `<a><b/><c/></a>`,
			selector: "/*/",
			expected: `<a><b></b></a><a><c></c></a>`,
		},
		{
			name:     "deeper 1",
			xml:      `<a id="1">one<b id="2">two<c id="3">three</c>four</b>five<b id="4">six<c id="5">seven</c>eight</b>nine</a>`,
			selector: "/*/",
			expected: `<a id="1"><b id="2">two<c id="3">three</c>four</b></a>` + `<a id="1"><b id="4">six<c id="5">seven</c>eight</b></a>`,
		},
		{
			name:     "deeper 2",
			xml:      `<a id="1">one<b id="2">two<c id="3">three</c>four</b>five<b id="4">six<c id="5">seven</c>eight</b>nine</a>`,
			selector: "/*/*/c",
			expected: `<a id="1"><b id="2"><c id="3">three</c></b></a>` + `<a id="1"><b id="4"><c id="5">seven</c></b></a>`,
		},

		{
			name:     "root namespace",
			xml:      `<a xmlns:a="aaa" foo="1" a:bar="2"/>`,
			selector: "/",
			expected: `<a foo="1" xmlns:aaa="aaa" aaa:bar="2"></a>`,
		},
		{
			name:     "root namespace",
			xml:      `<a xmlns:a="aaa" foo="1" a:bar="2"/>`,
			selector: "/",
			nsFlag:   xmlpicker.NSStrip,
			expected: `<a foo="1" bar="2"></a>`,
		},
		{
			name:     "root namespace",
			xml:      `<a xmlns:a="aaa" foo="1" a:bar="2"/>`,
			selector: "/",
			nsFlag:   xmlpicker.NSPrefix,
			expected: `<a xmlns:a="aaa" a:bar="2" foo="1"></a>`,
		},

		{
			name:     "namespaces",
			xml:      `<a xmlns="DEF" xmlns:a="aaa" foo="1" a:bar="2"><b id="123" foo="3" a:bar="4">first</b><b id="456" foo="5">second</b></a>`,
			selector: "/*/",
			expected: `<a xmlns="DEF" foo="1" xmlns:aaa="aaa" aaa:bar="2"><b xmlns="DEF" id="123" foo="3" aaa:bar="4">first</b></a>` +
				`<a xmlns="DEF" foo="1" xmlns:aaa="aaa" aaa:bar="2"><b xmlns="DEF" id="456" foo="5">second</b></a>`,
		},
		{
			name:     "namespaces",
			xml:      `<a xmlns="DEF" xmlns:a="aaa" foo="1" a:bar="2"><b id="123" foo="3" a:bar="4">first</b><b id="456" foo="5">second</b></a>`,
			selector: "/*/",
			nsFlag:   xmlpicker.NSStrip,
			expected: `<a foo="1" bar="2"><b id="123" foo="3" bar="4">first</b></a>` +
				`<a foo="1" bar="2"><b id="456" foo="5">second</b></a>`,
		},
		{
			name:     "namespaces",
			xml:      `<a xmlns="DEF" xmlns:a="aaa" foo="1" a:bar="2"><b id="123" foo="3" a:bar="4">first</b><b id="456" foo="5">second</b></a>`,
			selector: "/*/",
			nsFlag:   xmlpicker.NSPrefix,
			expected: `<a xmlns:a="aaa" xmlns="DEF" a:bar="2" foo="1"><b a:bar="4" foo="3" id="123">first</b></a><a xmlns:a="aaa" xmlns="DEF" a:bar="2" foo="1"><b foo="5" id="456">second</b></a>`,
		},
	} {
		name := fmt.Sprintf("%d %s %s", idx, test.name, test.nsFlag)
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			e := xml.NewEncoder(&b)
			var actualErr error
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector(test.selector))
			parser.NSFlag = test.nsFlag
			for {
				n, err := parser.Next()
				if err == io.EOF {
					e.Flush()
					break
				}
				if err != nil {
					actualErr = err
					break
				}
				if err := isolateNode(e, n, test.nsFlag); err != nil {
					actualErr = err
					break
				}
			}
			if test.expectedErr != "" {
				assert.EqualError(t, actualErr, test.expectedErr, "%s\nXML:\n%s\n", name, test.xml)
			} else {
				assert.NoError(t, actualErr, "%s\nXML:\n%s\n", name, test.xml)
			}
			actual := strings.TrimSuffix(b.String(), "\n")
			assert.Equal(t, test.expected, actual, "%s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", name, test.xml, test.expected, actual)
		})
	}
}

func TestSimpleMapper(t *testing.T) {
	for idx, test := range []struct {
		name        string
		selector    string
		xml         string
		nsFlag      xmlpicker.NSFlag
		expected    string
		expectedErr string
	}{
		{
			name:     "control",
			xml:      `<a/>`,
			selector: "/",
			expected: `{"_name":"a"}`,
		},
		{
			name:     "attributes",
			xml:      `<a id="1" name="example"/>`,
			selector: "/",
			expected: `{"@id":"1","@name":"example","_name":"a"}`,
		},
		{
			name:     "child",
			xml:      `<a><b/></a>`,
			selector: "/",
			expected: `{"_name":"a","b":[{}]}`,
		},
		{
			name:     "repeating child",
			xml:      `<a><b/><b></b></a>`,
			selector: "/",
			expected: `{"_name":"a","b":[{},{}]}`,
		},
		{
			name:     "text",
			xml:      `<a>hello, world!</a>`,
			selector: "/",
			expected: `{"#text":["hello, world!"],"_name":"a"}`,
		},
		{
			name:     "children with text",
			xml:      `<a><b>hello</b><c>fred</c><c>wilma</c></a>`,
			selector: "/",
			expected: `{"_name":"a","b":[{"#text":["hello"]}],"c":[{"#text":["fred"]},{"#text":["wilma"]}]}`,
		},
		{
			name:     "text and attributes",
			xml:      `<a id="first">hello, world!</a>`,
			selector: "/",
			expected: `{"#text":["hello, world!"],"@id":"first","_name":"a"}`,
		},
		{
			name:     "text and attributes and children",
			xml:      `<a id="first"><b id="second">hello</b><c id="third">fred</c><c>wilma</c><c id="last"/></a>`,
			selector: "/",
			expected: `{"@id":"first","_name":"a","b":[{"#text":["hello"],"@id":"second"}],"c":[{"#text":["fred"],"@id":"third"},{"#text":["wilma"]},{"@id":"last"}]}`,
		},
		{
			name:     "mixed text and children",
			xml:      `<a>hello <b>fred</b> and <b>wilma</b></a>`,
			selector: "/",
			expected: `{"#text":["hello","and"],"_name":"a","b":[{"#text":["fred"]},{"#text":["wilma"]}]}`,
		},
	} {
		name := fmt.Sprintf("%d %s %s", idx, test.name, test.nsFlag)
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			e := json.NewEncoder(&b)
			e.SetEscapeHTML(false)
			mapper := xmlpicker.SimpleMapper{}
			var actualErr error
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector(test.selector))
			parser.NSFlag = test.nsFlag
			for {
				n, err := parser.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					actualErr = err
					break
				}
				v, err := mapper.FromNode(n)
				if err != nil {
					actualErr = err
					break
				}
				err = e.Encode(v)
				if err != nil {
					actualErr = err
					break
				}
			}
			if test.expectedErr != "" {
				assert.EqualError(t, actualErr, test.expectedErr, "%s\nXML:\n%s\n", name, test.xml)
			} else {
				assert.NoError(t, actualErr, "%s\nXML:\n%s\n", name, test.xml)
			}
			actual := strings.TrimSuffix(b.String(), "\n")
			assert.Equal(t, test.expected, actual, "%s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", name, test.xml, test.expected, actual)
		})
	}
}
