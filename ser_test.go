package ason

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathToAsonTestDataDir = "./testdata/main.go"
)

const (
	testName = "testName"
	testType = "testType"
)

var (
	testValueSpecs = []*ast.ValueSpec{
		{
			Names: []*ast.Ident{{Name: testName}},
			Type:  &ast.Ident{Name: testType},
			Values: []ast.Expr{
				&ast.BinaryExpr{
					X: &ast.Ident{Name: "iota"},
					Y: &ast.BasicLit{Kind: token.INT, Value: "2"},
				},
			},
		},
		{
			Names:  []*ast.Ident{{Name: testName}},
			Type:   nil,
			Values: nil,
		},
	}
)

func mockReadFileWithErr(name string) ([]byte, error) {
	return nil, errors.New("test error")
}

func TestNewSerPass(t *testing.T) {
	t.Run("valid: creation of pass without any params", func(t *testing.T) {
		fset := token.NewFileSet()
		pass := NewSerPass(fset)

		assert.NotNil(t, pass)
		assert.NotNil(t, pass.fset)
		assert.Nil(t, pass.refCache)
	})

	t.Run("valid: creation of pass with CACHE_REF", func(t *testing.T) {
		fset := token.NewFileSet()
		pass := NewSerPass(fset, CacheRef)

		assert.NotNil(t, pass)
		assert.NotNil(t, pass.fset)
		assert.NotNil(t, pass.refCache)
		assert.NotNil(t, true, pass.conf[CacheRef])
	})
}

func TestWithRefLookup(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		pass := NewSerPass(token.NewFileSet(), CacheRef)
		astNode := &ast.Ident{Name: testName}
		ident := SerializeIdent(pass, astNode)
		pass.refCache[astNode] = NewWeakRef(ident)

		// Panic at this point means that inside the function there was a call
		// to the function to the serializer, which we passed as nil.
		//Calling this function means that the link could not be found in the cache,
		// although it should have been there.
		assert.NotPanics(t, func() {
			SerRefLookup[*ast.Ident, *Ident](pass, astNode, nil)
		}, "unexpected: should not try to call serialize func")

		assert.Equal(t, ident, SerRefLookup[*ast.Ident, *Ident](pass, astNode, nil))
	})

	t.Run("invalid: searching for a link that is not in the cache", func(t *testing.T) {
		pass := NewSerPass(token.NewFileSet(), CacheRef)
		astNode := &ast.Ident{Name: testName}

		// The absence of panic in this case means that an unknown link was found
		// that for some reason corresponds to the one you were looking for,
		// although you did not save it before
		assert.Panics(t, func() {
			SerRefLookup[*ast.Ident, *Ident](pass, astNode, nil)
		}, "unexpected: should call serialize func")
	})
}

func TestSerializeOption(t *testing.T) {
	t.Run("valid: with non-nil input", func(t *testing.T) {
		pass := &serPass{}
		input := &ast.Ident{Name: testName}
		result := SerializeOption(pass, input, SerializeIdent)
		assert.NotNil(t, result)
	})

	t.Run("invalid: with nil input", func(t *testing.T) {
		pass := &serPass{}
		result := SerializeOption(pass, nil, SerializeIdent)
		assert.Nil(t, result)
	})

	t.Run("invalid: with typed nil input", func(t *testing.T) {
		pass := &serPass{}
		var ident *ast.Ident
		result := SerializeOption(pass, ident, SerializeIdent)
		assert.Nil(t, result)
	})
}

func TestCalcFileSize(t *testing.T) {
	t.Run("valid: calculation of file size", func(t *testing.T) {
		f, fset := uploadTestData(t, pathToAsonTestDataDir)
		pass := NewSerPass(fset)
		size := calcFileSize(pass, f)
		assert.Greater(t, size, 0)
	})
}

func TestSerializePos(t *testing.T) {
	t.Run("valid: correct creation of serPass", func(t *testing.T) {
		f, fset := uploadTestData(t, pathToAsonTestDataDir)
		pass := NewSerPass(fset)
		pos := SerializePos(pass, f.FileStart)
		assert.NotNil(t, pos)
		assert.IsType(t, &Position{}, pos)
	})

	t.Run("invalid: got token.NoPos", func(t *testing.T) {
		pass := NewSerPass(token.NewFileSet())
		pos := SerializePos(pass, token.NoPos)
		assert.NotNil(t, pos)
		assert.Equal(t, pos, new(NoPos))
	})
}

