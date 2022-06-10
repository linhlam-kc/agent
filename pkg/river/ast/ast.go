package ast

import (
	"fmt"

	"github.com/grafana/agent/pkg/river/token"
)

// Node is an overall node in the AST.
type Node interface {
	astNode()
}

// Stmt is a type of statement within the body of a file or block.
type Stmt interface {
	Node
	astStmt()
}

// Expr is an overall expression in the AST.
type Expr interface {
	Node
	astExpr()
}

// File is a parsed file.
type File struct {
	Name string // Filename provided to parser
	Body []Stmt // Content of File
}

// AttributeStmt is a key-value pair being set in a body or a BlockStmt.
type AttributeStmt struct {
	Name    string
	NamePos token.Pos
	Value   Expr
}

// BlockStmt is a declarative of a block inside of a body.
type BlockStmt struct {
	Name    []string
	NamePos token.Pos
	Label   string
	Body    []Stmt

	LCurly, RCurly token.Pos
}

// LiteralExpr is a constant value of a specific type.
type LiteralExpr struct {
	Kind     token.Token
	Value    string
	ValuePos token.Pos
}

// ArrayExpr is an array of values.
type ArrayExpr struct {
	Elements []Expr

	LBracket, RBracket token.Pos
}

// ObjectExpr is an object.
type ObjectExpr struct {
	Fields         []*ObjectField
	LCurly, RCurly token.Pos
}

type ObjectField struct {
	Name    string
	NamePos token.Pos
	Value   Expr
}

// IdentifierExpr refers to a value by name.
type IdentifierExpr struct {
	Name    string
	NamePos token.Pos
}

// AccessExpr accesses a field in an object value by name.
type AccessExpr struct {
	Value   Expr
	Name    string
	NamePos token.Pos
}

// IndexExpr accesses an index in an array value.
type IndexExpr struct {
	Value Expr
	Index Expr

	LBracket, RBracket token.Pos
}

// CallExpr invokes a function with a set of arguments.
type CallExpr struct {
	Value Expr
	Args  []Expr

	LParen, RParen token.Pos
}

// UnaryExpr performs a unary operation on a single value.
type UnaryExpr struct {
	Kind       token.Token
	Expression Expr
	KindPos    token.Pos
}

// BinaryExpr performs a binary operation on two values.
type BinaryExpr struct {
	Kind        token.Token
	KindPos     token.Pos
	Left, Right Expr
}

// Type checks

var (
	_ Node = (*File)(nil)
	_ Node = (*AttributeStmt)(nil)
	_ Node = (*BlockStmt)(nil)
	_ Node = (*LiteralExpr)(nil)
	_ Node = (*ArrayExpr)(nil)
	_ Node = (*ObjectExpr)(nil)
	_ Node = (*ObjectField)(nil)
	_ Node = (*IdentifierExpr)(nil)
	_ Node = (*AccessExpr)(nil)
	_ Node = (*IndexExpr)(nil)
	_ Node = (*CallExpr)(nil)
	_ Node = (*UnaryExpr)(nil)
	_ Node = (*BinaryExpr)(nil)

	_ Stmt = (*AttributeStmt)(nil)
	_ Stmt = (*BlockStmt)(nil)

	_ Expr = (*LiteralExpr)(nil)
	_ Expr = (*ArrayExpr)(nil)
	_ Expr = (*ObjectExpr)(nil)
	_ Expr = (*IdentifierExpr)(nil)
	_ Expr = (*AccessExpr)(nil)
	_ Expr = (*IndexExpr)(nil)
	_ Expr = (*CallExpr)(nil)
	_ Expr = (*UnaryExpr)(nil)
	_ Expr = (*BinaryExpr)(nil)
)

func (n *File) astNode()           {}
func (n *AttributeStmt) astNode()  {}
func (n *BlockStmt) astNode()      {}
func (n *LiteralExpr) astNode()    {}
func (n *ArrayExpr) astNode()      {}
func (n *ObjectExpr) astNode()     {}
func (n *ObjectField) astNode()    {}
func (n *IdentifierExpr) astNode() {}
func (n *AccessExpr) astNode()     {}
func (n *IndexExpr) astNode()      {}
func (n *CallExpr) astNode()       {}
func (n *UnaryExpr) astNode()      {}
func (n *BinaryExpr) astNode()     {}

func (n *AttributeStmt) astStmt() {}
func (n *BlockStmt) astStmt()     {}

func (n *LiteralExpr) astExpr()    {}
func (n *ArrayExpr) astExpr()      {}
func (n *ObjectExpr) astExpr()     {}
func (n *IdentifierExpr) astExpr() {}
func (n *AccessExpr) astExpr()     {}
func (n *IndexExpr) astExpr()      {}
func (n *CallExpr) astExpr()       {}
func (n *UnaryExpr) astExpr()      {}
func (n *BinaryExpr) astExpr()     {}

// StartPos returns the position of the first character belonging to a Node.
func StartPos(n Node) token.Pos {
	if n == nil {
		return 0
	}
	switch n := n.(type) {
	case *File:
		if len(n.Body) == 0 {
			return 0
		}
		return StartPos(n.Body[0])
	case *AttributeStmt:
		return n.NamePos
	case *BlockStmt:
		return n.NamePos
	case *LiteralExpr:
		return n.ValuePos
	case *ArrayExpr:
		return n.LBracket
	case *ObjectExpr:
		return n.LCurly
	case *ObjectField:
		return n.NamePos
	case *IdentifierExpr:
		return n.NamePos
	case *AccessExpr:
		return n.NamePos
	case *IndexExpr:
		return StartPos(n.Value)
	case *CallExpr:
		return StartPos(n.Value)
	case *UnaryExpr:
		return n.KindPos
	case *BinaryExpr:
		return StartPos(n.Left)
	default:
		panic(fmt.Sprintf("Unrecognized Node type %T", n))
	}
}

// EndPos returns the position of the first character immediately following a
// Node.
func EndPos(n Node) token.Pos {
	if n == nil {
		return 0
	}

	switch n := n.(type) {
	case *File:
		if len(n.Body) == 0 {
			return 0
		}
		return EndPos(n.Body[len(n.Body)-1])
	case *AttributeStmt:
		return EndPos(n.Value)
	case *BlockStmt:
		return n.RCurly
	case *LiteralExpr:
		return n.ValuePos + token.Pos(len(n.Value))
	case *ArrayExpr:
		return n.RBracket
	case *ObjectExpr:
		return n.RCurly
	case *ObjectField:
		return EndPos(n.Value)
	case *IdentifierExpr:
		return n.NamePos + token.Pos(len(n.Name))
	case *AccessExpr:
		return EndPos(n.Value)
	case *IndexExpr:
		return n.RBracket
	case *CallExpr:
		return n.RParen
	case *UnaryExpr:
		return EndPos(n.Expression)
	case *BinaryExpr:
		return EndPos(n.Right)
	default:
		panic(fmt.Sprintf("Unrecognized Node type %T", n))
	}
}
