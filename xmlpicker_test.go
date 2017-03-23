package xmlpicker_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t11e/xmlpicker"
)

func TestSimpleSelector(t *testing.T) {
	for idx, test := range []struct {
		selector string
		xml      string
		expected []string
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
	} {
		t.Run(fmt.Sprintf("%d %s", idx, test.selector), func(t *testing.T) {
			actual := make([]string, 0)
			selector := test.selector
			err := xmlpicker.Process(strings.NewReader(test.xml), xmlpicker.SimpleSelector(selector), func(path xmlpicker.Path, _ xmlpicker.Node) error {
				actual = append(actual, path.String())
				return nil
			})
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual, "[%d] %s\nXML:\n%s\n", idx, test.selector, test.xml)
		})
	}
}

func TestDefaultXMLImporter(t *testing.T) {
	for idx, test := range []struct {
		name        string
		selector    string
		xml         string
		expected    string
		expectedErr string
	}{
		{
			name:     "control",
			xml:      `<a/>`,
			expected: `{}`,
		},
		{
			name:        "invalid",
			xml:         `<a>`,
			expectedErr: "XML syntax error on line 1: unexpected EOF",
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
		t.Run(fmt.Sprintf("%d %s", idx, test.name), func(t *testing.T) {
			var b bytes.Buffer
			e := json.NewEncoder(&b)
			e.SetEscapeHTML(false)
			selector := test.selector
			if selector == "" {
				selector = "/"
			}
			xmlImporter := xmlpicker.DefaultXMLImporter{}
			err := xmlpicker.Process(strings.NewReader(test.xml), xmlpicker.SimpleSelector(selector), func(_ xmlpicker.Path, n xmlpicker.Node) error {
				v := xmlImporter.ImportXML(n)
				return e.Encode(v)
			})
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr, "[%d] %s\nXML:\n%s\n", idx, test.name, test.xml)
			} else {
				assert.NoError(t, err, "[%d] %s\nXML:\n%s\n", idx, test.name, test.xml)
			}
			actual := strings.TrimSuffix(b.String(), "\n")
			assert.Equal(t, test.expected, actual, "[%d] %s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", idx, test.name, test.xml, test.expected, actual)
		})
	}
}
