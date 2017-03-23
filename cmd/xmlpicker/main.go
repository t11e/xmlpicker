package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/t11e/xmlpicker"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: xmlpicker selector file...")
		os.Exit(1)
	}
	selector := xmlpicker.SimpleSelector(args[0])
	mapper := xmlpicker.SimpleMapper{}
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	for _, filename := range args {
		if err := process(filename, selector, mapper, encoder); err != nil {
			panic(err)
		}
	}
}

func process(filename string, selector xmlpicker.Selector, mapper xmlpicker.Mapper, encoder *json.Encoder) error {
	raw, shouldClose, err := open(filename)
	if err != nil {
		return err
	}
	if shouldClose {
		defer raw.Close()
	}
	reader, shouldClose, err := autoDecompress(raw)
	if err != nil {
		return err
	}
	if shouldClose {
		defer reader.Close()
	}
	decoder := xml.NewDecoder(reader)
	decoder.Strict = true
	//TODO Add dependency on "golang.org/x/net/html/charset" for more charset support
	//decoder.CharsetReader = charset.NewReaderLabel
	parser := xmlpicker.NewParser(decoder, selector)
	for {
		_, n, err := parser.Next()
		if err == xmlpicker.EOF {
			break
		}
		if err != nil {
			return err
		}
		v, err := mapper.FromNode(n)
		if err != nil {
			return err
		}
		encoder.Encode(v)
	}
	return nil
}

// Opens the filename for reading, uses stdin if it is "-" and returns true if the caller should close the returned Reader.
func open(filename string) (io.ReadCloser, bool, error) {
	if filename == "-" {
		return os.Stdin, false, nil
	}
	f, err := os.Open(filename)
	return f, true, err
}

// Wraps the reader to decompress if the gzip header is detected and returns true if the caller should close the returned Reader.
func autoDecompress(source io.Reader) (io.ReadCloser, bool, error) {
	br := bufio.NewReader(source)
	h, err := br.Peek(2)
	if err != nil {
		return nil, false, err
	}
	if h[0] != 0x1f || h[1] != 0x8b {
		return ioutil.NopCloser(br), false, nil
	}
	gr, err := gzip.NewReader(br)
	if err != nil {
		return nil, false, err
	}
	return gr, true, err
}
