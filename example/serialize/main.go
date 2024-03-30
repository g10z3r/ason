package main

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
