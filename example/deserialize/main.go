package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"

	"github.com/g10z3r/ason"
)

func main() {
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

	code, err := generate(fset, f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(code)
}

func generate(fset *token.FileSet, file *ast.File) (string, error) {
	var buf bytes.Buffer

	conf := printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: 2,
	}

	err := conf.Fprint(&buf, fset, file)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func serialize(path string) (*ason.File, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pass := ason.NewSerPass(fset)
	return ason.SerializeFile(pass, f), nil
}
