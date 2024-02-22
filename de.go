package ason

import (
	"fmt"
	"go/ast"
	"go/token"
	"unsafe"
)

type dePass struct {
	fset     *token.FileSet
	refCache map[Ason]*weakRef
	refCount uint
	conf     map[Mode]interface{}
}

func NewDePass(fset *token.FileSet) *dePass {
	return &dePass{
		fset: fset,
	}
}

type DeFn[I Ason, O ast.Node] func(*dePass, I) O

func DeRefLookup[I Ason, O ast.Node](pass *dePass, input I, deFn DeFn[I, O]) O {
	if weakRef, exists := pass.refCache[input]; exists {
		return weakRef.Load().(O)
	}

	node := deFn(pass, input)
	pass.refCache[input] = NewWeakRef(&node)
	pass.refCount++

	return node
}

func DeserializeOption[I Ason, R ast.Node](pass *dePass, input I, deFn DeFn[I, R]) (empty R) {
	if *(*interface{})(unsafe.Pointer(&input)) != nil {
		return deFn(pass, input)
	}

	return empty
}

func DeserializeList[I Ason, R ast.Node](pass *dePass, inputList []I, deFn DeFn[I, R]) (result []R) {
	if len(inputList) < 1 {
		return nil
	}

	for i := 0; i < len(inputList); i++ {
		val := DeserializeOption(pass, inputList[i], deFn)

		if *(*interface{})(unsafe.Pointer(&val)) != nil {
			result = append(result, val)
		}
	}

	return result
}

// ----------------- Scope ----------------- //

func DeserializePos(pass *dePass, input Pos) token.Pos {
	if valid := IsPosValid(input); !valid {
		return token.NoPos
	}

	tokPos := token.Pos(input.Offset())
	tokFile := pass.fset.File(tokPos)
	if tokFile == nil {
		return token.NoPos
	}

	return tokFile.Pos(input.Offset())
}

// ----------------- Comments ----------------- //

func DeserializeComment(pass *dePass, input *Comment) *ast.Comment {
	if pass.conf[SkipComments] != nil {
		return nil
	}

	return &ast.Comment{
		Slash: DeserializePos(pass, input.Slash),
		Text:  input.Text,
	}
}

func DeserializeCommentGroup(pass *dePass, input *CommentGroup) *ast.CommentGroup {
	if pass.conf[SkipComments] != nil {
		return nil
	}

	return &ast.CommentGroup{
		List: DeserializeList(pass, input.List, DeserializeComment),
	}
}

// ----------------- Expressions ----------------- //

func DeserializeField(pass *dePass, input *Field) *ast.Field {
	return &ast.Field{
		Doc:     DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
		Names:   DeserializeList(pass, input.Names, DeserializeIdent),
		Type:    DeserializeOption(pass, input.Type, DeserializeExpr),
		Tag:     DeserializeOption(pass, input.Tag, DeserializeBasicLit),
		Comment: DeserializeOption(pass, input.Comment, DeserializeCommentGroup),
	}
}

func DeserializeFieldList(pass *dePass, input *FieldList) *ast.FieldList {
	return &ast.FieldList{
		Opening: DeserializePos(pass, input.Opening),
		List:    DeserializeList(pass, input.List, DeserializeField),
		Closing: DeserializePos(pass, input.Closing),
	}
}

func DeserializeIdent(pass *dePass, input *Ident) *ast.Ident {
	return &ast.Ident{
		Name:    input.Name,
		NamePos: DeserializePos(pass, input.NamePos),
	}
}

func DeserializeBasicLit(pass *dePass, input *BasicLit) *ast.BasicLit {
	return &ast.BasicLit{
		ValuePos: DeserializePos(pass, input.ValuePos),
		Kind:     tokens[input.Kind],
		Value:    input.Value,
	}
}

