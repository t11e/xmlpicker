# xmlpicker

Wraps an `xml.Decoder` to support picking out smaller chunks from very large XML files
where each indidual chunk can be held in memory for processing.

# Usage

To convert one or more XML files to a JSON or XML stream:

# Example

Input file:
```xml
<listing id="123">
  <offices count="2">
    <office>
      <id>123</id>
    </office>
    <office>
      <id>124</id>
    </office>
  </offices>
</listing>
```

Select the entire document as JSON:
```sh
xmlpicker json --pretty example.xml
```
```json
{
    "@id": "123",
    "_name": "listing",
    "_namespaces": {},
    "offices": [
        {
            "@count": "2",
            "office": [
                {
                    "id": [
                        {
                            "#text": [
                                "123"
                            ]
                        }
                    ]
                },
                {
                    "id": [
                        {
                            "#text": [
                                "124"
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}
```

Select just office nodes as JSON:
```sh
xmlpicker json --pretty --selector /listing/offices/office example.xml
```
```json
{
    "_name": "office",
    "_namespaces": {},
    "id": [
        {
            "#text": [
                "123"
            ]
        }
    ]
}
```
```json
{
    "_name": "office",
    "_namespaces": {},
    "id": [
        {
            "#text": [
                "124"
            ]
        }
    ]
}
```

Select the entire document as XML:
```sh
xmlpicker xml --pretty example.xml
```
```xml
<listing id="123">
    <offices count="2">
        <office>
            <id>123</id>
        </office>
        <office>
            <id>124</id>
        </office>
    </offices>
</listing>
```


Select just office nodes as XML:
```sh
 xmlpicker xml --pretty --selector /listing/offices/office example.xml
```
```xml
<listing id="123">
    <offices count="2">
        <office>
            <id>123</id>
        </office>
    </offices>
</listing>
```
```xml
<listing id="123">
    <offices count="2">
        <office>
            <id>124</id>
        </office>
    </offices>
</listing>
```

By default, the `xmlpicker` tool preserves namespace prefixes from the original XML file. You can override this with
the `--namespace=` option. Possible values are:
 
 * `--namespace=prefix` preserve the original namespace with their prefixes 
 * `--namespace=strip` strip out any namespace information
 * `--namespace=expand` preserve just the namespace values, drop their prefixes

# Contributions

Clone this repository into your GOPATH and use [Glide](https://github.com/Masterminds/glide) to install its dependencies.

```sh
brew install glide
go get github.com/t11e/xmlpicker
cd "$GOPATH"/src/github.com/t11e/xmlpicker
glide install --strip-vendor
```

You can then run the tests:

```sh
go test $(go list ./... | grep -v /vendor/)
```

To install the commands into `$GOPATH/bin/`:

```sh
go install ./cmd/...
```

# License

MIT. See [LICENSE](LICENSE) file.
