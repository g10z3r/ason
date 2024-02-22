package ason

import (
	"go/ast"
	"go/token"
	"os"
	"unsafe"
)

// ignoredNodes

type serPass struct {
	fset     *token.FileSet
	refCache map[ast.Node]*weakRef // cache of weak refs to already serialized nodes
	refCount uint                  // serialized nodes counter
	conf     map[Mode]interface{}  // set of serialization parameters
}

func NewSerPass(fset *token.FileSet, options ...Mode) *serPass {
	pass := &serPass{
		fset: fset,
		conf: make(map[Mode]interface{}),
	}

	for i := 0; i < len(options); i++ {
		opt := options[i]
		pass.conf[opt] = struct{}{}

		if opt == CacheRef {
			pass.refCache = make(map[ast.Node]*weakRef)
		}
	}

	return pass
}

type SerFn[I ast.Node, O Ason] func(*serPass, I) O

func SerRefLookup[I ast.Node, O Ason](pass *serPass, input I, serFn SerFn[I, O]) O {
	if weakRef, exists := pass.refCache[input]; exists {
		return weakRef.Load().(O)
	}

	node := serFn(pass, input)
	pass.refCache[input] = NewWeakRef(&node)
	pass.refCount++

	return node
}

func SerializeOption[I ast.Node, R Ason](pass *serPass, input I, serFn SerFn[I, R]) (empty R) {
	if *(*interface{})(unsafe.Pointer(&input)) != nil {
		return serFn(pass, input)
	}

	return empty
}

func SerializeList[I ast.Node, R Ason](pass *serPass, inputList []I, serFn SerFn[I, R]) (result []R) {
	if len(inputList) < 1 {
		return nil
	}

	for i := 0; i < len(inputList); i++ {
		val := SerializeOption(pass, inputList[i], serFn)

		if *(*interface{})(unsafe.Pointer(&val)) != nil {
			result = append(result, val)
		}
	}

	return result
}

func SerializeMap[K comparable, V ast.Node, R Ason](pass *serPass, inputMap map[K]V, serFn SerFn[V, R]) map[K]R {
	result := make(map[K]R, len(inputMap))
	for k, v := range inputMap {
		result[k] = serFn(pass, v)
	}

	return result
}

// ----------------- Scope ----------------- //

func SerializeScope(pass *serPass, input *ast.Scope) *Scope {
	if pass.conf[ResolveScope] == nil {
		return nil
	}

	var objects map[string]*Object
	if input.Objects != nil {
		objects = make(map[string]*Object, len(input.Objects))

		for k, v := range input.Objects {
			if v != nil {
				objects[k] = SerializeObject(pass, v)
			}
		}
	}

	return &Scope{
		Outer:   SerializeScope(pass, input.Outer),
		Objects: objects,
	}
}

func SerializeObject(pass *serPass, input *ast.Object) *Object {
	if pass.conf[ResolveObject] == nil {
		return nil
	}

	return &Object{
		Kind: input.Kind.String(),
		Name: input.Name,
	}
}

func SerializePos(pass *serPass, pos token.Pos) Pos {
	if pos != token.NoPos {
		return NewPosition(pass.fset.PositionFor(pos, false))
	}

	return new(NoPos)
}

func SerializeLoc(pass *serPass, input ast.Node) *Loc {
	if pass.conf[ResolveLoc] == nil {
		return nil
	}

	return &Loc{
		Start: SerializePos(pass, input.Pos()),
		End:   SerializePos(pass, input.End()),
	}
}

// ----------------- Comments ----------------- //