func DeserializeFuncLit(pass *dePass, input *FuncLit) *ast.FuncLit {
	return &ast.FuncLit{
		Type: DeserializeOption(pass, input.Type, DeserializeFuncType),
		Body: DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeCompositeLit(pass *dePass, input *CompositeLit) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type:       DeserializeOption(pass, input.Type, DeserializeExpr),
		Lbrace:     DeserializePos(pass, input.Lbrace),
		Elts:       DeserializeList(pass, input.Elts, DeserializeExpr),
		Rbrace:     DeserializePos(pass, input.Rbrace),
		Incomplete: input.Incomplete,
	}
}

func DeserializeEllipsis(pass *dePass, input *Ellipsis) *ast.Ellipsis {
	return &ast.Ellipsis{
		Ellipsis: DeserializePos(pass, input.Ellipsis),
		Elt:      DeserializeOption(pass, input.Elt, DeserializeExpr),
	}
}

func DeserializeBadExpr(pass *dePass, input *BadExpr) *ast.BadExpr {
	return &ast.BadExpr{
		From: DeserializePos(pass, input.From),
		To:   DeserializePos(pass, input.To),
	}
}

func DeserializeParenExpr(pass *dePass, input *ParenExpr) *ast.ParenExpr {
	return &ast.ParenExpr{
		Lparen: DeserializePos(pass, input.Lparen),
		X:      DeserializeOption(pass, input.X, DeserializeExpr),
		Rparen: DeserializePos(pass, input.Rparen),
	}
}

func DeserializeSelectorExpr(pass *dePass, input *SelectorExpr) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   DeserializeOption(pass, input.X, DeserializeExpr),
		Sel: DeserializeOption(pass, input.Sel, DeserializeIdent),
	}
}

func DeserializeIndexExpr(pass *dePass, input *IndexExpr) *ast.IndexExpr {
	return &ast.IndexExpr{
		X:      DeserializeOption(pass, input.X, DeserializeExpr),
		Lbrack: DeserializePos(pass, input.Lbrack),
		Index:  DeserializeOption(pass, input.Index, DeserializeExpr),
		Rbrack: DeserializePos(pass, input.Rbrack),
	}
}

func DeserializeIndexListExpr(pass *dePass, input *IndexListExpr) *ast.IndexListExpr {
	return &ast.IndexListExpr{
		X:       DeserializeOption(pass, input.X, DeserializeExpr),
		Lbrack:  DeserializePos(pass, input.Lbrack),
		Indices: DeserializeList(pass, input.Indices, DeserializeExpr),
		Rbrack:  DeserializePos(pass, input.Rbrack),
	}
}

func DeserializeSliceExpr(pass *dePass, input *SliceExpr) *ast.SliceExpr {
	return &ast.SliceExpr{
		X:      DeserializeOption(pass, input.X, DeserializeExpr),
		Lbrack: DeserializePos(pass, input.Lbrack),
		Low:    DeserializeOption(pass, input.Low, DeserializeExpr),
		High:   DeserializeOption(pass, input.High, DeserializeExpr),
		Max:    DeserializeOption(pass, input.Max, DeserializeExpr),
		Slice3: input.Slice3,
		Rbrack: DeserializePos(pass, input.Rbrack),
	}
}

func DeserializeTypeAssertExpr(pass *dePass, input *TypeAssertExpr) *ast.TypeAssertExpr {
	return &ast.TypeAssertExpr{
		X:      DeserializeOption(pass, input.X, DeserializeExpr),
		Lparen: DeserializePos(pass, input.Lparen),
		Type:   DeserializeOption(pass, input.Type, DeserializeExpr),
		Rparen: DeserializePos(pass, input.Rparen),
	}
}

func DeserializeCallExpr(pass *dePass, input *CallExpr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:      DeserializeOption(pass, input.Fun, DeserializeExpr),
		Lparen:   DeserializePos(pass, input.Lparen),
		Args:     DeserializeList(pass, input.Args, DeserializeExpr),
		Ellipsis: DeserializePos(pass, input.Ellipsis),
		Rparen:   DeserializePos(pass, input.Rparen),
	}
}

func DeserializeStarExpr(pass *dePass, input *StarExpr) *ast.StarExpr {
	return &ast.StarExpr{
		Star: DeserializePos(pass, input.Star),
		X:    DeserializeOption(pass, input.X, DeserializeExpr),
	}
}

