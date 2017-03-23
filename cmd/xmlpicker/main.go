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
	raw, err := open(filename)
	if err != nil {
		return err
	}
	defer raw.Close()
	reader, err := autoDecompress(raw)
	if err != nil {
		return err
	}
	defer reader.Close()
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

// Opens the filename for reading, uses stdin if it is "-" the returned Reader should be closed.
func open(filename string) (io.ReadCloser, error) {
	if filename == "-" {
		return ioutil.NopCloser(os.Stdin), nil
	}
	return os.Open(filename)
}

// Wraps the reader to decompress if the gzip header is detected, the returned Reader should be closed.
func autoDecompress(source io.Reader) (io.ReadCloser, error) {
	br := bufio.NewReader(source)
	h, err := br.Peek(2)
	if err != nil {
		return nil, err
	}
	if h[0] != 0x1f || h[1] != 0x8b {
		return ioutil.NopCloser(br), nil
	}
	return gzip.NewReader(br)
}
