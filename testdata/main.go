package main

import (
	"fmt"
	"go/ast"
)

/*
Hello World
*/
var test2 = "test2Value"

var (
	// Some doc here
	test3 = "test3Value"
	test4 = "test4Value"
)

// var testArray = [5]int{1, 2, 3, 4, 5}

type serConf int

const (
	CACHE_REF serConf = iota << 2
	FILE_SCOPE
	PKG_SCOPE
	IDENT_OBJ
	LOC
)

const (
	CACHE_REF = 1
	FILE_SCOPE
	PKG_SCOPE
	IDENT_OBJ
	LOC
)

func serializeImports(pass *serPass, inputMap map[string]*ast.Object) map[string]*Object {

	result := make(map[string]*Object, len(inputMap))
	for k, v := range inputMap {
		result[k] = SerializeObject(pass, v)
	}

	fmt.Println("test")

	return result
}