func DeserializeUnaryExpr(pass *dePass, input *UnaryExpr) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		OpPos: DeserializePos(pass, input.OpPos),
		Op:    tokens[input.Op],
		X:     DeserializeOption(pass, input.X, DeserializeExpr),
	}
}

func DeserializeBinaryExpr(pass *dePass, input *BinaryExpr) *ast.BinaryExpr {
	return &ast.BinaryExpr{
		X:     DeserializeOption(pass, input.X, DeserializeExpr),
		OpPos: DeserializePos(pass, input.OpPos),
		Op:    tokens[input.Op],
		Y:     DeserializeOption(pass, input.Y, DeserializeExpr),
	}
}

func DeserializeKeyValueExpr(pass *dePass, input *KeyValueExpr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   DeserializeOption(pass, input.Key, DeserializeExpr),
		Colon: DeserializePos(pass, input.Colon),
		Value: DeserializeOption(pass, input.Value, DeserializeExpr),
	}
}

// ----------------- Types ----------------- //

func DeserializeArrayType(pass *dePass, input *ArrayType) *ast.ArrayType {
	return &ast.ArrayType{
		Lbrack: DeserializePos(pass, input.Lbrack),
		Len:    DeserializeOption(pass, input.Len, DeserializeExpr),
		Elt:    DeserializeOption(pass, input.Elt, DeserializeExpr),
	}
}

func DeserializeStructType(pass *dePass, input *StructType) *ast.StructType {
	return &ast.StructType{
		Struct:     DeserializePos(pass, input.Struct),
		Fields:     DeserializeOption(pass, input.Fields, DeserializeFieldList),
		Incomplete: input.Incomplete,
	}
}

func DeserializeFuncType(pass *dePass, input *FuncType) *ast.FuncType {
	return &ast.FuncType{
		Func:       DeserializePos(pass, input.Func),
		TypeParams: DeserializeOption(pass, input.TypeParams, DeserializeFieldList),
		Params:     DeserializeOption(pass, input.Params, DeserializeFieldList),
		Results:    DeserializeOption(pass, input.Results, DeserializeFieldList),
	}
}

func DeserializeInterfaceType(pass *dePass, input *InterfaceType) *ast.InterfaceType {
	return &ast.InterfaceType{
		Interface:  DeserializePos(pass, input.Interface),
		Methods:    DeserializeOption(pass, input.Methods, DeserializeFieldList),
		Incomplete: input.Incomplete,
	}
}

func DeserializeMapType(pass *dePass, input *MapType) *ast.MapType {
	return &ast.MapType{
		Map:   DeserializePos(pass, input.Map),
		Key:   DeserializeOption(pass, input.Key, DeserializeExpr),
		Value: DeserializeOption(pass, input.Value, DeserializeExpr),
	}
}

func DeserializeChanType(pass *dePass, input *ChanType) *ast.ChanType {
	return &ast.ChanType{
		Begin: DeserializePos(pass, input.Begin),
		Arrow: DeserializePos(pass, input.Arrow),
		Dir:   ast.ChanDir(input.Dir),
		Value: DeserializeOption(pass, input.Value, DeserializeExpr),
	}
}

func DeserializeExpr(pass *dePass, expr Expr) ast.Expr {
	switch e := expr.(type) {
	case *Ident:
		return DeserializeIdent(pass, e)
	case *BasicLit:
		return DeserializeBasicLit(pass, e)
	case *CompositeLit:
		return DeserializeCompositeLit(pass, e)
	case *FuncLit:
		return DeserializeFuncLit(pass, e)
	case *Ellipsis:
		return DeserializeEllipsis(pass, e)
	case *BadExpr:
		return DeserializeBadExpr(pass, e)
	case *ParenExpr:
		return DeserializeParenExpr(pass, e)
	case *SelectorExpr:
		return DeserializeSelectorExpr(pass, e)
	case *IndexExpr:
		return DeserializeIndexExpr(pass, e)
	case *IndexListExpr:
		return DeserializeIndexListExpr(pass, e)
	case *SliceExpr:
		return DeserializeSliceExpr(pass, e)
	case *TypeAssertExpr:
		return DeserializeTypeAssertExpr(pass, e)
	case *CallExpr:
		return DeserializeCallExpr(pass, e)
	case *StarExpr:
		return DeserializeStarExpr(pass, e)
	case *UnaryExpr:
		return DeserializeUnaryExpr(pass, e)
	case *BinaryExpr:
		return DeserializeBinaryExpr(pass, e)
	case *KeyValueExpr:
		return DeserializeKeyValueExpr(pass, e)

	case *ArrayType:
		return DeserializeArrayType(pass, e)
	case *StructType:
		return DeserializeStructType(pass, e)
	case *FuncType:
		return DeserializeFuncType(pass, e)
	case *InterfaceType:
		return DeserializeInterfaceType(pass, e)
	case *MapType:
		return DeserializeMapType(pass, e)
	case *ChanType:
		return DeserializeChanType(pass, e)
	default:
		return nil
	}
}