func SerializeComment(pass *serPass, input *ast.Comment) *Comment {
	if pass.conf[SkipComments] != nil {
		return nil
	}

	return &Comment{
		Slash: SerializePos(pass, input.Slash),
		Text:  input.Text,
		Node: Node{
			Type: NodeTypeComment,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeCommentGroup(pass *serPass, input *ast.CommentGroup) *CommentGroup {
	if pass.conf[SkipComments] != nil {
		return nil
	}

	return &CommentGroup{
		List: SerializeList[*ast.Comment, *Comment](pass, input.List, SerializeComment),
		Node: Node{
			Type: NodeTypeCommentGroup,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

// ----------------- Expressions ----------------- //

func SerializeIdent(pass *serPass, input *ast.Ident) *Ident {
	var obj *Object
	if input.Obj != nil {
		obj = SerializeObject(pass, input.Obj)
	}

	return &Ident{
		NamePos: SerializePos(pass, input.NamePos),
		Name:    input.String(),
		Obj:     obj,
		Node: Node{
			Type: NodeTypeIdent,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeBasicLit(pass *serPass, input *ast.BasicLit) *BasicLit {
	return &BasicLit{
		ValuePos: SerializePos(pass, input.ValuePos),
		Kind:     input.Kind.String(),
		Value:    input.Value,
		Node: Node{
			Type: NodeTypeBasicLit,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeFuncLit(pass *serPass, input *ast.FuncLit) *FuncLit {
	return &FuncLit{
		Type: SerializeOption(pass, input.Type, SerializeFuncType),
		Body: SerializeOption(pass, input.Body, SerializeBlockStmt),
		Node: Node{
			Type: NodeTypeFuncLit,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeCompositeLit(pass *serPass, input *ast.CompositeLit) *CompositeLit {
	return &CompositeLit{
		Type:   SerializeOption(pass, input.Type, SerializeExpr),
		Lbrace: SerializePos(pass, input.Lbrace),
		Elts:   SerializeList(pass, input.Elts, SerializeExpr),
		Node: Node{
			Type: NodeTypeCompositeLit,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeField(pass *serPass, input *ast.Field) *Field {
	return &Field{
		Doc:     SerializeOption(pass, input.Doc, SerializeCommentGroup),
		Names:   SerializeList(pass, input.Names, SerializeIdent),
		Type:    SerializeOption(pass, input.Type, SerializeExpr),
		Tag:     SerializeOption(pass, input.Tag, SerializeBasicLit),
		Comment: SerializeOption(pass, input.Comment, SerializeCommentGroup),
		Node: Node{
			Type: NodeTypeField,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeFieldList(pass *serPass, input *ast.FieldList) *FieldList {
	return &FieldList{
		Opening: SerializePos(pass, input.Opening),
		List:    SerializeList(pass, input.List, SerializeField),
		Closing: SerializePos(pass, input.Closing),
		Node: Node{
			Type: NodeTypeFieldList,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeEllipsis(pass *serPass, input *ast.Ellipsis) *Ellipsis {
	return &Ellipsis{
		Ellipsis: SerializePos(pass, input.Ellipsis),
		Elt:      SerializeOption(pass, input.Elt, SerializeExpr),
		Node: Node{
			Type: NodeTypeEllipsis,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeBadExpr(pass *serPass, input *ast.BadExpr) *BadExpr {
	return &BadExpr{
		From: SerializePos(pass, input.From),
		To:   SerializePos(pass, input.To),
		Node: Node{
			Type: NodeTypeBadExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeParenExpr(pass *serPass, input *ast.ParenExpr) *ParenExpr {
	return &ParenExpr{
		Lparen: SerializePos(pass, input.Lparen),
		X:      SerializeOption(pass, input.X, SerializeExpr),
		Rparen: SerializePos(pass, input.Rparen),
		Node: Node{
			Type: NodeTypeParenExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeSelectorExpr(pass *serPass, input *ast.SelectorExpr) *SelectorExpr {
	return &SelectorExpr{
		X:   SerializeOption(pass, input.X, SerializeExpr),
		Sel: SerializeOption(pass, input.Sel, SerializeIdent),
		Node: Node{
			Type: NodeTypeSelectorExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeIndexExpr(pass *serPass, input *ast.IndexExpr) *IndexExpr {
	return &IndexExpr{
		X:      SerializeOption(pass, input.X, SerializeExpr),
		Lbrack: SerializePos(pass, input.Lbrack),
		Index:  SerializeOption(pass, input.Index, SerializeExpr),
		Rbrack: SerializePos(pass, input.Rbrack),
		Node: Node{
			Type: NodeTypeIndexExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeIndexListExpr(pass *serPass, input *ast.IndexListExpr) *IndexListExpr {
	return &IndexListExpr{
		X:       SerializeOption(pass, input.X, SerializeExpr),
		Lbrack:  SerializePos(pass, input.Lbrack),
		Indices: SerializeList(pass, input.Indices, SerializeExpr),
		Rbrack:  SerializePos(pass, input.Rbrack),
		Node: Node{
			Type: NodeTypeIndexListExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeSliceExpr(pass *serPass, input *ast.SliceExpr) *SliceExpr {
	return &SliceExpr{
		X:      SerializeOption(pass, input.X, SerializeExpr),
		Lbrack: SerializePos(pass, input.Lbrack),
		Low:    SerializeOption(pass, input.Low, SerializeExpr),
		High:   SerializeOption(pass, input.High, SerializeExpr),
		Max:    SerializeOption(pass, input.Max, SerializeExpr),
		Slice3: input.Slice3,
		Rbrack: SerializePos(pass, input.Rbrack),
		Node: Node{
			Type: NodeTypeSliceExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeTypeAssertExpr(pass *serPass, input *ast.TypeAssertExpr) *TypeAssertExpr {
	return &TypeAssertExpr{
		X:      SerializeOption(pass, input.X, SerializeExpr),
		Lparen: SerializePos(pass, input.Lparen),
		Type:   SerializeOption(pass, input.Type, SerializeExpr),
		Rparen: SerializePos(pass, input.Rparen),
		Node: Node{
			Type: NodeTypeTypeAssertExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeCallExpr(pass *serPass, input *ast.CallExpr) *CallExpr {
	return &CallExpr{
		Fun:      SerializeOption(pass, input.Fun, SerializeExpr),
		Lparen:   SerializePos(pass, input.Lparen),
		Args:     SerializeList(pass, input.Args, SerializeExpr),
		Ellipsis: SerializePos(pass, input.Ellipsis),
		Rparen:   SerializePos(pass, input.Rparen),
		Node: Node{
			Type: NodeTypeCallExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeStarExpr(pass *serPass, input *ast.StarExpr) *StarExpr {
	return &StarExpr{
		Star: SerializePos(pass, input.Star),
		X:    SerializeOption(pass, input.X, SerializeExpr),
		Node: Node{
			Type: NodeTypeStarExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeUnaryExpr(pass *serPass, input *ast.UnaryExpr) *UnaryExpr {
	return &UnaryExpr{
		OpPos: SerializePos(pass, input.OpPos),
		Op:    input.Op.String(),
		X:     SerializeOption(pass, input.X, SerializeExpr),
		Node: Node{
			Type: NodeTypeUnaryExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeBinaryExpr(pass *serPass, input *ast.BinaryExpr) *BinaryExpr {
	return &BinaryExpr{
		X:     SerializeOption(pass, input.X, SerializeExpr),
		OpPos: SerializePos(pass, input.OpPos),
		Op:    input.Op.String(),
		Y:     SerializeOption(pass, input.Y, SerializeExpr),
		Node: Node{
			Type: NodeTypeBinaryExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeKeyValueExpr(pass *serPass, input *ast.KeyValueExpr) *KeyValueExpr {
	return &KeyValueExpr{
		Key:   SerializeOption(pass, input.Key, SerializeExpr),
		Colon: SerializePos(pass, input.Colon),
		Value: SerializeOption(pass, input.Value, SerializeExpr),
		Node: Node{
			Type: NodeTypeKeyValueExpr,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

// ----------------- Types ----------------- //

func SerializeArrayType(pass *serPass, input *ast.ArrayType) *ArrayType {
	return &ArrayType{
		Lbrack: SerializePos(pass, input.Lbrack),
		Len:    SerializeOption(pass, input.Len, SerializeExpr),
		Elt:    SerializeOption(pass, input.Elt, SerializeExpr),
		Node: Node{
			Type: NodeTypeArrayType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeStructType(pass *serPass, input *ast.StructType) *StructType {
	return &StructType{
		Struct:     SerializePos(pass, input.Struct),
		Fields:     SerializeOption(pass, input.Fields, SerializeFieldList),
		Incomplete: input.Incomplete,
		Node: Node{
			Type: NodeTypeStructType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeFuncType(pass *serPass, input *ast.FuncType) *FuncType {
	return &FuncType{
		Func:       SerializePos(pass, input.Func),
		TypeParams: SerializeOption(pass, input.TypeParams, SerializeFieldList),
		Params:     SerializeOption(pass, input.Params, SerializeFieldList),
		Results:    SerializeOption(pass, input.Results, SerializeFieldList),
		Node: Node{
			Type: NodeTypeFuncType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeInterfaceType(pass *serPass, input *ast.InterfaceType) *InterfaceType {
	return &InterfaceType{
		Interface:  SerializePos(pass, input.Interface),
		Methods:    SerializeOption(pass, input.Methods, SerializeFieldList),
		Incomplete: input.Incomplete,
		Node: Node{
			Type: NodeTypeInterfaceType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeMapType(pass *serPass, input *ast.MapType) *MapType {
	return &MapType{
		Map:   SerializePos(pass, input.Map),
		Key:   SerializeOption(pass, input.Key, SerializeExpr),
		Value: SerializeOption(pass, input.Value, SerializeExpr),
		Node: Node{
			Type: NodeTypeMapType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeChanType(pass *serPass, input *ast.ChanType) *ChanType {
	return &ChanType{
		Begin: SerializePos(pass, input.Begin),
		Arrow: SerializePos(pass, input.Arrow),
		Dir:   int(input.Dir),
		Value: SerializeOption(pass, input.Value, SerializeExpr),
		Node: Node{
			Type: NodeTypeChanType,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeExpr(pass *serPass, expr ast.Expr) Expr {
	switch e := expr.(type) {
	case *ast.Ident:
		return SerializeIdent(pass, e)
	case *ast.Ellipsis:
		return SerializeEllipsis(pass, e)
	case *ast.BasicLit:
		return SerializeBasicLit(pass, e)
	case *ast.FuncLit:
		return SerializeFuncLit(pass, e)
	case *ast.CompositeLit:
		return SerializeCompositeLit(pass, e)
	case *ast.BadExpr:
		return SerializeBadExpr(pass, e)
	case *ast.SelectorExpr:
		return SerializeSelectorExpr(pass, e)
	case *ast.ParenExpr:
		return SerializeParenExpr(pass, e)
	case *ast.IndexExpr:
		return SerializeIndexExpr(pass, e)
	case *ast.IndexListExpr:
		return SerializeIndexListExpr(pass, e)
	case *ast.UnaryExpr:
		return SerializeUnaryExpr(pass, e)
	case *ast.SliceExpr:
		return SerializeSliceExpr(pass, e)
	case *ast.TypeAssertExpr:
		return SerializeTypeAssertExpr(pass, e)
	case *ast.CallExpr:
		return SerializeCallExpr(pass, e)
	case *ast.StarExpr:
		return SerializeStarExpr(pass, e)
	case *ast.BinaryExpr:
		return SerializeBinaryExpr(pass, e)
	case *ast.KeyValueExpr:
		return SerializeKeyValueExpr(pass, e)

	case *ast.ArrayType:
		return SerializeArrayType(pass, e)
	case *ast.StructType:
		return SerializeStructType(pass, e)
	case *ast.FuncType:
		return SerializeFuncType(pass, e)
	case *ast.InterfaceType:
		return SerializeInterfaceType(pass, e)
	case *ast.MapType:
		return SerializeMapType(pass, e)
	case *ast.ChanType:
		return SerializeChanType(pass, e)
	default:
		return nil
	}
}

// ----------------- Statements ----------------- //

func SerializeBadStmt(pass *serPass, input *ast.BadStmt) *BadStmt {
	return &BadStmt{
		From: SerializePos(pass, input.From),
		To:   SerializePos(pass, input.To),
		Node: Node{
			Type: NodeTypeBadStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeDeclStmt(pass *serPass, input *ast.DeclStmt) *DeclStmt {
	return &DeclStmt{
		Decl: SerializeOption(pass, input.Decl, SerializeDecl),
		Node: Node{
			Type: NodeTypeDeclStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeEmptyStmt(pass *serPass, input *ast.EmptyStmt) *EmptyStmt {
	return &EmptyStmt{
		Semicolon: SerializePos(pass, input.Semicolon),
		Implicit:  input.Implicit,
		Node: Node{
			Type: NodeTypeEmptyStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeLabeledStmt(pass *serPass, input *ast.LabeledStmt) *LabeledStmt {
	return &LabeledStmt{
		Label: SerializeOption(pass, input.Label, SerializeIdent),
		Colon: SerializePos(pass, input.Colon),
		Stmt:  SerializeOption(pass, input.Stmt, SerializeStmt),
		Node: Node{
			Type: NodeTypeLabeledStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeExprStmt(pass *serPass, input *ast.ExprStmt) *ExprStmt {
	return &ExprStmt{
		X: SerializeOption(pass, input.X, SerializeExpr),
		Node: Node{
			Type: NodeTypeExprStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeSendStmt(pass *serPass, input *ast.SendStmt) *SendStmt {
	return &SendStmt{
		Chan:  SerializeOption(pass, input.Chan, SerializeExpr),
		Arrow: SerializePos(pass, input.Arrow),
		Value: SerializeOption(pass, input.Value, SerializeExpr),
		Node: Node{
			Type: NodeTypeSendStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeIncDecStmt(pass *serPass, input *ast.IncDecStmt) *IncDecStmt {
	return &IncDecStmt{
		X:      SerializeOption(pass, input.X, SerializeExpr),
		TokPos: SerializePos(pass, input.TokPos),
		Tok:    input.Tok.String(),
		Node: Node{
			Type: NodeTypeIncDecStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeAssignStmt(pass *serPass, input *ast.AssignStmt) *AssignStmt {
	return &AssignStmt{
		Lhs:    SerializeList(pass, input.Lhs, SerializeExpr),
		TokPos: SerializePos(pass, input.TokPos),
		Tok:    input.Tok.String(),
		Rhs:    SerializeList(pass, input.Rhs, SerializeExpr),
		Node: Node{
			Type: NodeTypeAssignStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeGoStmt(pass *serPass, input *ast.GoStmt) *GoStmt {
	return &GoStmt{
		Go:   SerializePos(pass, input.Go),
		Call: SerializeOption(pass, input.Call, SerializeCallExpr),
		Node: Node{
			Type: NodeTypeGoStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeDeferStmt(pass *serPass, input *ast.DeferStmt) *DeferStmt {
	return &DeferStmt{
		Defer: SerializePos(pass, input.Defer),
		Call:  SerializeOption(pass, input.Call, SerializeCallExpr),
		Node: Node{
			Type: NodeTypeDeferStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeReturnStmt(pass *serPass, input *ast.ReturnStmt) *ReturnStmt {
	return &ReturnStmt{
		Return:  SerializePos(pass, input.Return),
		Results: SerializeList(pass, input.Results, SerializeExpr),
		Node: Node{
			Type: NodeTypeReturnStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeBranchStmt(pass *serPass, input *ast.BranchStmt) *BranchStmt {
	return &BranchStmt{
		TokPos: SerializePos(pass, input.TokPos),
		Tok:    input.Tok.String(),
		Label:  SerializeOption(pass, input.Label, SerializeIdent),
		Node: Node{
			Type: NodeTypeBranchStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeBlockStmt(pass *serPass, input *ast.BlockStmt) *BlockStmt {
	return &BlockStmt{
		Lbrace: SerializePos(pass, input.Lbrace),
		List:   SerializeList(pass, input.List, SerializeStmt),
		Rbrace: SerializePos(pass, input.Rbrace),
		Node: Node{
			Type: NodeTypeBlockStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeIfStmt(pass *serPass, input *ast.IfStmt) *IfStmt {
	return &IfStmt{
		If:   SerializePos(pass, input.If),
		Init: SerializeOption(pass, input.Init, SerializeStmt),
		Cond: SerializeOption(pass, input.Cond, SerializeExpr),
		Body: SerializeOption(pass, input.Body, SerializeBlockStmt),
		Else: SerializeOption(pass, input.Else, SerializeStmt),
		Node: Node{
			Type: NodeTypeIfStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeCaseClause(pass *serPass, input *ast.CaseClause) *CaseClause {
	return &CaseClause{
		Case:  SerializePos(pass, input.Case),
		List:  SerializeList(pass, input.List, SerializeExpr),
		Colon: SerializePos(pass, input.Colon),
		Body:  SerializeList(pass, input.Body, SerializeStmt),
		Node: Node{
			Type: NodeTypeCaseClause,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeSwitchStmt(pass *serPass, input *ast.SwitchStmt) *SwitchStmt {
	return &SwitchStmt{
		Switch: SerializePos(pass, input.Switch),
		Init:   SerializeOption(pass, input.Init, SerializeStmt),
		Tag:    SerializeOption(pass, input.Tag, SerializeExpr),
		Body:   SerializeOption(pass, input.Body, SerializeBlockStmt),
		Node: Node{
			Type: NodeTypeSwitchStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeTypeSwitchStmt(pass *serPass, input *ast.TypeSwitchStmt) *TypeSwitchStmt {
	return &TypeSwitchStmt{
		Switch: SerializePos(pass, input.Switch),
		Init:   SerializeStmt(pass, input.Init),
		Assign: SerializeStmt(pass, input.Assign),
		Body:   SerializeBlockStmt(pass, input.Body),
		Node: Node{
			Type: NodeTypeExprStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeCommClause(pass *serPass, input *ast.CommClause) *CommClause {
	return &CommClause{
		Case:  SerializePos(pass, input.Case),
		Comm:  SerializeOption(pass, input.Comm, SerializeStmt),
		Colon: SerializePos(pass, input.Colon),
		Body:  SerializeList(pass, input.Body, SerializeStmt),
		Node: Node{
			Type: NodeTypeCommClause,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeSelectStmt(pass *serPass, input *ast.SelectStmt) *SelectStmt {
	return &SelectStmt{
		Select: SerializePos(pass, input.Select),
		Body:   SerializeOption(pass, input.Body, SerializeBlockStmt),
		Node: Node{
			Type: NodeTypeSelectStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeForStmt(pass *serPass, input *ast.ForStmt) *ForStmt {
	return &ForStmt{
		For:  SerializePos(pass, input.For),
		Init: SerializeOption(pass, input.Init, SerializeStmt),
		Cond: SerializeOption(pass, input.Cond, SerializeExpr),
		Post: SerializeOption(pass, input.Post, SerializeStmt),
		Body: SerializeOption(pass, input.Body, SerializeBlockStmt),
		Node: Node{
			Type: NodeTypeForStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeRangeStmt(pass *serPass, input *ast.RangeStmt) *RangeStmt {
	return &RangeStmt{
		For:    SerializePos(pass, input.For),
		Key:    SerializeOption(pass, input.Key, SerializeExpr),
		Value:  SerializeOption(pass, input.Value, SerializeExpr),
		TokPos: SerializePos(pass, input.TokPos),
		Tok:    input.Tok.String(),
		Range:  SerializePos(pass, input.Range),
		X:      SerializeOption(pass, input.X, SerializeExpr),
		Body:   SerializeBlockStmt(pass, input.Body),
		Node: Node{
			Type: NodeTypeExprStmt,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeStmt(pass *serPass, stmt ast.Stmt) Stmt {
	switch s := stmt.(type) {
	case *ast.BadStmt:
		return SerializeBadStmt(pass, s)
	case *ast.DeclStmt:
		return SerializeDeclStmt(pass, s)
	case *ast.EmptyStmt:
		return SerializeEmptyStmt(pass, s)
	case *ast.LabeledStmt:
		return SerializeLabeledStmt(pass, s)
	case *ast.ExprStmt:
		return SerializeExprStmt(pass, s)
	case *ast.SendStmt:
		return SerializeSendStmt(pass, s)
	case *ast.IncDecStmt:
		return SerializeIncDecStmt(pass, s)
	case *ast.AssignStmt:
		return SerializeAssignStmt(pass, s)
	case *ast.GoStmt:
		return SerializeGoStmt(pass, s)
	case *ast.DeferStmt:
		return SerializeDeferStmt(pass, s)
	case *ast.ReturnStmt:
		return SerializeReturnStmt(pass, s)
	case *ast.BranchStmt:
		return SerializeBranchStmt(pass, s)
	case *ast.BlockStmt:
		return SerializeBlockStmt(pass, s)
	case *ast.IfStmt:
		return SerializeIfStmt(pass, s)
	case *ast.ForStmt:
		return SerializeForStmt(pass, s)
	case *ast.RangeStmt:
		return SerializeRangeStmt(pass, s)
	case *ast.SelectStmt:
		return SerializeSelectStmt(pass, s)
	case *ast.CaseClause:
		return SerializeCaseClause(pass, s)
	case *ast.CommClause:
		return SerializeCommClause(pass, s)
	case *ast.TypeSwitchStmt:
		return SerializeTypeSwitchStmt(pass, s)
	case *ast.SwitchStmt:
		return SerializeSwitchStmt(pass, s)
	default:
		return nil
	}
}

// ----------------- Specifications ----------------- //

func SerializeImportSpec(pass *serPass, input *ast.ImportSpec) *ImportSpec {
	serializeImportSpec := func(pass *serPass, input *ast.ImportSpec) *ImportSpec {
		return &ImportSpec{
			Doc:     SerializeOption(pass, input.Doc, SerializeCommentGroup),
			Name:    SerializeOption(pass, input.Name, SerializeIdent),
			Path:    SerializeOption(pass, input.Path, SerializeBasicLit),
			Comment: SerializeOption(pass, input.Comment, SerializeCommentGroup),
			EndPos:  SerializePos(pass, input.EndPos),
			Node: Node{
				Type: NodeTypeImportSpec,
				Ref:  pass.refCount,
				Loc:  SerializeLoc(pass, input),
			},
		}
	}

	if pass.conf[CacheRef] != nil {
		return SerRefLookup(pass, input, serializeImportSpec)
	}

	return serializeImportSpec(pass, input)
}

func SerializeValueSpec(pass *serPass, input *ast.ValueSpec) *ValueSpec {
	serializeValueSpec := func(pass *serPass, input *ast.ValueSpec) *ValueSpec {
		return &ValueSpec{
			Doc:     SerializeOption(pass, input.Doc, SerializeCommentGroup),
			Names:   SerializeList(pass, input.Names, SerializeIdent),
			Type:    SerializeOption(pass, input.Type, SerializeExpr),
			Values:  SerializeList(pass, input.Values, SerializeExpr),
			Comment: SerializeOption(pass, input.Comment, SerializeCommentGroup),
			Node: Node{
				Type: NodeTypeValueSpec,
				Ref:  pass.refCount,
				Loc:  SerializeLoc(pass, input),
			},
		}
	}

	if pass.conf[CacheRef] != nil {
		return SerRefLookup(pass, input, serializeValueSpec)
	}

	return serializeValueSpec(pass, input)
}

func SerializeTypeSpec(pass *serPass, input *ast.TypeSpec) *TypeSpec {
	serializeTypeSpec := func(pass *serPass, input *ast.TypeSpec) *TypeSpec {
		return &TypeSpec{
			Doc:        SerializeOption(pass, input.Doc, SerializeCommentGroup),
			Name:       SerializeOption(pass, input.Name, SerializeIdent),
			TypeParams: SerializeOption(pass, input.TypeParams, SerializeFieldList),
			Assign:     SerializePos(pass, input.Assign),
			Type:       SerializeOption(pass, input.Type, SerializeExpr),
			Comment:    SerializeOption(pass, input.Comment, SerializeCommentGroup),
			Node: Node{
				Type: NodeTypeTypeSpec,
				Ref:  pass.refCount,
				Loc:  SerializeLoc(pass, input),
			},
		}
	}

	if pass.conf[CacheRef] != nil {
		return SerRefLookup(pass, input, serializeTypeSpec)
	}

	return serializeTypeSpec(pass, input)
}

func SerializeSpec(pass *serPass, spec ast.Spec) Spec {
	switch s := spec.(type) {
	case *ast.ImportSpec:
		return SerializeImportSpec(pass, s)
	case *ast.ValueSpec:
		return SerializeValueSpec(pass, s)
	case *ast.TypeSpec:
		return SerializeTypeSpec(pass, s)
	default:
		return nil
	}
}

// ----------------- Declarations ----------------- //

func SerializeBadDecl(pass *serPass, input *ast.BadDecl) *BadDecl {
	return &BadDecl{
		From: SerializePos(pass, input.From),
		To:   SerializePos(pass, input.To),
		Node: Node{
			Type: NodeTypeBadDecl,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func SerializeGenDecl(pass *serPass, input *ast.GenDecl) *GenDecl {
	serializeGenDecl := func(pass *serPass, input *ast.GenDecl) *GenDecl {
		return &GenDecl{
			Doc:      SerializeOption(pass, input.Doc, SerializeCommentGroup),
			TokenPos: SerializePos(pass, input.TokPos),
			Lparen:   SerializePos(pass, input.Lparen),
			Rparen:   SerializePos(pass, input.Rparen),
			Tok:      input.Tok.String(),
			Specs:    SerializeList(pass, input.Specs, SerializeSpec),
			Node: Node{
				Type: NodeTypeGenDecl,
				Ref:  pass.refCount,
				Loc:  SerializeLoc(pass, input),
			},
		}
	}

	if pass.conf[CacheRef] != nil {
		return SerRefLookup(pass, input, SerializeGenDecl)
	}

	return serializeGenDecl(pass, input)
}

func SerializeFuncDecl(pass *serPass, input *ast.FuncDecl) *FuncDecl {
	serializeFuncDecl := func(pass *serPass, input *ast.FuncDecl) *FuncDecl {
		return &FuncDecl{
			Doc:  SerializeOption(pass, input.Doc, SerializeCommentGroup),
			Recv: SerializeOption(pass, input.Recv, SerializeFieldList),
			Name: SerializeOption(pass, input.Name, SerializeIdent),
			Type: SerializeOption(pass, input.Type, SerializeFuncType),
			Body: SerializeOption(pass, input.Body, SerializeBlockStmt),
			Node: Node{
				Type: NodeTypeFuncDecl,
				Ref:  pass.refCount,
				Loc:  SerializeLoc(pass, input),
			},
		}
	}

	if pass.conf[CacheRef] != nil {
		return SerRefLookup(pass, input, SerializeFuncDecl)
	}

	return serializeFuncDecl(pass, input)
}

func SerializeDecl(pass *serPass, decl ast.Decl) Decl {
	switch d := decl.(type) {
	case *ast.BadDecl:
		return SerializeBadDecl(pass, d)
	case *ast.GenDecl:
		return SerializeGenDecl(pass, d)
	case *ast.FuncDecl:
		return SerializeFuncDecl(pass, d)
	default:
		return nil
	}
}

// ----------------- Files and Packages ----------------- //

func SerializeFile(pass *serPass, input *ast.File) *File {
	var scope *Scope
	if pass.conf[ResolveScope] != nil && input.Scope != nil {
		scope = SerializeScope(pass, input.Scope)
	}

	return &File{
		Doc:        SerializeOption(pass, input.Doc, SerializeCommentGroup),
		Name:       SerializeOption(pass, input.Name, SerializeIdent),
		Decls:      SerializeList(pass, input.Decls, SerializeDecl),
		Size:       calcFileSize(pass, input),
		FileStart:  SerializePos(pass, input.FileStart),
		FileEnd:    SerializePos(pass, input.FileEnd),
		Scope:      scope,
		Imports:    SerializeList(pass, input.Imports, SerializeImportSpec),
		Unresolved: SerializeList(pass, input.Unresolved, SerializeIdent),
		Package:    SerializePos(pass, input.Package),
		Comments:   SerializeList(pass, input.Comments, SerializeCommentGroup),
		GoVersion:  input.GoVersion,
		Node: Node{
			Type: NodeTypeFile,
			Ref:  pass.refCount,
			Loc:  SerializeLoc(pass, input),
		},
	}
}

func calcFileSize(pass *serPass, input *ast.File) int {
	position := pass.fset.PositionFor(input.Name.NamePos, false)
	content, err := os.ReadFile(position.Filename)
	if err != nil {
		return 1<<_GOARCH() - 2
	}

	return len(content)
}

func SerializePackage(pass *serPass, input *ast.Package) *Package {
	var scope *Scope
	if pass.conf[ResolveScope] != nil && input.Scope != nil {
		scope = SerializeScope(pass, input.Scope)
	}

	return &Package{
		Name:  input.Name,
		Scope: scope,
		// cant use SerializeMap func, because `*ast.Object` does not satisfy the interface `ast.Node`.
		Imports: serializeImports(pass, input.Imports),
		Files:   SerializeMap(pass, input.Files, SerializeFile),
	}
}

func serializeImports(pass *serPass, inputMap map[string]*ast.Object) map[string]*Object {
	result := make(map[string]*Object, len(inputMap))
	for k, v := range inputMap {
		result[k] = SerializeObject(pass, v)
	}

	return result
}
