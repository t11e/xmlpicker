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