// ----------------- Statements ----------------- //

func DeserializeBadStmt(pass *dePass, input *BadStmt) *ast.BadStmt {
	return &ast.BadStmt{
		From: DeserializePos(pass, input.From),
		To:   DeserializePos(pass, input.To),
	}
}

func DeserializeDeclStmt(pass *dePass, input *DeclStmt) *ast.DeclStmt {
	return &ast.DeclStmt{
		Decl: DeserializeOption(pass, input.Decl, DeserializeDecl),
	}
}

func DeserializeEmptyStmt(pass *dePass, input *EmptyStmt) *ast.EmptyStmt {
	return &ast.EmptyStmt{
		Semicolon: DeserializePos(pass, input.Semicolon),
		Implicit:  input.Implicit,
	}
}

func DeserializeLabeledStmt(pass *dePass, input *LabeledStmt) *ast.LabeledStmt {
	return &ast.LabeledStmt{
		Label: DeserializeOption(pass, input.Label, DeserializeIdent),
		Colon: DeserializePos(pass, input.Colon),
		Stmt:  DeserializeOption(pass, input.Stmt, DeserializeStmt),
	}
}

func DeserializeExprStmt(pass *dePass, input *ExprStmt) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: DeserializeOption(pass, input.X, DeserializeExpr),
	}
}

func DeserializeSendStmt(pass *dePass, input *SendStmt) *ast.SendStmt {
	return &ast.SendStmt{
		Chan:  DeserializeOption(pass, input.Chan, DeserializeExpr),
		Arrow: DeserializePos(pass, input.Arrow),
		Value: DeserializeOption(pass, input.Value, DeserializeExpr),
	}
}

func DeserializeIncDecStmt(pass *dePass, input *IncDecStmt) *ast.IncDecStmt {
	return &ast.IncDecStmt{
		X: DeserializeOption(pass, input.X, DeserializeExpr),
	}
}

func DeserializeAssignStmt(pass *dePass, input *AssignStmt) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs:    DeserializeList(pass, input.Lhs, DeserializeExpr),
		TokPos: DeserializePos(pass, input.TokPos),
		Tok:    tokens[input.Tok],
		Rhs:    DeserializeList(pass, input.Rhs, DeserializeExpr),
	}
}

func DeserializeGoStmt(pass *dePass, input *GoStmt) *ast.GoStmt {
	return &ast.GoStmt{
		Go:   DeserializePos(pass, input.Go),
		Call: DeserializeOption(pass, input.Call, DeserializeCallExpr),
	}
}

func DeserializeDeferStmt(pass *dePass, input *DeferStmt) *ast.DeferStmt {
	return &ast.DeferStmt{
		Defer: DeserializePos(pass, input.Defer),
		Call:  DeserializeOption(pass, input.Call, DeserializeCallExpr),
	}
}

func DeserializeBranchStmt(pass *dePass, input *BranchStmt) *ast.BranchStmt {
	return &ast.BranchStmt{
		TokPos: DeserializePos(pass, input.TokPos),
		Tok:    tokens[input.Tok],
		Label:  DeserializeOption(pass, input.Label, DeserializeIdent),
	}
}

