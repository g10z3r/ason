package ason

// NodeType constants define the string representation of various AST node types.
const (
	NodeTypeInvalid        = "Invalid"
	NodeTypeFile           = "File"
	NodeTypeComment        = "Comment"
	NodeTypeCommentGroup   = "CommentGroup"
	NodeTypeIdent          = "Ident"
	NodeTypeBasicLit       = "BasicLit"
	NodeTypeFuncLit        = "FuncLit"
	NodeTypeCompositeLit   = "CompositeLit"
	NodeTypeField          = "Field"
	NodeTypeFieldList      = "FieldList"
	NodeTypeEllipsis       = "Ellipsis"
	NodeTypeBadExpr        = "BadExpr"
	NodeTypeParenExpr      = "ParenExpr"
	NodeTypeSelectorExpr   = "SelectorExpr"
	NodeTypeIndexExpr      = "IndexExpr"
	NodeTypeIndexListExpr  = "IndexListExpr"
	NodeTypeSliceExpr      = "SliceExpr"
	NodeTypeTypeAssertExpr = "TypeAssertExpr"
	NodeTypeCallExpr       = "CallExpr"
	NodeTypeStarExpr       = "StarExpr"
	NodeTypeUnaryExpr      = "UnaryExpr"
	NodeTypeBinaryExpr     = "BinaryExpr"
	NodeTypeKeyValueExpr   = "KeyValueExpr"
	NodeTypeArrayType      = "ArrayType"
	NodeTypeStructType     = "StructType"
	NodeTypeFuncType       = "FuncType"
	NodeTypeInterfaceType  = "InterfaceType"
	NodeTypeMapType        = "MapType"
	NodeTypeChanType       = "ChanType"
	NodeTypeBadStmt        = "BadStmt"
	NodeTypeDeclStmt       = "DeclStmt"
	NodeTypeEmptyStmt      = "EmptyStmt"
	NodeTypeLabeledStmt    = "LabeledStmt"
	NodeTypeExprStmt       = "ExprStmt"
	NodeTypeSendStmt       = "SendStmt"
	NodeTypeIncDecStmt     = "IncDecStmt"
	NodeTypeAssignStmt     = "AssignStmt"
	NodeTypeGoStmt         = "GoStmt"
	NodeTypeDeferStmt      = "DeferStmt"
	NodeTypeReturnStmt     = "ReturnStmt"
	NodeTypeBranchStmt     = "BranchStmt"
	NodeTypeBlockStmt      = "BlockStmt"
	NodeTypeIfStmt         = "IfStmt"
	NodeTypeCaseClause     = "CaseClause"
	NodeTypeSwitchStmt     = "SwitchStmt"
	NodeTypeTypeSwitchStmt = "TypeSwitchStmt"
	NodeTypeCommClause     = "CommClause"
	NodeTypeSelectStmt     = "SelectStmt"
	NodeTypeForStmt        = "ForStmt"
	NodeTypeRangeStmt      = "RangeStmt"
	NodeTypeImportSpec     = "ImportSpec"
	NodeTypeValueSpec      = "ValueSpec"
	NodeTypeTypeSpec       = "TypeSpec"
	NodeTypeGenDecl        = "GenDecl"
	NodeTypeBadDecl        = "BadDecl"
	NodeTypeFuncDecl       = "FuncDecl"
)

type Ason interface {
	asonNode()
}

type Expr interface {
	Ason
	exprNode()
}

type Spec interface {
	Ason
	specNode()
}
type Stmt interface {
	Ason
	stmtNode()
}

type Decl interface {
	Ason
	declNode()
}

type Node struct {
	Ref  uint   `json:"_ref"`
	Type string `json:"_type"`
	Loc  *Loc   `json:"_loc"`
}

// ----------------- Comments ----------------- //

type (
	Comment struct {
		Slash Pos    // position of "/" starting the comment
		Text  string // comment text (excluding '\n' for //-style comments)

		Node
	}

	// A CommentGroup represents a sequence of comments
	// with no other tokens and no empty lines between.
	CommentGroup struct {
		List []*Comment

		Node
	}
)

