package xmlpicker_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t11e/xmlpicker"
)

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
			expected: `<a id="1"><b id="2">two<c id="3">three</c>four</b></a>` +
				`<a id="1"><b id="4">six<c id="5">seven</c>eight</b></a>`,
		},
		{
			name:     "deeper 2",
			xml:      `<a id="1">one<b id="2">two<c id="3">three</c>four</b>five<b id="4">six<c id="5">seven</c>eight</b>nine</a>`,
			selector: "/*/*/c",
			expected: `<a id="1"><b id="2"><c id="3">three</c></b></a>` +
				`<a id="1"><b id="4"><c id="5">seven</c></b></a>`,
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
			expected: `<a xmlns:a="aaa" foo="1" a:bar="2"></a>`,
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
			expected: `<a xmlns="DEF" xmlns:a="aaa" foo="1" a:bar="2"><b id="123" foo="3" a:bar="4">first</b></a>` +
				`<a xmlns="DEF" xmlns:a="aaa" foo="1" a:bar="2"><b id="456" foo="5">second</b></a>`,
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
				if err := xmlpicker.XMLExport(e, n, test.nsFlag); err != nil {
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
