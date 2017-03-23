package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/t11e/xmlpicker"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("usage: xmlpicker selector file...")
	}
	selector := xmlpicker.SimpleSelector(args[0])
	xmlImporter := xmlpicker.DefaultXMLImporter{}
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	if err := run(selector, xmlImporter, args); err != nil {
		panic(err)
	}
}

func run(selector xmlpicker.Selector, xmlImporter xmlpicker.XMLImporter, args []string) error {
	for _, arg := range args {
		if err := withReader(arg, func(r io.Reader) error {
			return autoDecompress(r, func(r io.Reader) error {
				w := os.Stdout
				e := json.NewEncoder(w)
				e.SetEscapeHTML(false)
				return xmlpicker.Process(r, selector, func(_ xmlpicker.Path, n xmlpicker.Node) error {
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