func (*Comment) asonNode()      {}
func (*CommentGroup) asonNode() {}

// ----------------- Expressions ----------------- //

type Field struct {
	Doc     *CommentGroup // associated documentation; or nil
	Names   []*Ident      // field/method/(type) parameter names; or nil
	Type    Expr          // field/method/parameter type; or nil
	Tag     *BasicLit     // field tag; or nil
	Comment *CommentGroup // line comments; or nil

	Node
}

// A FieldList represents a list of Fields, enclosed by parentheses,
// curly braces, or square brackets.
type FieldList struct {
	Opening Pos      // position of opening parenthesis/brace/bracket, if any
	List    []*Field // field list; or nil
	Closing Pos      // position of closing parenthesis/brace/bracket, if any

	Node
}

func (*Field) asonNode()     {}
func (*FieldList) asonNode() {}

type (
	// An Ident node represents an identifier.
	Ident struct {
		NamePos Pos     // identifier position
		Name    string  // identifier name
		Obj     *Object // denoted object; or nil

		Node
	}

	// A BasicLit node represents a literal of basic type.
	BasicLit struct {
		ValuePos Pos    // literal position
		Kind     string // token.INT, token.FLOAT, token.IMAG, token.CHAR, or token.STRING
		Value    string // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`

		Node
	}

	// A FuncLit node represents a function literal.
	FuncLit struct {
		Type *FuncType  // function type
		Body *BlockStmt // function body

		Node
	}

	// A CompositeLit node represents a composite literal.
	CompositeLit struct {
		Type       Expr   // literal type; or nil
		Lbrace     Pos    // position of "{"
		Elts       []Expr // list of composite elements; or nil
		Rbrace     Pos    // position of "}"
		Incomplete bool   // true if (source) expressions are missing in the Elts list

		Node
	}

	// An Ellipsis node stands for the "..." type in a
	// parameter list or the "..." length in an array type.
	Ellipsis struct {
		Ellipsis Pos  // position of "..."
		Elt      Expr // ellipsis element type (parameter lists only); or nil

		Node
	}

	// A BadExpr node is a placeholder for an expression containing
	// syntax errors for which a correct expression node cannot be
	// created.
	BadExpr struct {
		From, To Pos // position range of bad expression

		Node
	}

	// A ParenExpr node represents a parenthesized expression.
	ParenExpr struct {
		Lparen Pos  // position of "("
		X      Expr // parenthesized expression
		Rparen Pos  // position of ")"

		Node
	}

	// A SelectorExpr node represents an expression followed by a selector.
	SelectorExpr struct {
		X   Expr   // expression
		Sel *Ident // field selector

		Node
	}

	// An IndexExpr node represents an expression followed by an index.
	IndexExpr struct {
		X      Expr // expression
		Lbrack Pos  // position of "["
		Index  Expr // index expression
		Rbrack Pos  // position of "]"

		Node
	}

	// An IndexListExpr node represents an expression followed by multiple indices.
	IndexListExpr struct {
		X       Expr   // expression
		Lbrack  Pos    // position of "["
		Indices []Expr // index expressions
		Rbrack  Pos    // position of "]"

		Node
	}

	// A SliceExpr node represents an expression followed by slice indices.
	SliceExpr struct {
		X      Expr // expression
		Lbrack Pos  // position of "["
		Low    Expr // begin of slice range; or nil
		High   Expr // end of slice range; or nil
		Max    Expr // maximum capacity of slice; or nil
		Slice3 bool // true if 3-index slice (2 colons present)
		Rbrack Pos  // position of "]"

		Node
	}

	// A TypeAssertExpr node represents an expression followed by a type assertion.
	TypeAssertExpr struct {
		X      Expr // expression
		Lparen Pos  // position of "("
		Type   Expr // asserted type; nil means type switch X.(type)
		Rparen Pos  // position of ")"

		Node
	}

	// A CallExpr node represents an expression followed by an argument list.
	CallExpr struct {
		Fun      Expr   // function expression
		Lparen   Pos    // position of "("
		Args     []Expr // function arguments; or nil
		Ellipsis Pos    // position of "..." (token.NoPos if there is no "...")
		Rparen   Pos    // position of ")"

		Node
	}

	// A StarExpr node represents an expression of the form "*" Expression.
	// Semantically it could be a unary "*" expression, or a pointer type.
	StarExpr struct {
		Star Pos  // position of "*"
		X    Expr // operand

		Node
	}

	// A UnaryExpr node represents a unary expression.
	// Unary "*" expressions are represented via StarExpr nodes.
	//
	UnaryExpr struct {
		OpPos Pos    // position of Op
		Op    string // operator (token)
		X     Expr   // operand

		Node
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X     Expr   // left operand
		OpPos Pos    // position of Op
		Op    string // operator (token)
		Y     Expr   // right operand

		Node
	}

	// A KeyValueExpr node represents (key : value) pairs
	// in composite literals.
	//
	KeyValueExpr struct {
		Key   Expr
		Colon Pos // position of ":"
		Value Expr

		Node
	}
)