func DeserializeReturnStmt(pass *dePass, input *ReturnStmt) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Return:  DeserializePos(pass, input.Return),
		Results: DeserializeList[Expr, ast.Expr](pass, input.Results, DeserializeExpr),
	}
}

func DeserializeBlockStmt(pass *dePass, input *BlockStmt) *ast.BlockStmt {
	return &ast.BlockStmt{
		Lbrace: DeserializePos(pass, input.Lbrace),
		List:   DeserializeList[Stmt, ast.Stmt](pass, input.List, DeserializeStmt),
		Rbrace: DeserializePos(pass, input.Rbrace),
	}
}

func DeserializeIfStmt(pass *dePass, input *IfStmt) *ast.IfStmt {
	return &ast.IfStmt{
		If:   DeserializePos(pass, input.If),
		Init: DeserializeOption(pass, input.Init, DeserializeStmt),
		Cond: DeserializeOption(pass, input.Cond, DeserializeExpr),
		Body: DeserializeOption(pass, input.Body, DeserializeBlockStmt),
		Else: DeserializeOption(pass, input.Else, DeserializeStmt),
	}
}

func DeserializeCaseClause(pass *dePass, input *CaseClause) *ast.CaseClause {
	return &ast.CaseClause{
		Case:  DeserializePos(pass, input.Case),
		List:  DeserializeList(pass, input.List, DeserializeExpr),
		Colon: DeserializePos(pass, input.Colon),
		Body:  DeserializeList(pass, input.Body, DeserializeStmt),
	}
}

