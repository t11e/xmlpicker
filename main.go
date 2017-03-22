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
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("usage: ingest level file...")
	}
	startDepth, err := strconv.Atoi(args[0])
	if err != nil {
		panic(err)
	}
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	if err := run(startDepth, args); err != nil {
		panic(err)
	}
}

func run(startDepth int, args []string) error {
	for _, arg := range args {
		if err := withReader(arg, func(r io.Reader) error {
			return autoDecompress(r, func(r io.Reader) error {
				w := os.Stdout
				e := json.NewEncoder(w)
				e.SetEscapeHTML(false)
				return xmlparts(r, startDepth, func(v map[string]interface{}) error {
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
	name     xml.Name
	attrs    []xml.Attr
	children []*node
}

func (n node) export() map[string]interface{} {
	out := make(map[string]interface{})
	if n.name.Local == "#text" {
		out["#text"] = []string{n.attrs[0].Value}
		return out
	}
	for _, a := range n.attrs {
		out[fmt.Sprintf("@%s", a.Name.Local)] = a.Value
	}
	if len(n.children) == 1 && n.children[0].name.Local == "#text" {
		out["#text"] = n.children[0].attrs[0].Value
	}
	for _, c := range n.children {
		var t []interface{}
		key := c.name.Local
		if key == "#text" {
			continue
		}
		if prev, ok := out[key]; ok {
			t = prev.([]interface{})
		} else {
			t = make([]interface{}, 0)
			out[key] = t
		}
		value := c.export()
		out[key] = append(t, value)
	}
	return out
}

type nodes []*node

func (s nodes) String() string {
	var b bytes.Buffer
	for _, f := range s {
		b.WriteRune('/')
		b.WriteString(f.name.Local)
	}
	return b.String()
}

func xmlparts(r io.Reader, startDepth int, yield func(map[string]interface{}) error) error {
	d := xml.NewDecoder(r)
	//TODO Add dependency on "golang.org/x/net/html/charset" for more charset support
	//d.CharsetReader = charset.NewReaderLabel
	stack := make(nodes, 0)
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
			cur := node{
				name:  t.Name,
				attrs: t.Attr,
			}
			stack = append(stack, &cur)
			if len(stack) > maxDepth {
				return errors.New("too many xml levels")
			}
			if len(stack) >= 2 && len(stack)-1 > startDepth {
				prev := stack[len(stack)-2]
				prev.children = append(prev.children, &cur)
				if len(prev.children) > maxChildren {
					return errors.New("too many child xml elements")
				}
			}
		case xml.EndElement:
			prev := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) == startDepth {
				t := node{children: []*node{prev}}
				e := t.export()
				yield(e)
			}
		case xml.CharData:
			if len(stack) < 1 || len(stack)-1 < startDepth {
				continue
			}
			t = t.Copy()
			s := string(t)
			s = strings.TrimSpace(s)
			if len(s) == 0 {
				continue
			}
			if false {
				continue
			}
			cur := stack[len(stack)-1]
			text := node{
				name: xml.Name{Local: "#text"},
				attrs: []xml.Attr{
					{
						Value: s,
					},
				},
			}
			cur.children = append(cur.children, &text)
			if len(cur.children) > maxChildren {
				return errors.New("too many child xml elements")
			}
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
			return fmt.Errorf("unexpected xml token %+v", t)
		}
	}
	if len(stack) != 0 {
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