// ----------------- Types ----------------- //

// A type is represented by a tree consisting of one
// or more of the following type-specific expression nodes.
// Pointer types are represented via StarExpr nodes.
type (
	// An ArrayType node represents an array or slice type.
	ArrayType struct {
		Lbrack Pos  // position of "["
		Len    Expr // Ellipsis node for [...]T array types, nil for slice types
		Elt    Expr // element type

		Node
	}

	// A StructType node represents a struct type.
	StructType struct {
		Struct     Pos        // position of "struct" keyword
		Fields     *FieldList // list of field declarations
		Incomplete bool       // true if (source) fields are missing in the Fields list

		Node
	}

	// A FuncType node represents a function type.
	FuncType struct {
		Func       Pos        // position of "func" keyword (token.NoPos if there is no "func")
		TypeParams *FieldList // type parameters; or nil
		Params     *FieldList // (incoming) parameters; non-nil
		Results    *FieldList // (outgoing) results; or nil

		Node
	}

	// An InterfaceType node represents an interface type.
	InterfaceType struct {
		Interface  Pos        // position of "interface" keyword
		Methods    *FieldList // list of embedded interfaces, methods, or types
		Incomplete bool       // true if (source) methods or types are missing in the Methods list

		Node
	}

	// A MapType node represents a map type.
	MapType struct {
		Map   Pos // position of "map" keyword
		Key   Expr
		Value Expr

		Node
	}

	// A ChanType node represents a channel type.
	ChanType struct {
		Begin Pos  // position of "chan" keyword or "<-" (whichever comes first)
		Arrow Pos  // position of "<-" (token.NoPos if there is no "<-")
		Dir   int  // channel direction
		Value Expr // value type

		Node
	}
)

func (*Ident) asonNode()          {}
func (*BasicLit) asonNode()       {}
func (*BadExpr) asonNode()        {}
func (*Ellipsis) asonNode()       {}
func (*CompositeLit) asonNode()   {}
func (*SelectorExpr) asonNode()   {}
func (*ParenExpr) asonNode()      {}
func (*IndexExpr) asonNode()      {}
func (*IndexListExpr) asonNode()  {}
func (*SliceExpr) asonNode()      {}
func (*TypeAssertExpr) asonNode() {}
func (*CallExpr) asonNode()       {}
func (*StarExpr) asonNode()       {}
func (*UnaryExpr) asonNode()      {}
func (*BinaryExpr) asonNode()     {}
func (*KeyValueExpr) asonNode()   {}
func (*ArrayType) asonNode()      {}
func (*StructType) asonNode()     {}
func (*FuncType) asonNode()       {}
func (*InterfaceType) asonNode()  {}
func (*MapType) asonNode()        {}
func (*ChanType) asonNode()       {}
func (*FuncLit) asonNode()        {}

