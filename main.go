package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("usage: ingest selector file...")
	}
	selector := SimpleSelector(args[0])
	xmlImporter := DefaultXMLImporter{}
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	if err := run(selector, xmlImporter, args); err != nil {
		panic(err)
	}
}

func run(selector Selector, xmlImporter XMLImporter, args []string) error {
	for _, arg := range args {
		if err := withReader(arg, func(r io.Reader) error {
			return autoDecompress(r, func(r io.Reader) error {
				w := os.Stdout
				e := json.NewEncoder(w)
				e.SetEscapeHTML(false)
				return xmlparts(r, selector, func(_ Path, n Node) error {
					v := xmlImporter.ImportXML(n)
					if err := e.Encode(v); err != nil {
						return err
					}
					return nil
				})
			})
		}); err != nil {
			return err
		}
	}
	return nil
}

type Node struct {
	Element  *xml.StartElement
	text     string
	Parent   *Node
	children []*Node
}

type XMLImporter interface {
	ImportXML(node Node) map[string]interface{}
}

type DefaultXMLImporter struct {
}

func (xi DefaultXMLImporter) ImportXML(n Node) map[string]interface{} {
	out := make(map[string]interface{})
	if n.Element == nil {
		out["#text"] = []string{n.text}
		return out
	}
	for _, a := range n.Element.Attr {
		out[fmt.Sprintf("@%s", a.Name.Local)] = a.Value
	}
	for _, c := range n.children {
		var key string
		var value interface{}
		if c.Element == nil {
			key = "#text"
			value = c.text
		} else {
			key = c.Element.Name.Local
			value = xi.ImportXML(*c)
		}
		var values []interface{}
		if prev, ok := out[key]; ok {
			values = prev.([]interface{})
		} else {
			values = make([]interface{}, 0)
			out[key] = values
		}
		out[key] = append(values, value)
	}
	return out
}

type Path []xml.StartElement

func (p Path) String() string {
	var b bytes.Buffer
	b.WriteRune('/')
	for i, t := range p {
		if i > 0 {
			b.WriteRune('/')
		}
		b.WriteString(t.Name.Local)
	}
	return b.String()
}

type Selector interface {
	Matches(path Path) bool
}

func SimpleSelector(selector string) Selector {
	parts := strings.Split(selector, "/")
	if len(parts) > 1 && parts[0] == "" {
		parts = parts[1:]
	}
	matchLen := len(parts)
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return simpleSelector{parts, matchLen}
}

type simpleSelector struct {
	Parts    []string
	MatchLen int
}

func (s simpleSelector) Matches(path Path) bool {
	if len(path) != s.MatchLen {
		return false
	}
	for i, part := range s.Parts {
		p := path[i]
		if part != p.Name.Local {
			return false
		}
	}
	return true
}

func xmlparts(r io.Reader, selector Selector, yield func(Path, Node) error) error {
	d := xml.NewDecoder(r)
	//TODO Add dependency on "golang.org/x/net/html/charset" for more charset support
	//d.CharsetReader = charset.NewReaderLabel
	path := make(Path, 0)
	var n *Node
	const maxTokens = -1
	const maxDepth = 1000
	const maxChildren = 1000
	for c := 0; maxTokens < 0 || c < maxTokens; c = c + 1 {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch t := t.(type) {
		case xml.StartElement:
			t = t.Copy()
			path = append(path, t)
			if len(path) > maxDepth {
				return errors.New("too many xml levels")
			}
			if n == nil {
				if !selector.Matches(path) {
					continue
				}
				n = &Node{Element: &t}
				continue
			}
			c := &Node{Element: &t}
			c.Parent = n
			n.children = append(n.children, c)
			if len(n.children) > maxChildren {
				return errors.New("too many child xml elements")
			}
			n = c
		case xml.EndElement:
			if n != nil && n.Parent == nil {
				yield(path, *n)
			}
			path = path[:len(path)-1]
			if n == nil {
				continue
			}
			n = n.Parent
		case xml.CharData:
			if n == nil {
				continue
			}
			t = t.Copy()
			s := string(t)
			s = strings.TrimSpace(s)
			if len(s) == 0 {
				continue
			}
			c := &Node{text: s}
			c.Parent = n
			n.children = append(n.children, c)
			if len(n.children) > maxChildren {
				return errors.New("too many child xml elements")
			}
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
			return fmt.Errorf("unexpected xml token %+v", t)
		}
	}
	if len(path) != 0 {
		return errors.New("rest of file skipped")
	}
	return nil
}

// Opens the file name (or stdin if -) for reading
func withReader(name string, next func(io.Reader) error) error {
	var r io.Reader
	if name == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer func() {
			err := f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "problem closing %s: %s\n", name, err)
			}
		}()
		r = f
	}
	return next(r)
}

// Wraps the reader to decompress if the gzip header is detected
func autoDecompress(source io.Reader, next func(io.Reader) error) error {
	br := bufio.NewReader(source)
	h, err := br.Peek(2)
	if err != nil {
		return err
	}
	if h[0] != 0x1f || h[1] != 0x8b {
		return next(br)
	}
	gr, err := gzip.NewReader(br)
	if err != nil {
		return err
	}
	defer func() {
		err := gr.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "problem closing gzip br: %s\n", err)
		}
	}()
	return next(gr)
}

// Opens the file name (or stdout if -) for writing
func withWriter(name string, next func(io.Writer) error) error {
	var w io.Writer
	if name == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer func() {
			err := f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "problem closing %s: %s\n", name, err)
			}
		}()
		w = f
	}
	return next(w)
}

// Wraps the reader to decompress if the gzip header is detected
func compress(source io.Writer, next func(io.Writer) error) error {
	w := gzip.NewWriter(source)
	defer func() {
		err := w.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "problem closing gzip reader: %s\n", err)
		}
	}()
	return next(w)
}
