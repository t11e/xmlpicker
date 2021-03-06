package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/t11e/xmlpicker"
)

type cmds struct {
	jsonCmd `command:"json" description:"convert to JSON"`
	xmlCmd  `command:"xml" description:"convert to XML"`
}

type options struct {
	Selector  string `short:"s" long:"selector" default:"/" description:"path selector to describe which nodes are exported"`
	Namespace string `short:"n" long:"namespace" choice:"expand" choice:"strip" choice:"prefix" default:"prefix" description:"how to handle namespaces"`
}

func (o *options) NewSelector() xmlpicker.Selector {
	return xmlpicker.PathSelector(o.Selector)
}

func (o *options) NSFlag() xmlpicker.NSFlag {
	switch o.Namespace {
	case "strip":
		return xmlpicker.NSStrip
	case "expand":
		return xmlpicker.NSExpand
	case "prefix":
		return xmlpicker.NSPrefix
	}
	panic("Bad namespace: " + o.Namespace)
}

type jsonCmd struct {
	Options options
	Pretty  bool `short:"p" long:"pretty" description:"generated formatted JSON"`
	Args    struct {
		Filenames []string `required:"1" positional-arg-name:"file"`
	} `positional-args:"yes"`
}

func (c *jsonCmd) Execute(_ []string) error {
	p := newJSONProcessor(os.Stdout)
	if c.Pretty {
		p.encoder.SetIndent("", "    ")
	}
	return mainImpl(&c.Options, c.Args.Filenames, p)
}

type xmlCmd struct {
	Options           options
	Pretty            bool   `short:"p" long:"pretty" description:"generated formatted XML"`
	ContainerXml      string `long:"container-xml" description:"xml container for output elements, if empty output each one in its original position"`
	ContainerSelector string `long:"container-selector" description:"used to find the first matching path in --container-xml' when generating the output, the rest of container-xml is ignored"`
	Args              struct {
		Filenames []string `required:"1" positional-arg-name:"file"`
	} `positional-args:"yes"`
}

func (c *xmlCmd) Execute(_ []string) error {
	p := newXMLProcessor(os.Stdout)
	var err error
	p.containerNode, err = c.createContainerNode()
	if err != nil {
		return err
	}
	if c.Pretty {
		p.exporter.Encoder.Indent("", "    ")
	}
	return mainImpl(&c.Options, c.Args.Filenames, p)
}

func (c *xmlCmd) createContainerNode() (*xmlpicker.Node, error) {
	if c.ContainerXml == "" {
		return nil, nil
	}
	r := strings.NewReader(c.ContainerXml)
	decoder := xml.NewDecoder(r)
	decoder.Strict = true
	//TODO Add dependency on "golang.org/x/net/html/charset" for more charset support
	//decoder.CharsetReader = charset.NewReaderLabel
	parser := xmlpicker.NewParser(decoder, xmlpicker.PathSelector(c.ContainerSelector))
	parser.NSFlag = c.Options.NSFlag()
	node, err := parser.Next()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func main() {
	parser := flags.NewParser(&cmds{}, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if _, ok := err.(*flags.Error); ok {
			os.Exit(2)
		}
		panic(err)
	}
}

func mainImpl(o *options, fs []string, proc processor) error {
	if err := proc.Begin(); err != nil {
		return err
	}
	for _, f := range fs {
		if err := parse(f, o, proc); err != nil {
			return err
		}
	}
	return proc.Finish()
}

func parse(filename string, o *options, proc processor) error {
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
	parser := xmlpicker.NewParser(decoder, o.NewSelector())
	parser.NSFlag = o.NSFlag()
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
		n.Parent = nil // ensure parser doesn't care if we overwrite this value
	}
	return nil
}

type processor interface {
	Begin() error
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

func (p *jsonProcessor) Begin() error {
	return nil
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
	writer        io.Writer
	exporter      *xmlpicker.XMLExporter
	containerNode *xmlpicker.Node
}

func (p *xmlProcessor) Begin() error {
	if p.containerNode != nil {
		if err := p.exporter.StartPath(p.containerNode); err != nil {
			return err
		}
	}
	return nil
}

func (p *xmlProcessor) Process(node *xmlpicker.Node) error {
	if p.containerNode == nil {
		if err := p.exporter.StartPath(node.Parent); err != nil {
			return err
		}
	} else {
		node.Parent = p.containerNode
	}
	if err := p.exporter.EncodeNode(node); err != nil {
		return err
	}
	if p.containerNode == nil {
		if err := p.exporter.EndPath(node.Parent); err != nil {
			return err
		}
		// must flush here to allow us to send the newline directly to the writer afterward
		if err := p.exporter.Encoder.Flush(); err != nil {
			return err
		}
		if _, err := p.writer.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func (p *xmlProcessor) Finish() error {
	if p.containerNode != nil {
		if err := p.exporter.EndPath(p.containerNode); err != nil {
			return err
		}
	}
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
