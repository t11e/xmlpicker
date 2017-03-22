# ingest

# Usage

To convert one or more XML files to a JSON stream:

```
ingest selector file...
```

Where selector is a simple XML path matcher that determines which
nodes are converted to JSON objects.

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

Convert just office nodes:
```sh
ingest /listing/offices/office example.xml
```
```json
{
  "id": [
    {
      "#text": "123"
    }
  ]
}
{
  "id": [
    {
      "#text": "124"
    }
  ]
}
```

Convert the root node:
```sh
ingest /listing example.xml
```
```json
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
