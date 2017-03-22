# ingest

# Usage

To convert one or more XML files to a JSON stream:

```
ingest level file...
```

Where level is the depth in the XML from which elements
are converted to JSON objects. A level of 0 will treat
the root element as a single JSON object. A level of 1
will treat each child of the root element as a JSON object.

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

Convert just level 1 nodes:
```sh
ingest 1 example.xml
```
```json
{
  "offices": [
    {
      "@count": "2",
      "office": [
        {
          "id": [
            {
              "#text": "123"
            }
          ]
        },
        {
          "id": [
            {
              "#text": "124"
            }
          ]
        }
      ]
    }
  ]
}
```

Convert the whole file:
```sh
ingest 0 example.xml
```
```json
{
  "listing": [
    {
      "@id": "123",
      "offices": [
        {
          "@count": "2",
          "office": [
            {
              "id": [
                {
                  "#text": "123"
                }
              ]
            },
            {
              "id": [
                {
                  "#text": "124"
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

# Contributions

Clone this repository into your GOPATH.

```sh
go get github.com/t11e/ingest
cd "$GOPATH"/src/github.com/t11e/ingest
```

You can then run the tests:

```sh
go test
```

To install the commands into `$GOPATH/bin/`:

```sh
go install
```