func (*Ident) exprNode()          {}
func (*BasicLit) exprNode()       {}
func (*BadExpr) exprNode()        {}
func (*Ellipsis) exprNode()       {}
func (*CompositeLit) exprNode()   {}
func (*SelectorExpr) exprNode()   {}
func (*ParenExpr) exprNode()      {}
func (*IndexExpr) exprNode()      {}
func (*IndexListExpr) exprNode()  {}
func (*SliceExpr) exprNode()      {}
func (*TypeAssertExpr) exprNode() {}
func (*CallExpr) exprNode()       {}
func (*StarExpr) exprNode()       {}
func (*UnaryExpr) exprNode()      {}
func (*BinaryExpr) exprNode()     {}
func (*KeyValueExpr) exprNode()   {}
func (*ArrayType) exprNode()      {}
func (*StructType) exprNode()     {}
func (*FuncType) exprNode()       {}
func (*InterfaceType) exprNode()  {}
func (*MapType) exprNode()        {}
func (*ChanType) exprNode()       {}
func (*FuncLit) exprNode()        {}

// ----------------- Statements ----------------- //

type (
	// A BadStmt node is a placeholder for statements containing
	// syntax errors for which no correct statement nodes can be created.
	BadStmt struct {
		From, To Pos // position range of bad statement

		Node
	}

	// A DeclStmt node represents a declaration in a statement list.
	DeclStmt struct {
		Decl Decl // *GenDecl with CONST, TYPE, or VAR token

		Node
	}

	// An EmptyStmt node represents an empty statement.
	// The "position" of the empty statement is the position
	// of the immediately following (explicit or implicit) semicolon.
	EmptyStmt struct {
		Semicolon Pos  // position of following ";"
		Implicit  bool // if set, ";" was omitted in the source

		Node
	}

	// A LabeledStmt node represents a labeled statement.
	LabeledStmt struct {
		Label *Ident
		Colon Pos // position of ":"
		Stmt  Stmt

		Node
	}

	// An ExprStmt node represents a (stand-alone) expression in a statement list.
	ExprStmt struct {
		X Expr // expression

		Node
	}

	// A SendStmt node represents a send statement.
	SendStmt struct {
		Chan  Expr
		Arrow Pos // position of "<-"
		Value Expr

		Node
	}

	// An IncDecStmt node represents an increment or decrement statement.
	IncDecStmt struct {
		X      Expr
		TokPos Pos    // position of Tok
		Tok    string // INC or DEC

		Node
	}

	// An AssignStmt node represents an assignment or a short variable declaration.
	AssignStmt struct {
		Lhs    []Expr
		TokPos Pos    // position of Tok
		Tok    string // assignment token, DEFINE
		Rhs    []Expr

		Node
	}

	// A GoStmt node represents a go statement.
	GoStmt struct {
		Go   Pos // position of "go" keyword
		Call *CallExpr

		Node
	}

	// A DeferStmt node represents a defer statement.
	DeferStmt struct {
		Defer Pos // position of "defer" keyword
		Call  *CallExpr

		Node
	}

	// A ReturnStmt node represents a return statement.
	ReturnStmt struct {
		Return  Pos    // position of "return" keyword
		Results []Expr // result expressions; or nil

		Node
	}

	// A BranchStmt node represents a break, continue, goto,
	// or fallthrough statement.
	BranchStmt struct {
		TokPos Pos    // position of Tok
		Tok    string // keyword token (BREAK, CONTINUE, GOTO, FALLTHROUGH)
		Label  *Ident // label name; or nil

		Node
	}

	// A BlockStmt node represents a braced statement list.
	BlockStmt struct {
		Lbrace Pos // position of "{"
		List   []Stmt
		Rbrace Pos // position of "}", if any (may be absent due to syntax error)

		Node
	}

	// An IfStmt node represents an if statement.
	IfStmt struct {
		If   Pos  // position of "if" keyword
		Init Stmt // initialization statement; or nil
		Cond Expr // condition
		Body *BlockStmt
		Else Stmt // else branch; or nil

		Node
	}

	// A CaseClause represents a case of an expression or type switch statement.
	CaseClause struct {
		Case  Pos    // position of "case" or "default" keyword
		List  []Expr // list of expressions or types; nil means default case
		Colon Pos    // position of ":"
		Body  []Stmt // statement list; or nil

		Node
	}

	// A SwitchStmt node represents an expression switch statement.
	SwitchStmt struct {
		Switch Pos        // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Tag    Expr       // tag expression; or nil
		Body   *BlockStmt // CaseClauses only

		Node
	}

	// A TypeSwitchStmt node represents a type switch statement.
	TypeSwitchStmt struct {
		Switch Pos        // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Assign Stmt       // x := y.(type) or y.(type)
		Body   *BlockStmt // CaseClauses only

		Node
	}

	// A CommClause node represents a case of a select statement.
	CommClause struct {
		Case  Pos    // position of "case" or "default" keyword
		Comm  Stmt   // send or receive statement; nil means default case
		Colon Pos    // position of ":"
		Body  []Stmt // statement list; or nil

		Node
	}

	// A SelectStmt node represents a select statement.
	SelectStmt struct {
		Select Pos        // position of "select" keyword
		Body   *BlockStmt // CommClauses only

		Node
	}

	// A ForStmt represents a for statement.
	ForStmt struct {
		For  Pos  // position of "for" keyword
		Init Stmt // initialization statement; or nil
		Cond Expr // condition; or nil
		Post Stmt // post iteration statement; or nil
		Body *BlockStmt

		Node
	}

	// A RangeStmt represents a for statement with a range clause.
	RangeStmt struct {
		For        Pos    // position of "for" keyword
		Key, Value Expr   // Key, Value may be nil
		TokPos     Pos    // position of Tok; invalid if Key == nil
		Tok        string // ILLEGAL if Key == nil, ASSIGN, DEFINE
		Range      Pos    // position of "range" keyword
		X          Expr   // value to range over
		Body       *BlockStmt

		Node
	}
)

