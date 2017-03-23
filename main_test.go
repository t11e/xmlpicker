package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXmlToJson(t *testing.T) {
	for idx, test := range []struct {
		name        string
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
		t.Run(fmt.Sprintf("%d %s", idx, test.name), func(tt *testing.T) {
			var b bytes.Buffer
			e := json.NewEncoder(&b)
			e.SetEscapeHTML(false)
			err := xmlparts(strings.NewReader(test.xml), "/a", func(row map[string]interface{}) error {
				return e.Encode(row)
			})
			actual := strings.TrimSuffix(b.String(), "\n")
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr, "[%d] %s\nXML:\n%s\n", idx, test.name, test.xml)
			} else {
				assert.NoError(t, err, "[%d] %s\nXML:\n%s\n", idx, test.name, test.xml)
			}
			assert.Equal(t, test.expected, actual, "[%d] %s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", idx, test.name, test.xml, test.expected, actual)
		})
	}
}