func DeserializeSwitchStmt(pass *dePass, input *SwitchStmt) *ast.SwitchStmt {
	return &ast.SwitchStmt{
		Switch: DeserializePos(pass, input.Switch),
		Init:   DeserializeOption(pass, input.Init, DeserializeStmt),
		Tag:    DeserializeOption(pass, input.Tag, DeserializeExpr),
		Body:   DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeTypeSwitchStmt(pass *dePass, input *TypeSwitchStmt) *ast.TypeSwitchStmt {
	return &ast.TypeSwitchStmt{
		Switch: DeserializePos(pass, input.Switch),
		Init:   DeserializeOption(pass, input.Init, DeserializeStmt),
		Assign: DeserializeOption(pass, input.Assign, DeserializeStmt),
		Body:   DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeCommClause(pass *dePass, input *CommClause) *ast.CommClause {
	return &ast.CommClause{
		Case:  DeserializePos(pass, input.Case),
		Comm:  DeserializeOption(pass, input.Comm, DeserializeStmt),
		Colon: DeserializePos(pass, input.Colon),
		Body:  DeserializeList(pass, input.Body, DeserializeStmt),
	}
}

func DeserializeSelectStmt(pass *dePass, input *SelectStmt) *ast.SelectStmt {
	return &ast.SelectStmt{
		Select: DeserializePos(pass, input.Select),
		Body:   DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeForStmt(pass *dePass, input *ForStmt) *ast.ForStmt {
	return &ast.ForStmt{
		For:  DeserializePos(pass, input.For),
		Init: DeserializeOption(pass, input.Init, DeserializeStmt),
		Cond: DeserializeOption(pass, input.Cond, DeserializeExpr),
		Post: DeserializeOption(pass, input.Post, DeserializeStmt),
		Body: DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeRangeStmt(pass *dePass, input *RangeStmt) *ast.RangeStmt {
	return &ast.RangeStmt{
		For:    DeserializePos(pass, input.For),
		Key:    DeserializeOption(pass, input.Key, DeserializeExpr),
		Value:  DeserializeOption(pass, input.Value, DeserializeExpr),
		TokPos: DeserializePos(pass, input.TokPos),
		Tok:    tokens[input.Tok],
		Range:  DeserializePos(pass, input.Range),
		X:      DeserializeOption(pass, input.X, DeserializeExpr),
		Body:   DeserializeOption(pass, input.Body, DeserializeBlockStmt),
	}
}

func DeserializeStmt(pass *dePass, stmt Stmt) ast.Stmt {
	switch s := stmt.(type) {
	case *BadStmt:
		return DeserializeBadStmt(pass, s)
	case *DeclStmt:
		return DeserializeDeclStmt(pass, s)
	case *EmptyStmt:
		return DeserializeEmptyStmt(pass, s)
	case *LabeledStmt:
		return DeserializeLabeledStmt(pass, s)
	case *ExprStmt:
		return DeserializeExprStmt(pass, s)
	case *IncDecStmt:
		return DeserializeIncDecStmt(pass, s)
	case *AssignStmt:
		return DeserializeAssignStmt(pass, s)
	case *GoStmt:
		return DeserializeGoStmt(pass, s)
	case *DeferStmt:
		return DeserializeDeferStmt(pass, s)
	case *ReturnStmt:
		return DeserializeReturnStmt(pass, s)
	case *BranchStmt:
		return DeserializeBranchStmt(pass, s)
	case *SendStmt:
		return DeserializeSendStmt(pass, s)
	case *IfStmt:
		return DeserializeIfStmt(pass, s)
	case *CaseClause:
		return DeserializeCaseClause(pass, s)
	case *SwitchStmt:
		return DeserializeSwitchStmt(pass, s)
	case *TypeSwitchStmt:
		return DeserializeTypeSwitchStmt(pass, s)
	case *BlockStmt:
		return DeserializeBlockStmt(pass, s)
	case *CommClause:
		return DeserializeCommClause(pass, s)
	case *SelectStmt:
		return DeserializeSelectStmt(pass, s)
	case *ForStmt:
		return DeserializeForStmt(pass, s)
	case *RangeStmt:
		return DeserializeRangeStmt(pass, s)
	default:
		return nil
	}
}

// ----------------- Specifications ----------------- //

func DeserializeImportSpec(pass *dePass, input *ImportSpec) *ast.ImportSpec {
	deserializeImportSpec := func(pass *dePass, input *ImportSpec) *ast.ImportSpec {
		return &ast.ImportSpec{
			Doc:     DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
			Name:    DeserializeOption(pass, input.Name, DeserializeIdent),
			Path:    DeserializeOption(pass, input.Path, DeserializeBasicLit),
			Comment: DeserializeOption(pass, input.Comment, DeserializeCommentGroup),
			EndPos:  DeserializePos(pass, input.EndPos),
		}
	}

	if pass.conf[CacheRef] != nil {
		return DeRefLookup(pass, input, deserializeImportSpec)
	}

	return deserializeImportSpec(pass, input)
}

func DeserializeValueSpec(pass *dePass, input *ValueSpec) *ast.ValueSpec {
	deserializeValueSpec := func(pass *dePass, input *ValueSpec) *ast.ValueSpec {
		return &ast.ValueSpec{
			Doc:     DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
			Names:   DeserializeList(pass, input.Names, DeserializeIdent),
			Type:    DeserializeOption(pass, input.Type, DeserializeExpr),
			Values:  DeserializeList(pass, input.Values, DeserializeExpr),
			Comment: DeserializeOption(pass, input.Comment, DeserializeCommentGroup),
		}
	}

	if pass.conf[CacheRef] != nil {
		return DeRefLookup(pass, input, deserializeValueSpec)
	}

	return deserializeValueSpec(pass, input)
}

func DeserializeTypeSpec(pass *dePass, input *TypeSpec) *ast.TypeSpec {
	deserializeTypeSpec := func(pass *dePass, input *TypeSpec) *ast.TypeSpec {
		return &ast.TypeSpec{
			Doc:        DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
			Name:       DeserializeOption(pass, input.Name, DeserializeIdent),
			TypeParams: DeserializeOption(pass, input.TypeParams, DeserializeFieldList),
			Assign:     DeserializePos(pass, input.Assign),
			Type:       DeserializeOption(pass, input.Type, DeserializeExpr),
			Comment:    DeserializeOption(pass, input.Comment, DeserializeCommentGroup),
		}
	}

	if pass.conf[CacheRef] != nil {
		return DeRefLookup(pass, input, deserializeTypeSpec)
	}

	return deserializeTypeSpec(pass, input)
}

func DeserializeSpec(pass *dePass, spec Spec) ast.Spec {
	switch s := spec.(type) {
	case *ImportSpec:
		return DeserializeImportSpec(pass, s)
	case *ValueSpec:
		return DeserializeValueSpec(pass, s)
	case *TypeSpec:
		return DeserializeTypeSpec(pass, s)
	default:
		return nil
	}
}

// ----------------- Declarations ----------------- //

func DeserializeBadDecl(pass *dePass, input *BadDecl) *ast.BadDecl {
	return &ast.BadDecl{
		From: DeserializePos(pass, input.From),
		To:   DeserializePos(pass, input.To),
	}
}

func DeserializeGenDecl(pass *dePass, input *GenDecl) *ast.GenDecl {
	deserializeGenDecl := func(pass *dePass, input *GenDecl) *ast.GenDecl {
		return &ast.GenDecl{
			Doc:    DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
			TokPos: DeserializePos(pass, input.TokenPos),
			Tok:    tokens[input.Tok],
			Lparen: DeserializePos(pass, input.Lparen),
			Specs:  DeserializeList(pass, input.Specs, DeserializeSpec),
			Rparen: DeserializePos(pass, input.Rparen),
		}
	}

	if pass.conf[CacheRef] != nil {
		return DeRefLookup(pass, input, deserializeGenDecl)
	}

	return deserializeGenDecl(pass, input)
}

func DeserializeFuncDecl(pass *dePass, input *FuncDecl) *ast.FuncDecl {
	deserializeFuncDecl := func(pass *dePass, input *FuncDecl) *ast.FuncDecl {
		return &ast.FuncDecl{
			Doc:  DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
			Recv: DeserializeOption(pass, input.Recv, DeserializeFieldList),
			Name: DeserializeOption(pass, input.Name, DeserializeIdent),
			Type: DeserializeOption(pass, input.Type, DeserializeFuncType),
			Body: DeserializeOption(pass, input.Body, DeserializeBlockStmt),
		}
	}

	if pass.conf[CacheRef] != nil {
		return DeRefLookup(pass, input, deserializeFuncDecl)
	}

	return deserializeFuncDecl(pass, input)
}

func DeserializeDecl(pass *dePass, decl Decl) ast.Decl {
	switch d := decl.(type) {
	case *BadDecl:
		return DeserializeBadDecl(pass, d)
	case *GenDecl:
		return DeserializeGenDecl(pass, d)
	case *FuncDecl:
		return DeserializeFuncDecl(pass, d)
	default:
		return nil
	}
}

// ----------------- Files and Packages ----------------- //

func DeserializeFile(pass *dePass, input *File) (*ast.File, error) {
	if err := processTokenFile(pass, input); err != nil {
		return nil, err
	}

	return &ast.File{
		Doc:        DeserializeOption(pass, input.Doc, DeserializeCommentGroup),
		Package:    DeserializePos(pass, input.Package),
		Name:       DeserializeOption(pass, input.Name, DeserializeIdent),
		Decls:      DeserializeList(pass, input.Decls, DeserializeDecl),
		FileStart:  DeserializePos(pass, input.FileStart),
		FileEnd:    DeserializePos(pass, input.FileEnd),
		Imports:    DeserializeList(pass, input.Imports, DeserializeImportSpec),
		Unresolved: DeserializeList(pass, input.Unresolved, DeserializeIdent),
		Comments:   DeserializeList(pass, input.Comments, DeserializeCommentGroup),
		GoVersion:  input.GoVersion,
	}, nil
}

func processTokenFile(pass *dePass, input *File) error {
	if input == nil || input.Name == nil {
		return fmt.Errorf("input or input.Name is nil")
	}

	pos, ok := input.Name.NamePos.(*Position)
	if !ok {
		return fmt.Errorf("failed to get start pos for file `%s`", input.Name.Name)
	}

	fileSize := input.Size
	if fileSize <= 0 {
		fileSize = _GOARCH()
	}

	tokFile := pass.fset.AddFile(pos.Filename, pos.Offset(), fileSize)
	if tokFile == nil {
		return fmt.Errorf("failed to add file to file set")
	}

	tokFile.SetLinesForContent([]byte{})
	return nil
}
