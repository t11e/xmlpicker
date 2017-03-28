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