func (*BadStmt) asonNode()        {}
func (*DeclStmt) asonNode()       {}
func (*EmptyStmt) asonNode()      {}
func (*LabeledStmt) asonNode()    {}
func (*ExprStmt) asonNode()       {}
func (*SendStmt) asonNode()       {}
func (*IncDecStmt) asonNode()     {}
func (*AssignStmt) asonNode()     {}
func (*GoStmt) asonNode()         {}
func (*DeferStmt) asonNode()      {}
func (*ReturnStmt) asonNode()     {}
func (*BranchStmt) asonNode()     {}
func (*BlockStmt) asonNode()      {}
func (*IfStmt) asonNode()         {}
func (*CaseClause) asonNode()     {}
func (*SwitchStmt) asonNode()     {}
func (*TypeSwitchStmt) asonNode() {}
func (*CommClause) asonNode()     {}
func (*SelectStmt) asonNode()     {}
func (*ForStmt) asonNode()        {}
func (*RangeStmt) asonNode()      {}

func (*BadStmt) stmtNode()        {}
func (*DeclStmt) stmtNode()       {}
func (*EmptyStmt) stmtNode()      {}
func (*LabeledStmt) stmtNode()    {}
func (*ExprStmt) stmtNode()       {}
func (*SendStmt) stmtNode()       {}
func (*IncDecStmt) stmtNode()     {}
func (*AssignStmt) stmtNode()     {}
func (*GoStmt) stmtNode()         {}
func (*DeferStmt) stmtNode()      {}
func (*ReturnStmt) stmtNode()     {}
func (*BranchStmt) stmtNode()     {}
func (*BlockStmt) stmtNode()      {}
func (*IfStmt) stmtNode()         {}
func (*CaseClause) stmtNode()     {}
func (*SwitchStmt) stmtNode()     {}
func (*TypeSwitchStmt) stmtNode() {}
func (*CommClause) stmtNode()     {}
func (*SelectStmt) stmtNode()     {}
func (*ForStmt) stmtNode()        {}
func (*RangeStmt) stmtNode()      {}

