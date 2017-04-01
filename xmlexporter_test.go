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
			name: "deeper 0",
			xml: `
				<a id="1">
				  one
				  <b id="2">two<c id="3">three</c>four</b>
				  five
				  <b id="4">six<c id="5">seven</c>eight</b>
				  nine
				</a>`,
			selector: "/",
			expected: `<a id="1">one<b id="2">two<c id="3">three</c>four</b>five<b id="4">six<c id="5">seven</c>eight</b>nine</a>`,
		},
		{
			name: "deeper 1",
			xml: `
				<a id="1">
				  one
				  <b id="2">two<c id="3">three</c>four</b>
				  five
				  <b id="4">six<c id="5">seven</c>eight</b>
				  nine
				</a>`,
			selector: "/*/",
			expected: `` +
				`<a id="1"><b id="2">two<c id="3">three</c>four</b></a>` +
				`<a id="1"><b id="4">six<c id="5">seven</c>eight</b></a>`,
		},
		{
			name: "deeper 2",
			xml: `
				<a id="1">
				  one
				  <b id="2">two<c id="3">three</c>four</b>
				  five
				  <b id="4">six<c id="5">seven</c>eight</b>
				  nine
				</a>`,
			selector: "/*/*/",
			expected: `` +
				`<a id="1"><b id="2"><c id="3">three</c></b></a>` +
				`<a id="1"><b id="4"><c id="5">seven</c></b></a>`,
		},
	} {
		t.Run(fmt.Sprintf("%d %s", idx, test.name), func(t *testing.T) {
			for _, nsFlag := range []xmlpicker.NSFlag{xmlpicker.NSExpand, xmlpicker.NSStrip, xmlpicker.NSPrefix} {
				t.Run(nsFlag.String(), func(t *testing.T) {
					name := fmt.Sprintf("%d %s %s", idx, test.name, nsFlag)
					var b bytes.Buffer
					e := xmlpicker.XMLExporter{Encoder: xml.NewEncoder(&b)}
					var actualErr error
					parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector(test.selector))
					parser.NSFlag = nsFlag
					for {
						n, err := parser.Next()
						if err == io.EOF {
							e.Encoder.Flush()
							break
						}
						if err != nil {
							actualErr = err
							break
						}
						if err := e.StartPath(n.Parent); err != nil {
							actualErr = err
							break
						}
						if err := e.EncodeNode(n); err != nil {
							actualErr = err
							break
						}
						if err := e.EndPath(n.Parent); err != nil {
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
		})
	}
}

func TestXMLExporter_Namespaces(t *testing.T) {
	type scenario struct {
		nsFlag      xmlpicker.NSFlag
		expected    string
		expectedErr string
	}
	for idx, test := range []struct {
		name      string
		selector  string
		xml       string
		scenarios []scenario
	}{
		{
			name:     "root namespace",
			xml:      `<a xmlns:a="http://example.com/x" foo="1" a:bar="2"/>`,
			selector: "/",
			scenarios: []scenario{
				{
					nsFlag:   xmlpicker.NSExpand,
					expected: `<a foo="1" xmlns:x="http://example.com/x" x:bar="2"></a>`,
				},
				{
					nsFlag:   xmlpicker.NSStrip,
					expected: `<a foo="1" bar="2"></a>`,
				},
				{
					nsFlag:   xmlpicker.NSPrefix,
					expected: `<a foo="1" a:bar="2" xmlns:a="http://example.com/x"></a>`,
				},
			},
		},
		{
			name: "namespaces",
			xml: `
				<a xmlns="http://example.com/y" xmlns:a="http://example.com/x" foo="1" a:bar="2">
				  <b id="123" foo="3" a:bar="4">first</b>
				  <b id="456" foo="5">second</b>
				</a>`,
			selector: "/*/",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<a xmlns="http://example.com/y" foo="1" xmlns:x="http://example.com/x" x:bar="2"><b id="123" foo="3" x:bar="4">first</b></a>` +
						`<a xmlns="http://example.com/y" foo="1" xmlns:x="http://example.com/x" x:bar="2"><b id="456" foo="5">second</b></a>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<a foo="1" bar="2"><b id="123" foo="3" bar="4">first</b></a>` +
						`<a foo="1" bar="2"><b id="456" foo="5">second</b></a>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<a foo="1" a:bar="2" xmlns="http://example.com/y" xmlns:a="http://example.com/x"><b id="123" foo="3" a:bar="4">first</b></a>` +
						`<a foo="1" a:bar="2" xmlns="http://example.com/y" xmlns:a="http://example.com/x"><b id="456" foo="5">second</b></a>`,
				},
			},
		},

		{
			name: "always bad prefix",
			xml: `
				<a>
				  <b id="123" a:foo="1">first</b>
				  <b id="456" b:foo="2">second</b>
				  <b id="789" c:foo="3">third</b>
				</a>`,
			selector: "/*/",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<a><b id="123" xmlns:a="a" a:foo="1">first</b></a>` +
						`<a><b id="456" xmlns:b="b" b:foo="2">second</b></a>` +
						`<a><b id="789" xmlns:c="c" c:foo="3">third</b></a>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<a><b id="123" foo="1">first</b></a>` +
						`<a><b id="456" foo="2">second</b></a>` +
						`<a><b id="789" foo="3">third</b></a>`,
				},
				{
					nsFlag:      xmlpicker.NSPrefix,
					expectedErr: "xmlpicker: undeclared prefix a at /a/b",
				},
			},
		},
		{
			name: "sometimes bad prefix",
			xml: `
				<a>
				  <b id="123" a:foo="1">first</b>
				  <b id="456" b:foo="2" xmlns:b="http://example.com/x">second</b>
				  <b id="789" c:foo="3">third</b>
				</a>`,
			selector: "/*/",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<a><b id="123" xmlns:a="a" a:foo="1">first</b></a>` +
						`<a><b id="456" xmlns:x="http://example.com/x" x:foo="2">second</b></a>` +
						`<a><b id="789" xmlns:c="c" c:foo="3">third</b></a>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<a><b id="123" foo="1">first</b></a>` +
						`<a><b id="456" foo="2">second</b></a>` +
						`<a><b id="789" foo="3">third</b></a>`,
				},
				{
					nsFlag:      xmlpicker.NSPrefix,
					expectedErr: "xmlpicker: undeclared prefix a at /a/b",
				},
			},
		},

		// Examples taken from https://www.w3.org/TR/xml-names/
		{
			name: "namespace scoping",
			xml: `
				<html:html xmlns:html='http://www.w3.org/1999/xhtml'>
				  <html:head><html:title>Frobnostication</html:title></html:head>
				  <html:body><html:p>Moved to
				    <html:a href='http://frob.example.com'>here.</html:a></html:p></html:body>
				</html:html>`,
			selector: "/html/*/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Frobnostication</title></head></html>` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><body><p>Moved to<a href="http://frob.example.com">here.</a></p></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<html><head><title>Frobnostication</title></head></html>` +
						`<html><body><p>Moved to<a href="http://frob.example.com">here.</a></p></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<html:html xmlns:html="http://www.w3.org/1999/xhtml"><html:head><html:title>Frobnostication</html:title></html:head></html:html>` +
						`<html:html xmlns:html="http://www.w3.org/1999/xhtml"><html:body><html:p>Moved to<html:a href="http://frob.example.com">here.</html:a></html:p></html:body></html:html>`,
				},
			},
		},
		{
			name: "both namespace prefixes are available throughout",
			xml: `
				<bk:book xmlns:bk='urn:loc.gov:books'
					 xmlns:isbn='urn:ISBN:0-395-36341-6'>
				    <bk:title>Cheaper by the Dozen</bk:title>
				    <isbn:number>1568491379</isbn:number>
				</bk:book>`,
			selector: "/book/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<book xmlns="urn:loc.gov:books"><title>Cheaper by the Dozen</title></book>` +
						`<book xmlns="urn:loc.gov:books"><number xmlns="urn:ISBN:0-395-36341-6">1568491379</number></book>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<book><title>Cheaper by the Dozen</title></book>` +
						`<book><number>1568491379</number></book>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<bk:book xmlns:bk="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><bk:title>Cheaper by the Dozen</bk:title></bk:book>` +
						`<bk:book xmlns:bk="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><isbn:number>1568491379</isbn:number></bk:book>`,
				},
			},
		},
		{
			name: "elements are in the HTML namespace, in this case by default",
			xml: `
				<html xmlns='http://www.w3.org/1999/xhtml'>
				  <head><title>Frobnostication</title></head>
				  <body><p>Moved to
				    <a href='http://frob.example.com'>here</a>.</p></body>
				</html>`,
			selector: "/html/*/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Frobnostication</title></head></html>` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><body><p>Moved to<a href="http://frob.example.com">here</a>.</p></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<html><head><title>Frobnostication</title></head></html>` +
						`<html><body><p>Moved to<a href="http://frob.example.com">here</a>.</p></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Frobnostication</title></head></html>` +
						`<html xmlns="http://www.w3.org/1999/xhtml"><body><p>Moved to<a href="http://frob.example.com">here</a>.</p></body></html>`,
				},
			},
		},
		{
			name: "unprefixed element types are from 'books'",
			xml: `
				<book xmlns='urn:loc.gov:books'
				  xmlns:isbn='urn:ISBN:0-395-36341-6'>
				  <title>Cheaper by the Dozen</title>
				  <isbn:number>1568491379</isbn:number>
				</book>`,
			selector: "/book/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<book xmlns="urn:loc.gov:books"><title>Cheaper by the Dozen</title></book>` +
						`<book xmlns="urn:loc.gov:books"><number xmlns="urn:ISBN:0-395-36341-6">1568491379</number></book>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<book><title>Cheaper by the Dozen</title></book>` +
						`<book><number>1568491379</number></book>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<book xmlns="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><title>Cheaper by the Dozen</title></book>` +
						`<book xmlns="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><isbn:number>1568491379</isbn:number></book>`,
				},
			},
		},
		{
			name: "a larger example of namespace scoping",
			xml: `
				<!-- initially, the default namespace is "books" -->
				<book xmlns='urn:loc.gov:books'
				  xmlns:isbn='urn:ISBN:0-395-36341-6'>
				  <title>Cheaper by the Dozen</title>
				  <isbn:number>1568491379</isbn:number>
				  <notes>
				    <!-- make HTML the default namespace for some commentary -->
				    <p xmlns='http://www.w3.org/1999/xhtml'>
				      This is a <i>funny</i> book!
				    </p>
				  </notes>
				</book>`,
			selector: "/book/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<book xmlns="urn:loc.gov:books"><title>Cheaper by the Dozen</title></book>` +
						`<book xmlns="urn:loc.gov:books"><number xmlns="urn:ISBN:0-395-36341-6">1568491379</number></book>` +
						`<book xmlns="urn:loc.gov:books"><notes><p xmlns="http://www.w3.org/1999/xhtml">This is a<i>funny</i>book!</p></notes></book>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<book><title>Cheaper by the Dozen</title></book>` +
						`<book><number>1568491379</number></book>` +
						`<book><notes><p>This is a<i>funny</i>book!</p></notes></book>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<book xmlns="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><title>Cheaper by the Dozen</title></book>` +
						`<book xmlns="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><isbn:number>1568491379</isbn:number></book>` +
						`<book xmlns="urn:loc.gov:books" xmlns:isbn="urn:ISBN:0-395-36341-6"><notes><p xmlns="http://www.w3.org/1999/xhtml">This is a<i>funny</i>book!</p></notes></book>`,
				},
			},
		},
		{
			name: "empty default namespace override",
			xml: `
				<Beers>
				  <!-- the default namespace inside tables is that of HTML -->
				  <table xmlns='http://www.w3.org/1999/xhtml'>
				   <th><td>Name</td><td>Origin</td><td>Description</td></th>
				   <tr>
				     <!-- no default namespace inside table cells -->
				     <td><brandName xmlns="">Huntsman</brandName></td>
				     <td><origin xmlns="">Bath, UK</origin></td>
				     <td>
				       <details xmlns=""><class>Bitter</class><hop>Fuggles</hop>
					 <pro>Wonderful hop, light alcohol, good summer beer</pro>
					 <con>Fragile; excessive variance pub to pub</con>
					 </details>
					</td>
				      </tr>
				    </table>
				  </Beers>`,
			selector: "/*/table/tr/td",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><brandName>Huntsman</brandName></td></tr></table></Beers>` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><origin>Bath, UK</origin></td></tr></table></Beers>` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><details><class>Bitter</class><hop>Fuggles</hop><pro>Wonderful hop, light alcohol, good summer beer</pro><con>Fragile; excessive variance pub to pub</con></details></td></tr></table></Beers>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<Beers><table><tr><td><brandName>Huntsman</brandName></td></tr></table></Beers>` +
						`<Beers><table><tr><td><origin>Bath, UK</origin></td></tr></table></Beers>` +
						`<Beers><table><tr><td><details><class>Bitter</class><hop>Fuggles</hop><pro>Wonderful hop, light alcohol, good summer beer</pro><con>Fragile; excessive variance pub to pub</con></details></td></tr></table></Beers>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><brandName xmlns="">Huntsman</brandName></td></tr></table></Beers>` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><origin xmlns="">Bath, UK</origin></td></tr></table></Beers>` +
						`<Beers><table xmlns="http://www.w3.org/1999/xhtml"><tr><td><details xmlns=""><class>Bitter</class><hop>Fuggles</hop><pro>Wonderful hop, light alcohol, good summer beer</pro><con>Fragile; excessive variance pub to pub</con></details></td></tr></table></Beers>`,
				},
			},
		},
		{
			name: "uniqueness of attributes",
			xml: `
				<!-- http://www.w3.org is bound to n1 and is the default -->
				<x xmlns:n1="http://www.w3.org"
				   xmlns="http://www.w3.org" >
				  <good a="1"     b="2" />
				  <good a="1"     n1:a="2" />
				</x>`,
			selector: "/x/*",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<x xmlns="http://www.w3.org"><good a="1" b="2"></good></x>` +
						`<x xmlns="http://www.w3.org"><good a="1" xmlns:www.w3.org="http://www.w3.org" www.w3.org:a="2"></good></x>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<x><good a="1" b="2"></good></x>` +
						`<x><good a="1" a="2"></good></x>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<x xmlns="http://www.w3.org" xmlns:n1="http://www.w3.org"><good a="1" b="2"></good></x>` +
						`<x xmlns="http://www.w3.org" xmlns:n1="http://www.w3.org"><good a="1" n1:a="2"></good></x>`,
				},
			},
		},

		// Real world XML example copied from https://www.xml.com/pub/a/1999/01/namespaces.html
		{
			name: "html",
			xml: `
				<h:html xmlns:xdc="http://www.xml.com/books"
					xmlns:h="http://www.w3.org/HTML/1998/html4">
				 <h:head><h:title>Book Review</h:title></h:head>
				 <h:body>
				  <xdc:bookreview>
				   <xdc:title>XML: A Primer</xdc:title>
				   <h:table>
				    <h:tr align="center">
				     <h:td>Author</h:td><h:td>Price</h:td>
				     <h:td>Pages</h:td><h:td>Date</h:td></h:tr>
				    <h:tr align="left">
				     <h:td><xdc:author>Simon St. Laurent</xdc:author></h:td>
				     <h:td><xdc:price>31.98</xdc:price></h:td>
				     <h:td><xdc:pages>352</xdc:pages></h:td>
				     <h:td><xdc:date>1998/01</xdc:date></h:td>
				    </h:tr>
				   </h:table>
				  </xdc:bookreview>
				 </h:body>
				</h:html>`,
			selector: "bookreview/table/tr",
			scenarios: []scenario{
				{
					nsFlag: xmlpicker.NSExpand,
					expected: `` +
						`<html xmlns="http://www.w3.org/HTML/1998/html4"><body><bookreview xmlns="http://www.xml.com/books"><table xmlns="http://www.w3.org/HTML/1998/html4"><tr align="center"><td>Author</td><td>Price</td><td>Pages</td><td>Date</td></tr></table></bookreview></body></html>` +
						`<html xmlns="http://www.w3.org/HTML/1998/html4"><body><bookreview xmlns="http://www.xml.com/books"><table xmlns="http://www.w3.org/HTML/1998/html4"><tr align="left"><td><author xmlns="http://www.xml.com/books">Simon St. Laurent</author></td><td><price xmlns="http://www.xml.com/books">31.98</price></td><td><pages xmlns="http://www.xml.com/books">352</pages></td><td><date xmlns="http://www.xml.com/books">1998/01</date></td></tr></table></bookreview></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSStrip,
					expected: `` +
						`<html><body><bookreview><table><tr align="center"><td>Author</td><td>Price</td><td>Pages</td><td>Date</td></tr></table></bookreview></body></html>` +
						`<html><body><bookreview><table><tr align="left"><td><author>Simon St. Laurent</author></td><td><price>31.98</price></td><td><pages>352</pages></td><td><date>1998/01</date></td></tr></table></bookreview></body></html>`,
				},
				{
					nsFlag: xmlpicker.NSPrefix,
					expected: `` +
						`<h:html xmlns:h="http://www.w3.org/HTML/1998/html4" xmlns:xdc="http://www.xml.com/books"><h:body><xdc:bookreview><h:table><h:tr align="center"><h:td>Author</h:td><h:td>Price</h:td><h:td>Pages</h:td><h:td>Date</h:td></h:tr></h:table></xdc:bookreview></h:body></h:html>` +
						`<h:html xmlns:h="http://www.w3.org/HTML/1998/html4" xmlns:xdc="http://www.xml.com/books"><h:body><xdc:bookreview><h:table><h:tr align="left"><h:td><xdc:author>Simon St. Laurent</xdc:author></h:td><h:td><xdc:price>31.98</xdc:price></h:td><h:td><xdc:pages>352</xdc:pages></h:td><h:td><xdc:date>1998/01</xdc:date></h:td></h:tr></h:table></xdc:bookreview></h:body></h:html>`,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%d %s", idx, test.name), func(t *testing.T) {
			for _, scenario := range test.scenarios {
				t.Run(scenario.nsFlag.String(), func(t *testing.T) {
					name := fmt.Sprintf("%d %s %s", idx, test.name, scenario.nsFlag)
					var b bytes.Buffer
					e := xmlpicker.XMLExporter{Encoder: xml.NewEncoder(&b)}
					var actualErr error
					parser := xmlpicker.NewParser(xml.NewDecoder(strings.NewReader(test.xml)), xmlpicker.PathSelector(test.selector))
					parser.NSFlag = scenario.nsFlag
					for {
						n, err := parser.Next()
						if err == io.EOF {
							e.Encoder.Flush()
							break
						}
						if err != nil {
							actualErr = err
							break
						}
						if err := e.StartPath(n.Parent); err != nil {
							actualErr = err
							break
						}
						if err := e.EncodeNode(n); err != nil {
							actualErr = err
							break
						}
						if err := e.EndPath(n.Parent); err != nil {
							actualErr = err
							break
						}
					}
					if scenario.expectedErr != "" {
						assert.EqualError(t, actualErr, scenario.expectedErr, "%s\nXML:\n%s\n", name, test.xml)
					} else {
						assert.NoError(t, actualErr, "%s\nXML:\n%s\n", name, test.xml)
					}
					actual := strings.TrimSuffix(b.String(), "\n")
					assert.Equal(t, scenario.expected, actual, "%s\nXML:\n%s\nExpected:\n%s\nActual:\n%s\n", name, test.xml, scenario.expected, actual)
				})
			}
		})
	}
}
