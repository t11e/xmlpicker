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
			selector: "/",
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
			selector: "/a/b",
			xml:      `<a><b/><c/><b/></a>`,
			expected: []string{"/a/b", "/a/b"},
		},
		{
			selector: "/a/b/c",
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
		name := fmt.Sprintf("%d %s %s", idx, test.selector, test.nsFlag)
		t.Run(name, func(t *testing.T) {
			actual := make([]string, 0)
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.SimpleSelector(test.selector))
			parser.NSFlag = test.nsFlag
			for {
				path, _, err := parser.Next()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err, "%s\nXML:\n%s\n", name, test.xml) {
					return
				}
				var b bytes.Buffer
				b.WriteRune('/')
				for i, pe := range path {
					if i > 0 {
						b.WriteRune('/')
					}
					s := pe.Name.Space
					if s != "" && test.expandPrefixes {
						var ok bool
						s, ok = pe.Namespaces[pe.Name.Space]
						if !ok {
							s = fmt.Sprintf("!{%s}MISSING", pe.Name.Space)
						}
					}
					if s != "" {
						b.WriteString(s)
						b.WriteRune(':')
					}
					b.WriteString(pe.Name.Local)
				}
				actual = append(actual, b.String())
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
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.SimpleSelector("/"))
			parser.NSFlag = test.nsFlag
			for {
				_, _, err := parser.Next()
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
			expected: `{}`,
		},
		{
			name:     "attributes",
			xml:      `<a id="1" name="example"/>`,
			expected: `{"@id":"1","@name":"example"}`,
		},
		{
			name:     "child",
			xml:      `<a><b/></a>`,
			expected: `{"b":[{}]}`,
		},
		{
			name:     "repeating child",
			xml:      `<a><b/><b></b></a>`,
			expected: `{"b":[{},{}]}`,
		},
		{
			name:     "text",
			xml:      `<a>hello, world!</a>`,
			expected: `{"#text":["hello, world!"]}`,
		},
		{
			name:     "children with text",
			xml:      `<a><b>hello</b><c>fred</c><c>wilma</c></a>`,
			expected: `{"b":[{"#text":["hello"]}],"c":[{"#text":["fred"]},{"#text":["wilma"]}]}`,
		},
		{
			name:     "text and attributes",
			xml:      `<a id="first">hello, world!</a>`,
			expected: `{"#text":["hello, world!"],"@id":"first"}`,
		},
		{
			name:     "text and attributes and children",
			xml:      `<a id="first"><b id="second">hello</b><c id="third">fred</c><c>wilma</c><c id="last"/></a>`,
			expected: `{"@id":"first","b":[{"#text":["hello"],"@id":"second"}],"c":[{"#text":["fred"],"@id":"third"},{"#text":["wilma"]},{"@id":"last"}]}`,
		},
		{
			name:     "mixed text and children",
			xml:      `<a>hello <b>fred</b> and <b>wilma</b></a>`,
			expected: `{"#text":["hello","and"],"b":[{"#text":["fred"]},{"#text":["wilma"]}]}`,
		},
	} {
		name := fmt.Sprintf("%d %s %s", idx, test.name, test.nsFlag)
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			e := json.NewEncoder(&b)
			e.SetEscapeHTML(false)
			selector := test.selector
			if selector == "" {
				selector = "/"
			}
			mapper := xmlpicker.SimpleMapper{}
			var actualErr error
			parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.SimpleSelector(test.selector))
			parser.NSFlag = test.nsFlag
			for {
				_, n, err := parser.Next()
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
