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
	selector := args[0]
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	if err := run(selector, args); err != nil {
		panic(err)
	}
}

func run(selector string, args []string) error {
	for _, arg := range args {
		if err := withReader(arg, func(r io.Reader) error {
			return autoDecompress(r, func(r io.Reader) error {
				w := os.Stdout
				e := json.NewEncoder(w)
				e.SetEscapeHTML(false)
				return xmlparts(r, selector, func(v map[string]interface{}) error {
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

type node struct {
	elem     *xml.StartElement
	text     string
	parent   *node
	children []*node
}

func (n node) export() map[string]interface{} {
	out := make(map[string]interface{})
	if n.elem == nil {
		out["#text"] = []string{n.text}
		return out
	}
	for _, a := range n.elem.Attr {
		out[fmt.Sprintf("@%s", a.Name.Local)] = a.Value
	}
	for _, c := range n.children {
		var key string
		var value interface{}
		if c.elem == nil {
			key = "#text"
			value = c.text
		} else {
			key = c.elem.Name.Local
			value = c.export()
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

type path []xml.StartElement

func (p path) String() string {
	var b bytes.Buffer
	for _, t := range p {
		b.WriteRune('/')
		b.WriteString(t.Name.Local)
	}
	return b.String()
}

func xmlparts(r io.Reader, selector string, yield func(map[string]interface{}) error) error {
	d := xml.NewDecoder(r)
	//TODO Add dependency on "golang.org/x/net/html/charset" for more charset support
	//d.CharsetReader = charset.NewReaderLabel
	path := make(path, 0)
	var n *node
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
				if selector != path.String() {
					continue
				}
				n = &node{elem: &t}
				continue
			}
			c := &node{elem: &t}
			c.parent = n
			n.children = append(n.children, c)
			if len(n.children) > maxChildren {
				return errors.New("too many child xml elements")
			}
			n = c
		case xml.EndElement:
			path = path[:len(path)-1]
			if n == nil {
				continue
			}
			if n.parent == nil {
				yield(n.export())
			}
			n = n.parent
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
			c := &node{text: s}
			c.parent = n
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
