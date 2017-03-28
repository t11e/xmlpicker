package xmlpicker_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t11e/xmlpicker"
)

func TestPathSelector(t *testing.T) {
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
