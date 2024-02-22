package ason

import (
	"go/ast"
	"go/token"
	"strconv"
)

func _GOARCH() int {
	return strconv.IntSize
}

type Pos interface {
	// Offset is an absolute position of a character in the file text.
	Offset() int
	// Line is a line number in the file where this piece of code is located.
	Line() int
	// Column ia s character position in a specific line, starting from zero.
	Column() int
}

type (
	NoPos    int
	Position struct {
		Location [3]int // set of coordinates used to indicate a specific location in the source code; the order is `Offset`, `Line`, `Column`
		Filename string
	}
)

func (*NoPos) Offset() int { return 0 }
func (*NoPos) Line() int   { return 0 }
func (*NoPos) Column() int { return 0 }

func (p *Position) Offset() int { return p.Location[0] }
func (p *Position) Line() int   { return p.Location[1] }
func (p *Position) Column() int { return p.Location[2] }

// Checking whether the position exists.
func IsPosValid(pos Pos) bool {
	if pos == nil {
		return false
	}

	if pos.Offset() == 0 {
		return false
	}

	return true
}

func NewPosition(pos token.Position) *Position {
	return &Position{
		Filename: pos.Filename,
		Location: [3]int{
			pos.Offset,
			pos.Line,
			pos.Column,
		},
	}
}

type Loc struct {
	_     [0]int
	Start Pos `json:"Start"`
	End   Pos `json:"End"`
}

var tokens = map[string]token.Token{
	"ILLEGAL": token.ILLEGAL,

	"EOF":     token.EOF,
	"COMMENT": token.COMMENT,

	"IDENT":  token.IDENT,
	"INT":    token.INT,
	"FLOAT":  token.FLOAT,
	"IMAG":   token.IMAG,
	"CHAR":   token.CHAR,
	"STRING": token.STRING,

	"+": token.ADD,
	"-": token.SUB,
	"*": token.MUL,
	"/": token.QUO,
	"%": token.REM,

	"&":  token.AND,
	"|":  token.OR,
	"^":  token.XOR,
	"<<": token.SHL,
	">>": token.SHR,
	"&^": token.AND_NOT,

	"+=": token.ADD_ASSIGN,
	"-=": token.SUB_ASSIGN,
	"*=": token.MUL_ASSIGN,
	"/=": token.QUO_ASSIGN,
	"%=": token.REM_ASSIGN,

	"&=":  token.AND_ASSIGN,
	"|=":  token.OR_ASSIGN,
	"^=":  token.XOR_ASSIGN,
	"<<=": token.SHL_ASSIGN,
	">>=": token.SHR_ASSIGN,
	"&^=": token.AND_NOT_ASSIGN,

	"&&": token.LAND,
	"||": token.LOR,
	"<-": token.ARROW,
	"++": token.INC,
	"--": token.DEC,

	"==": token.EQL,
	"<":  token.LSS,
	">":  token.GTR,
	"=":  token.ASSIGN,
	"!":  token.NOT,

	"!=":  token.NEQ,
	"<=":  token.LEQ,
	">=":  token.GEQ,
	":=":  token.DEFINE,
	"...": token.ELLIPSIS,

	"(": token.LPAREN,
	"[": token.LBRACK,
	"{": token.LBRACE,
	",": token.COMMA,
	".": token.PERIOD,

	")": token.RPAREN,
	"]": token.RBRACK,
	"}": token.RBRACE,
	";": token.SEMICOLON,
	":": token.COLON,

	"break":    token.BREAK,
	"case":     token.CASE,
	"chan":     token.CHAN,
	"const":    token.CONST,
	"continue": token.CONTINUE,

	"default":     token.DEFAULT,
	"defer":       token.DEFER,
	"else":        token.ELSE,
	"fallthrough": token.FALLTHROUGH,
	"for":         token.FOR,

	"func":   token.FUNC,
	"go":     token.GO,
	"goto":   token.GOTO,
	"if":     token.IF,
	"import": token.IMPORT,

	"interface": token.INTERFACE,
	"map":       token.MAP,
	"package":   token.PACKAGE,
	"range":     token.RANGE,
	"return":    token.RETURN,

	"select": token.SELECT,
	"struct": token.STRUCT,
	"switch": token.SWITCH,
	"type":   token.TYPE,
	"var":    token.VAR,

	"~": token.TILDE,
}

var objKinds = map[string]ast.ObjKind{
	"bad":     ast.Bad,
	"package": ast.Pkg,
	"const":   ast.Con,
	"type":    ast.Typ,
	"var":     ast.Var,
	"func":    ast.Fun,
	"label":   ast.Lbl,
}

type Object struct {
	Kind string
	Name string // declared name
}

type Scope struct {
	Outer   *Scope
	Objects map[string]*Object
}
