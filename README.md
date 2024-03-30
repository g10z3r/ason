<p align="center"><img height="200" src="https://raw.githubusercontent.com/g10z3r/ason/main/assets/banner.png"></p>

<h1 align="center">ASON</h1>
<p align="center">Go AST to JSON, with support for converting back to Go code </p>

The library is a tool for serializing abstract syntax trees (AST) in Go and deserializing them back into Go code. This tool is designed to simplify working with AST in Go, enabling easy conversion of code structures into a serialized format and vice versa. This provides a convenient mechanism for analyzing, modifying, and generating Go code.

# Serialize Mode

```golang
pass := ason.NewSerPass(fset, SkipComments)
```

#### `Default`

Serialize everything


#### `CacheRef`

Cache large nodes. Use this mode when you carry out some manual manipulations with the source AST tree. For example, you duplicate nodes, which can create nodes that have the
same references to the original object in memory. Only large nodes can be cached, such as specifications, types and declarations.

#### `SkipComments`

Will not serialize any comments declarations

#### `ResolveObject`

ResolveObject identifiers to objects

#### `ResolveScope` 

ResolveScope file and package scope and identifiers objects

#### `ResolveLoc`

ResolveLoc allow to resolve start and end position for all AST nodes.


# Example 

## Serialize

Full example: example/deserialize/main.go

### Target

```golang
/*
Hello World
*/
var test2 = "test2Value"

```

### Code 

```golang 
import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"log"

	"github.com/g10z3r/ason"
)

func main() {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "../testdata/main.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	pass := ason.NewSerPass(fset)
	fileAstJson := ason.SerializeFile(pass, f)

	jsonData, err := json.Marshal(fileAstJson)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonData))
}
```

### Output 

```json
{
  "Doc": null,
  "Name": {
    "NamePos": { "Location": [8, 1, 9], "Filename": "../testdata/main.go" },
    "Name": "main",
    "Obj": null,
    "_ref": 0,
    "_type": "Ident",
    "_loc": null
  },
  "Decls": [
    {
      "Doc": {
        "List": [
          {
            "Slash": {
              "Location": [14, 3, 1],
              "Filename": "../testdata/main.go"
            },
            "Text": "/*\nHello World\n*/",
            "_ref": 0,
            "_type": "Comment",
            "_loc": null
          }
        ],
        "_ref": 0,
        "_type": "CommentGroup",
        "_loc": null
      },
      "TokenPos": { "Location": [32, 6, 1], "Filename": "../testdata/main.go" },
      "Tok": "var",
      "Lparen": 0,
      "Specs": [
        {
          "Doc": null,
          "Names": [
            {
              "NamePos": {
                "Location": [36, 6, 5],
                "Filename": "../testdata/main.go"
              },
              "Name": "test2",
              "Obj": null,
              "_ref": 0,
              "_type": "Ident",
              "_loc": null
            }
          ],
          "Type": null,
          "Values": [
            {
              "ValuePos": {
                "Location": [44, 6, 13],
                "Filename": "../testdata/main.go"
              },
              "Kind": "STRING",
              "Value": "\"test2Value\"",
              "_ref": 0,
              "_type": "BasicLit",
              "_loc": null
            }
          ],
          "Comment": null,
          "_ref": 0,
          "_type": "ValueSpec",
          "_loc": null
        }
      ],
      "Rparen": 0,
      "_ref": 0,
      "_type": "GenDecl",
      "_loc": null
    }
  ],
  "Size": 57,
  "FileStart": { "Location": [0, 1, 1], "Filename": "../testdata/main.go" },
  "FileEnd": { "Location": [57, 6, 26], "Filename": "../testdata/main.go" },
  "Scope": null,
  "Imports": null,
  "Unresolved": null,
  "Package": { "Location": [0, 1, 1], "Filename": "../testdata/main.go" },
  "Comments": [
    {
      "List": [
        {
          "Slash": {
            "Location": [14, 3, 1],
            "Filename": "../testdata/main.go"
          },
          "Text": "/*\nHello World\n*/",
          "_ref": 0,
          "_type": "Comment",
          "_loc": null
        }
      ],
      "_ref": 0,
      "_type": "CommentGroup",
      "_loc": null
    }
  ],
  "GoVersion": "",
  "_ref": 0,
  "_type": "File",
  "_loc": null
}
```

## Deserialize 

Full example: example/deserialize/main.go

### Code 

```golang
func main(){
    file, err := serialize("../testdata/main.go")
	if err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()
	pass := ason.NewDePass(fset)
	f, err := ason.DeserializeFile(pass, file)
	if err != nil {
		log.Fatal(err)
	}
}
```

### Output

```golang
package main /*
Hello World
*/
var test2 = "test2Value"
```