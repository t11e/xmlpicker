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
		fmt.Fprintln(os.Stderr, "usage: xmlpicker json|xml selector file...")
		os.Exit(1)
	}
	var w io.Writer
	w = os.Stdout
	var proc processor
	outformat := args[0]
	switch outformat {
	case "json":
		jp := newJSONProcessor(w)
		// TODO Override any settings on jp
		proc = jp
	case "xml":
		xp := newXMLProcessor(w)
		// TODO Override any settings on xp
		proc = xp
	default:
		panic("invalid outformat")
	}
	args = args[1:]
	s := xmlpicker.PathSelector(args[0])
	args = args[1:]
	if len(args) == 0 {
		args = []string{"-"}
	}
	if err := mainImpl(s, args, proc); err != nil {
		panic(err)
	}
}

func mainImpl(s xmlpicker.Selector, fs []string, proc processor) error {
	for _, f := range fs {
		if err := parse(f, s, proc); err != nil {
			return err
		}
	}
	return proc.Finish()
}

func parse(filename string, selector xmlpicker.Selector, proc processor) error {
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
		n, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := proc.Process(n); err != nil {
			return err
		}
	}
	return nil
}

type processor interface {
	Process(node *xmlpicker.Node) error
	Finish() error
}

func newJSONProcessor(w io.Writer) *jsonProcessor {
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	return &jsonProcessor{
		encoder: e,
		mapper:  xmlpicker.SimpleMapper{},
	}
}

type jsonProcessor struct {
	encoder *json.Encoder
	mapper  xmlpicker.Mapper
}

func (p *jsonProcessor) Process(node *xmlpicker.Node) error {
	v, err := p.mapper.FromNode(node)
	if err != nil {
		return err
	}
	return p.encoder.Encode(v)
}

func (p *jsonProcessor) Finish() error {
	return nil
}

func newXMLProcessor(w io.Writer) *xmlProcessor {
	return &xmlProcessor{
		writer:   w,
		exporter: &xmlpicker.XMLExporter{Encoder: xml.NewEncoder(w)},
	}
}

type xmlProcessor struct {
	writer   io.Writer
	exporter *xmlpicker.XMLExporter
}

func (p *xmlProcessor) Process(node *xmlpicker.Node) error {
	if err := p.exporter.EncodeNode(node); err != nil {
		return err
	}
	// must flush here to allow us to send the newline directly to the writer afterward
	if err := p.exporter.Encoder.Flush(); err != nil {
		return err
	}
	if _, err := p.writer.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func (p *xmlProcessor) Finish() error {
	return p.exporter.Encoder.Flush()
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