func uploadTestData(t *testing.T, pathToTestData string) (*ast.File, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	require.NotNil(t, fset)

	f, err := parser.ParseFile(fset, pathToTestData, nil, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, f)

	return f, fset
}

// ----------------- Comments ----------------- //

func TestSerializeComment(t *testing.T) {
	t.Run("valid: comment creation", func(t *testing.T) {
		input := &ast.Comment{
			Slash: token.Pos(10),
			Text:  "Example comment",
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeComment(pass, input)
		assert.Equal(t, "Example comment", actual.Text)
		assert.Equal(t, NodeTypeComment, actual.Node.Type)
	})
}

func TestSerializeCommentGroup(t *testing.T) {
	t.Run("valid: comment group creation", func(t *testing.T) {
		input := &ast.CommentGroup{
			List: []*ast.Comment{
				{Slash: token.Pos(10), Text: "First comment"},
				{Slash: token.Pos(20), Text: "Second comment"},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeCommentGroup(pass, input)

		assert.Equal(t, 2, len(actual.List))
		assert.Equal(t, NodeTypeCommentGroup, actual.Node.Type)
	})
}

// ----------------- Expressions ----------------- //

func TestSerializeIdent(t *testing.T) {
	t.Run("valid: ident creation", func(t *testing.T) {
		input := &ast.Ident{
			NamePos: token.Pos(10),
			Name:    testName,
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeIdent(pass, input)

		assert.Equal(t, testName, actual.Name)
		assert.Equal(t, NodeTypeIdent, actual.Node.Type)
	})
}

func TestSerializeBasicLit(t *testing.T) {
	t.Run("valid: basic lit creation", func(t *testing.T) {
		input := &ast.BasicLit{
			ValuePos: token.Pos(10),
			Kind:     token.INT,
			Value:    "10",
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeBasicLit(pass, input)

		assert.Equal(t, token.INT.String(), actual.Kind)
		assert.Equal(t, "10", actual.Value)
		assert.Equal(t, NodeTypeBasicLit, actual.Node.Type)
	})
}

func TestSerializeFuncLit(t *testing.T) {
	t.Run("valid: func lit creation", func(t *testing.T) {
		input := &ast.FuncLit{
			Type: &ast.FuncType{},
			Body: &ast.BlockStmt{},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeFuncLit(pass, input)

		assert.NotNil(t, actual.Type)
		assert.NotNil(t, actual.Body)
		assert.Equal(t, NodeTypeFuncLit, actual.Node.Type)
	})
}

func TestSerializeCompositeLit(t *testing.T) {
	t.Run("valid: ast.CompositeLit serialization", func(t *testing.T) {
		input := &ast.CompositeLit{
			Lbrace: token.Pos(10),
			Elts:   []ast.Expr{&ast.BasicLit{Value: "foo"}},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeCompositeLit(pass, input)

		assert.Equal(t, 1, len(actual.Elts))
		assert.Equal(t, "foo", actual.Elts[0].(*BasicLit).Value)
		assert.Equal(t, NodeTypeCompositeLit, actual.Node.Type)
	})
}

func TestSerializeField(t *testing.T) {
	t.Run("valid: ast.Field serialization", func(t *testing.T) {
		name := &ast.Ident{Name: "foo"}
		input := &ast.Field{Names: []*ast.Ident{name}}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeField(pass, input)

		assert.Equal(t, 1, len(actual.Names))
		assert.Equal(t, "foo", actual.Names[0].Name)
		assert.Equal(t, NodeTypeField, actual.Node.Type)
	})
}

func TestSerializeFieldList(t *testing.T) {
	t.Run("valid: ast.FieldList serialization", func(t *testing.T) {
		input := &ast.FieldList{}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeFieldList(pass, input)

		assert.Equal(t, NodeTypeFieldList, actual.Node.Type)
	})
}

func TestSerializeEllipsis(t *testing.T) {
	t.Run("valid: ast.Ellipsis serialization", func(t *testing.T) {
		input := &ast.Ellipsis{}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeEllipsis(pass, input)

		assert.Equal(t, NodeTypeEllipsis, actual.Node.Type)
	})
}

func TestSerializeBadExpr(t *testing.T) {
	t.Run("valid: ast.BadExpr serialization", func(t *testing.T) {
		input := &ast.BadExpr{
			From: token.Pos(10),
			To:   token.Pos(20),
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeBadExpr(pass, input)

		assert.Equal(t, NodeTypeBadExpr, actual.Node.Type)
	})
}

func TestSerializeParenExpr(t *testing.T) {
	t.Run("valid: ast.ParenExpr serialization", func(t *testing.T) {
		input := &ast.ParenExpr{
			X: &ast.Ident{Name: "x"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeParenExpr(pass, input)

		assert.Equal(t, "x", actual.X.(*Ident).Name)
		assert.Equal(t, NodeTypeParenExpr, actual.Node.Type)
	})
}

func TestSerializeSelectorExpr(t *testing.T) {
	t.Run("valid: ast.SelectorExpr serialization", func(t *testing.T) {
		input := &ast.SelectorExpr{
			X:   &ast.Ident{Name: "x"},
			Sel: &ast.Ident{Name: "y"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeSelectorExpr(pass, input)

		assert.Equal(t, "x", actual.X.(*Ident).Name)
		assert.Equal(t, "y", actual.Sel.Name)
		assert.Equal(t, NodeTypeSelectorExpr, actual.Node.Type)
	})
}

func TestSerializeIndexExpr(t *testing.T) {
	t.Run("valid: ast.IndexExpr serialization", func(t *testing.T) {
		input := &ast.IndexExpr{
			X:     &ast.Ident{Name: "arr"},
			Index: &ast.BasicLit{Kind: token.INT, Value: "3"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeIndexExpr(pass, input)

		assert.Equal(t, "arr", actual.X.(*Ident).Name)
		assert.Equal(t, "3", actual.Index.(*BasicLit).Value)
		assert.Equal(t, NodeTypeIndexExpr, actual.Node.Type)
	})
}

func TestSerializeIndexListExpr(t *testing.T) {
	t.Run("valid: ast.IndexListExpr serialization", func(t *testing.T) {
		input := &ast.IndexListExpr{
			X: &ast.Ident{Name: "x"},
			Indices: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "1"},
				&ast.BasicLit{Kind: token.INT, Value: "2"},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeIndexListExpr(pass, input)

		assert.Equal(t, "x", actual.X.(*Ident).Name)
		assert.Len(t, actual.Indices, 2)
		assert.Equal(t, "1", actual.Indices[0].(*BasicLit).Value)
		assert.Equal(t, "2", actual.Indices[1].(*BasicLit).Value)
		assert.Equal(t, NodeTypeIndexListExpr, actual.Node.Type)
	})
}

func TestSerializeSliceExpr(t *testing.T) {
	t.Run("valid: ast.SliceExpr serialization", func(t *testing.T) {
		input := &ast.SliceExpr{
			X:    &ast.Ident{Name: "slice"},
			Low:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			High: &ast.BasicLit{Kind: token.INT, Value: "5"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeSliceExpr(pass, input)

		assert.Equal(t, "slice", actual.X.(*Ident).Name)
		assert.Equal(t, "1", actual.Low.(*BasicLit).Value)
		assert.Equal(t, "5", actual.High.(*BasicLit).Value)
		assert.Equal(t, NodeTypeSliceExpr, actual.Node.Type)
	})
}

func TestSerializeTypeAssertExpr(t *testing.T) {
	t.Run("valid: ast.TypeAssertExpr serialization", func(t *testing.T) {
		input := &ast.TypeAssertExpr{
			X:    &ast.Ident{Name: "x"},
			Type: &ast.Ident{Name: "int"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeTypeAssertExpr(pass, input)

		assert.Equal(t, "x", actual.X.(*Ident).Name)
		assert.Equal(t, "int", actual.Type.(*Ident).Name)
		assert.Equal(t, NodeTypeTypeAssertExpr, actual.Node.Type)
	})
}

func TestSerializeCallExpr(t *testing.T) {
	t.Run("valid: ast.CallExpr serialization", func(t *testing.T) {
		input := &ast.CallExpr{
			Fun:  &ast.Ident{Name: "foo"},
			Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "42"}},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeCallExpr(pass, input)

		assert.Equal(t, "foo", actual.Fun.(*Ident).Name)
		assert.Len(t, actual.Args, 1)
		assert.Equal(t, "42", actual.Args[0].(*BasicLit).Value)
		assert.Equal(t, NodeTypeCallExpr, actual.Node.Type)
	})
}

func TestSerializeStarExpr(t *testing.T) {
	t.Run("valid: ast.StarExpr serialization", func(t *testing.T) {
		input := &ast.StarExpr{
			X: &ast.Ident{Name: "x"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeStarExpr(pass, input)

		assert.Equal(t, "x", actual.X.(*Ident).Name)
		assert.Equal(t, NodeTypeStarExpr, actual.Node.Type)
	})
}

func TestSerializeUnaryExpr(t *testing.T) {
	t.Run("valid: ast.UnaryExpr serialization", func(t *testing.T) {
		input := &ast.UnaryExpr{
			Op: token.NOT,
			X:  &ast.BasicLit{Kind: token.INT, Value: "5"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeUnaryExpr(pass, input)

		assert.Equal(t, "!", actual.Op)
		assert.Equal(t, "5", actual.X.(*BasicLit).Value)
		assert.Equal(t, NodeTypeUnaryExpr, actual.Node.Type)
	})
}

func TestSerializeBinaryExpr(t *testing.T) {
	t.Run("valid: ast.BinaryExpr serialization", func(t *testing.T) {
		input := &ast.BinaryExpr{
			X:  &ast.BasicLit{Kind: token.INT, Value: "5"},
			Op: token.ADD,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "3"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeBinaryExpr(pass, input)

		assert.Equal(t, "5", actual.X.(*BasicLit).Value)
		assert.Equal(t, "+", actual.Op)
		assert.Equal(t, "3", actual.Y.(*BasicLit).Value)
		assert.Equal(t, NodeTypeBinaryExpr, actual.Node.Type)
	})
}

func TestSerializeKeyValueExpr(t *testing.T) {
	t.Run("valid: ast.KeyValueExpr serialization", func(t *testing.T) {
		input := &ast.KeyValueExpr{
			Key:   &ast.Ident{Name: "key"},
			Value: &ast.BasicLit{Kind: token.STRING, Value: "\"value\""},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeKeyValueExpr(pass, input)

		assert.Equal(t, "key", actual.Key.(*Ident).Name)
		assert.Equal(t, "\"value\"", actual.Value.(*BasicLit).Value)
		assert.Equal(t, NodeTypeKeyValueExpr, actual.Node.Type)
	})
}

// ----------------- Types ----------------- //

func TestSerializeArrayType(t *testing.T) {
	t.Run("valid: ast.ArrayType serialization", func(t *testing.T) {
		input := &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: "5"},
			Elt: &ast.Ident{Name: "int"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeArrayType(pass, input)

		assert.Equal(t, "5", actual.Len.(*BasicLit).Value)
		assert.Equal(t, "int", actual.Elt.(*Ident).Name)
		assert.Equal(t, NodeTypeArrayType, actual.Node.Type)
	})
}

func TestSerializeStructType(t *testing.T) {
	t.Run("valid: ast.StructType serialization", func(t *testing.T) {
		input := &ast.StructType{
			Fields: &ast.FieldList{
				List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeStructType(pass, input)

		assert.Len(t, actual.Fields.List, 1)
		assert.Equal(t, "int", actual.Fields.List[0].Type.(*Ident).Name)
		assert.Equal(t, NodeTypeStructType, actual.Node.Type)
	})
}

func TestSerializeFuncType(t *testing.T) {
	t.Run("valid: ast.FuncType serialization", func(t *testing.T) {
		input := &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: &ast.Ident{Name: "bool"}}},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeFuncType(pass, input)

		assert.Len(t, actual.Params.List, 1)
		assert.Equal(t, "int", actual.Params.List[0].Type.(*Ident).Name)
		assert.Len(t, actual.Results.List, 1)
		assert.Equal(t, "bool", actual.Results.List[0].Type.(*Ident).Name)
		assert.Equal(t, NodeTypeFuncType, actual.Node.Type)
	})
}

func TestSerializeInterfaceType(t *testing.T) {
	t.Run("valid: ast.InterfaceType serialization", func(t *testing.T) {
		input := &ast.InterfaceType{
			Methods: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "MethodA"}},
					{Type: &ast.Ident{Name: "MethodB"}},
				},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeInterfaceType(pass, input)

		assert.Len(t, actual.Methods.List, 2)
		assert.Equal(t, "MethodA", actual.Methods.List[0].Type.(*Ident).Name)
		assert.Equal(t, "MethodB", actual.Methods.List[1].Type.(*Ident).Name)
		assert.Equal(t, NodeTypeInterfaceType, actual.Node.Type)
	})
}

func TestSerializeMapType(t *testing.T) {
	t.Run("valid: ast.MapType serialization", func(t *testing.T) {
		input := &ast.MapType{
			Key:   &ast.Ident{Name: "string"},
			Value: &ast.Ident{Name: "int"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeMapType(pass, input)

		assert.Equal(t, "string", actual.Key.(*Ident).Name)
		assert.Equal(t, "int", actual.Value.(*Ident).Name)
		assert.Equal(t, NodeTypeMapType, actual.Node.Type)
	})
}

func TestSerializeChanType(t *testing.T) {
	t.Run("valid: ast.ChanType serialization", func(t *testing.T) {
		input := &ast.ChanType{
			Dir:   ast.SEND | ast.RECV,
			Value: &ast.Ident{Name: "int"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeChanType(pass, input)

		assert.Equal(t, int(ast.SEND|ast.RECV), actual.Dir)
		assert.Equal(t, "int", actual.Value.(*Ident).Name)
		assert.Equal(t, NodeTypeChanType, actual.Node.Type)
	})
}

func TestSerializeExpr(t *testing.T) {
	fset := token.NewFileSet()
	pass := NewSerPass(fset)

	t.Run("valid: case Ident", func(t *testing.T) {
		expr := &ast.Ident{Name: "x"}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*Ident)
		assert.True(t, ok)
	})

	t.Run("valid: case Ellipsis", func(t *testing.T) {
		expr := &ast.Ellipsis{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*Ellipsis)
		assert.True(t, ok)
	})

	t.Run("valid: case BasicLit", func(t *testing.T) {
		expr := &ast.BasicLit{Value: "42"}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*BasicLit)
		assert.True(t, ok)
	})

	t.Run("valid: case FuncLit", func(t *testing.T) {
		expr := &ast.FuncLit{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*FuncLit)
		assert.True(t, ok)
	})

	t.Run("valid: case CompositeLit", func(t *testing.T) {
		expr := &ast.CompositeLit{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*CompositeLit)
		assert.True(t, ok)
	})

	t.Run("valid: case BadExpr", func(t *testing.T) {
		expr := &ast.BadExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*BadExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case SelectorExpr", func(t *testing.T) {
		expr := &ast.SelectorExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*SelectorExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case ParenExpr", func(t *testing.T) {
		expr := &ast.ParenExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*ParenExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case IndexExpr", func(t *testing.T) {
		expr := &ast.IndexExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*IndexExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case IndexListExpr", func(t *testing.T) {
		expr := &ast.IndexListExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*IndexListExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case UnaryExpr", func(t *testing.T) {
		expr := &ast.UnaryExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*UnaryExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case SliceExpr", func(t *testing.T) {
		expr := &ast.SliceExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*SliceExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case TypeAssertExpr", func(t *testing.T) {
		expr := &ast.TypeAssertExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*TypeAssertExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case CallExpr", func(t *testing.T) {
		expr := &ast.CallExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*CallExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case StarExpr", func(t *testing.T) {
		expr := &ast.StarExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*StarExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case BinaryExpr", func(t *testing.T) {
		expr := &ast.BinaryExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*BinaryExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case KeyValueExpr", func(t *testing.T) {
		expr := &ast.KeyValueExpr{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*KeyValueExpr)
		assert.True(t, ok)
	})

	t.Run("valid: case ArrayType", func(t *testing.T) {
		expr := &ast.ArrayType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*ArrayType)
		assert.True(t, ok)
	})

	t.Run("valid: case StructType", func(t *testing.T) {
		expr := &ast.StructType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*StructType)
		assert.True(t, ok)
	})

	t.Run("valid: case FuncType", func(t *testing.T) {
		expr := &ast.FuncType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*FuncType)
		assert.True(t, ok)
	})

	t.Run("Serialize InterfaceType", func(t *testing.T) {
		expr := &ast.InterfaceType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*InterfaceType)
		assert.True(t, ok)
	})

	t.Run("Serialize MapType", func(t *testing.T) {
		expr := &ast.MapType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*MapType)
		assert.True(t, ok)
	})

	t.Run("Serialize ChanType", func(t *testing.T) {
		expr := &ast.ChanType{}
		result := SerializeExpr(pass, expr)
		_, ok := result.(*ChanType)
		assert.True(t, ok)
	})
}

// ----------------- Statements ----------------- //

func TestSerializeBadStmt(t *testing.T) {
	t.Run("valid: ast.BadStmt serialization", func(t *testing.T) {
		input := &ast.BadStmt{
			From: token.Pos(1),
			To:   token.Pos(5),
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeBadStmt(pass, input)

		assert.Equal(t, NodeTypeBadStmt, actual.Node.Type)
	})
}

func TestSerializeDeclStmt(t *testing.T) {
	t.Run("valid: ast.DeclStmt serialization", func(t *testing.T) {
		input := &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Specs: []ast.Spec{&ast.ImportSpec{Path: &ast.BasicLit{Value: "\"fmt\""}}},
			},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeDeclStmt(pass, input)

		assert.Equal(t, NodeTypeDeclStmt, actual.Node.Type)
		assert.IsType(t, new(GenDecl), actual.Decl)
	})
}

func TestSerializeEmptyStmt(t *testing.T) {
	t.Run("valid: ast.EmptyStmt serialization", func(t *testing.T) {
		input := &ast.EmptyStmt{
			Semicolon: token.Pos(1),
			Implicit:  true,
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeEmptyStmt(pass, input)

		assert.Equal(t, NodeTypeEmptyStmt, actual.Node.Type)
		// Дополнительные проверки для Semicolon и Implicit
	})
}

func TestSerializeLabeledStmt(t *testing.T) {
	t.Run("valid: ast.LabeledStmt serialization", func(t *testing.T) {
		input := &ast.LabeledStmt{
			Label: &ast.Ident{Name: "label"},
			Stmt:  &ast.EmptyStmt{},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeLabeledStmt(pass, input)

		assert.Equal(t, "label", actual.Label.Name)
		assert.Equal(t, NodeTypeLabeledStmt, actual.Node.Type)
		assert.IsType(t, new(EmptyStmt), actual.Stmt)
	})
}

func TestSerializeExprStmt(t *testing.T) {
	t.Run("valid: ast.ExprStmt serialization", func(t *testing.T) {
		input := &ast.ExprStmt{
			X: &ast.BasicLit{Kind: token.INT, Value: "42"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeExprStmt(pass, input)

		assert.Equal(t, "42", actual.X.(*BasicLit).Value)
		assert.Equal(t, NodeTypeExprStmt, actual.Node.Type)
	})
}

func TestSerializeSendStmt(t *testing.T) {
	t.Run("valid: ast.SendStmt serialization", func(t *testing.T) {
		input := &ast.SendStmt{
			Chan:  &ast.Ident{Name: "ch"},
			Value: &ast.BasicLit{Kind: token.INT, Value: "42"},
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeSendStmt(pass, input)

		assert.Equal(t, "ch", actual.Chan.(*Ident).Name)
		assert.Equal(t, "42", actual.Value.(*BasicLit).Value)
		assert.Equal(t, NodeTypeSendStmt, actual.Node.Type)
	})
}

func TestSerializeIncDecStmt(t *testing.T) {
	t.Run("valid: ast.IncDecStmt serialization", func(t *testing.T) {
		input := &ast.IncDecStmt{
			X:   &ast.Ident{Name: "counter"},
			Tok: token.INC,
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeIncDecStmt(pass, input)

		assert.Equal(t, "counter", actual.X.(*Ident).Name)
		assert.Equal(t, "++", actual.Tok)
		assert.Equal(t, NodeTypeIncDecStmt, actual.Node.Type)
	})
}

func TestSerializeAssignStmt(t *testing.T) {
	t.Run("valid: ast.AssignStmt serialization", func(t *testing.T) {
		input := &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: "x"}},
			Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "42"}},
			Tok: token.ASSIGN,
		}

		fset := token.NewFileSet()
		pass := NewSerPass(fset)
		actual := SerializeAssignStmt(pass, input)

		assert.Len(t, actual.Lhs, 1)
		assert.Equal(t, "x", actual.Lhs[0].(*Ident).Name)
		assert.Len(t, actual.Rhs, 1)
		assert.Equal(t, "42", actual.Rhs[0].(*BasicLit).Value)
		assert.Equal(t, "=", actual.Tok)
		assert.Equal(t, NodeTypeAssignStmt, actual.Node.Type)
	})
}