// ----------------- Specifications ----------------- //

type (
	// An ImportSpec node represents a single package import.
	ImportSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Name    *Ident        // local package name (including "."); or nil
		Path    *BasicLit     // import path
		Comment *CommentGroup // line comments; or nil
		EndPos  Pos           // end of spec (overrides Path.Pos if nonzero)

		Node
	}

	// A ValueSpec node represents a constant or variable declaration
	ValueSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Names   []*Ident      // value names (len(Names) > 0)
		Type    Expr          // value type; or nil
		Values  []Expr        // initial values; or nil
		Comment *CommentGroup // line comments; or nil

		Node
	}

	// A TypeSpec node represents a type declaration.
	TypeSpec struct {
		Doc        *CommentGroup // associated documentation; or nil
		Name       *Ident        // type name
		TypeParams *FieldList    // type parameters; or nil
		Assign     Pos           // position of '=', if any
		Type       Expr          // *Ident, *ParenExpr, *SelectorExpr, *StarExpr, or any of the *XxxTypes
		Comment    *CommentGroup // line comments; or nil

		Node
	}
)

func (*ImportSpec) asonNode() {}
func (*ValueSpec) asonNode()  {}
func (*TypeSpec) asonNode()   {}

func (*ImportSpec) specNode() {}
func (*ValueSpec) specNode()  {}
func (*TypeSpec) specNode()   {}

// ----------------- Declarations ----------------- //

// A declaration is represented by one of the following declaration nodes.
type (
	// A BadDecl node is a placeholder for a declaration containing
	// syntax errors for which a correct declaration node cannot be
	// created.
	BadDecl struct {
		From, To Pos // position range of bad declaration

		Node
	}

	// A GenDecl node (generic declaration node) represents an import,
	// constant, type or variable declaration. A valid Lparen position
	// (Lparen.IsValid()) indicates a parenthesized declaration.
	GenDecl struct {
		Doc      *CommentGroup // associated documentation; or nil
		TokenPos Pos           // position of Tok
		Tok      string        // IMPORT, CONST, TYPE, or VAR
		Lparen   Pos           // position of '(', if any
		Specs    []Spec
		Rparen   Pos // position of ')', if any

		Node
	}

	// A FuncDecl node represents a function declaration.
	FuncDecl struct {
		Doc  *CommentGroup // associated documentation; or nil
		Recv *FieldList    // receiver (methods); or nil (functions)
		Name *Ident        // function/method name
		Type *FuncType     // function signature: type and value parameters, results, and position of "func" keyword
		Body *BlockStmt    // function body; or nil for external (non-Go) function

		Node
	}
)

func (*BadDecl) asonNode()  {}
func (*GenDecl) asonNode()  {}
func (*FuncDecl) asonNode() {}

func (*BadDecl) declNode()  {}
func (*GenDecl) declNode()  {}
func (*FuncDecl) declNode() {}

// ----------------- Files and Packages ----------------- //

// A File node represents a Go source file.
type File struct {
	Doc                *CommentGroup   // associated documentation; or empty
	Name               *Ident          // package name
	Decls              []Decl          // top-level declarations
	Size               int             // file size in bytes
	FileStart, FileEnd Pos             // start and end of entire file
	Scope              *Scope          // package scope (this file only)
	Imports            []*ImportSpec   // imports in this file
	Unresolved         []*Ident        // unresolved identifiers in this file
	Package            Pos             // position of "package" keyword
	Comments           []*CommentGroup // list of all comments in the source file
	GoVersion          string          // minimum Go version required by go:build or +build directives

	Node
}

// A Package node represents a set of source files
// collectively building a Go package.
type Package struct {
	Name    string             // package name
	Scope   *Scope             // package scope across all files
	Imports map[string]*Object // map of package id -> package object
	Files   map[string]*File   // Go source files by filename
}

func (*File) asonNode()    {}
func (*Package) asonNode() {}
