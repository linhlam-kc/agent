package parser

import (
	"fmt"

	"github.com/grafana/agent/pkg/river/ast"
	"github.com/grafana/agent/pkg/river/scanner"
	"github.com/grafana/agent/pkg/river/token"
)

// parser implements the River parser. The implementation of the parser differs
// slightly from the formal specification with the following changes:
//
//     1. The language accepted by the parser is slightly broader than the formal
//        specification to make the parser easier to write and maintain.
//
//     2. The grammar in the formal specification is simplified for reader
//        comprehension and contains left-recusion and ambiguities. The grammar
//        used in the actual implementation is converted to be LL(1).
//
// Exported methods may be safely used by caller as entrypoints for parsing.
// Non-exported parse methods are not guaranteed to be safe to be used as an
// entrypoint.
//
// Each Parse* and parse* method will describe the EBNF grammar being used for
// parsing that nonterminal.
//
// The parser will continue on encountering errors to allow a more complete
// list of errors to be returned to the user. The resulting AST should be
// discarded if errors were encountered during parsing.
type parser struct {
	file    *token.File
	errors  token.ErrorList
	scanner *scanner.Scanner

	pos token.Pos   // Current token position
	tok token.Token // Current token
	lit string      // Current token literal
}

// newParser creates a new parser which will parse the provided src.
func newParser(filename string, src []byte) *parser {
	file := token.NewFile(filename)

	p := &parser{
		file: file,
	}

	p.scanner = scanner.New(file, src, func(pos token.Pos, msg string) {
		p.errors.Add(&token.Error{
			Position: file.PositionFor(pos),
			Message:  msg,
		})
	}, 0)

	p.next()
	return p
}

// next advances the parser to the next token.
func (p *parser) next() { p.pos, p.tok, p.lit = p.scanner.Scan() }

// advance consumes tokens up to (but not including) the specified token.
func (p *parser) advance(to token.Token) {
	for p.tok != token.EOF {
		if p.tok == to {
			return
		}
		p.next()
	}
}

// advanceSet consumes tokens up to (but not including) one of the tokens in
// the "to" set.
func (p *parser) advanceSet(to map[token.Token]struct{}) {
	for p.tok != token.EOF {
		if _, inSet := to[p.tok]; inSet {
			return
		}
		p.next()
	}
}

func (p *parser) expect(t token.Token) (pos token.Pos, tok token.Token, lit string) {
	pos, tok, lit = p.pos, p.tok, p.lit
	if tok != t {
		p.addError(fmt.Sprintf("expected %s, got %s", t, p.tok))
	}
	p.next() // advance
	return
}

// allow consumes an optional token.
func (p *parser) allow(t token.Token) {
	if p.tok == t {
		p.next()
	}
}

func (p *parser) addError(msg string) {
	p.errors.Add(&token.Error{
		Position: p.file.PositionFor(p.pos),
		Message:  msg,
	})
}

// ParseFile parses an entire file.
//
//     File = Body .
func (p *parser) ParseFile() *ast.File {
	return &ast.File{
		Name: p.file.Name(),
		Body: p.parseBody(token.EOF),
	}
}

// parseBody parses a series of statements.
//
//     Body = [ Statement { terminator Statement } ] .
func (p *parser) parseBody(until token.Token) []ast.Stmt {
	var stmts []ast.Stmt

	for p.tok != until && p.tok != token.EOF {
		res := p.ParseStatement()
		if res != nil {
			stmts = append(stmts, res)
		}

		if p.tok == until {
			break
		}
		p.expect(token.TERMINATOR)
	}

	return stmts
}

// ParseStatement parses an individual statement within a block or file. A
// statement is either an attribute or a block.
//
//     Statement = Attribute | Block .
//     Attribute = identifier "=" Expression .
//     Block     = BlockName "{" Body "}" .
func (p *parser) ParseStatement() ast.Stmt {
	blockName := p.parseBlockName()
	if blockName == nil {
		// Skip to the next identifier to start a new Statement stanza
		p.advance(token.IDENT)
		return nil
	}

	switch p.tok {
	case token.ASSIGN: // Attribute
		p.next() // Consume "="

		if len(blockName.Fragments) != 1 {
			p.errors.Add(&token.Error{
				Position: p.file.PositionFor(blockName.Start),
				Message:  "attribute names may only consist of a single identifier",
			})
		} else if blockName.LabelPos != token.NoPos {
			p.errors.Add(&token.Error{
				Position: p.file.PositionFor(blockName.LabelPos),
				Message:  "attribute names may not have labels",
			})
		}

		return &ast.AttributeStmt{
			Name: &ast.IdentifierExpr{
				Name:    blockName.Fragments[0],
				NamePos: blockName.Start,
			},
			Value: p.ParseExpression(),
		}

	case token.LCURLY: // Block
		block := &ast.BlockStmt{
			Name:    blockName.Fragments,
			NamePos: blockName.Start,
			Label:   blockName.Label,
		}

		block.LCurly, _, _ = p.expect(token.LCURLY)
		block.Body = p.parseBody(token.RCURLY)
		block.RCurly, _, _ = p.expect(token.RCURLY)

		return block

	default:
		if blockName.ValidAttribute() {
			// This blockName could be an attribute or a block, inform user of both
			// cases.
			p.addError(fmt.Sprintf("expected attribute assignment or block body, got %s", p.tok))
		} else {
			// This blockName could only be a block.
			p.addError(fmt.Sprintf("expected block body, got %s", p.tok))
		}

		p.advance(token.IDENT)
		return nil
	}
}

// parseBlockName parses the name used for a block.
//
//     BlockName = identifier { "." identifier } [ string ] .
func (p *parser) parseBlockName() *blockName {
	if p.tok != token.IDENT {
		p.addError(fmt.Sprintf("expected identifier, got %s", p.tok))
		return nil
	}

	var n blockName

	n.Fragments = append(n.Fragments, p.lit)
	n.Start = p.pos
	p.next()

	// { "." identifier }
	for p.tok == token.DOT {
		p.next() // Consume "."

		if p.tok != token.IDENT {
			p.addError(fmt.Sprintf("expected identifier, got %s", p.tok))
		}

		n.Fragments = append(n.Fragments, p.lit)
		p.next()
	}

	if p.tok != token.ASSIGN && p.tok != token.LCURLY {
		// Allow for a string
		if p.tok == token.STRING {
			n.Label = p.lit[1 : len(p.lit)-1] // Strip quotes from label
			n.LabelPos = p.pos
		} else {
			p.addError(fmt.Sprintf("expected block label, got %s", p.tok))
		}

		p.next()
	}

	return &n
}

type blockName struct {
	Fragments []string // Name fragments (i.e., `a.b.c`)
	Label     string   // Optional user label

	Start    token.Pos
	LabelPos token.Pos
}

// ValidAttribute returns true if the blockName can be used as an
// attribute name.
func (n blockName) ValidAttribute() bool {
	return len(n.Fragments) == 1 && n.Label == ""
}

// ParseExpression parses a single expression.
//
//     Expression = BinOpExpr .
func (p *parser) ParseExpression() ast.Expr {
	return p.parseBinOp(1)
}

// parseBinOp is the entrypoint for binary expressions. If there is no binary
// expression in the current state, a single operand will be returned instead.
//
//     BinOpExpr = OrExpr .
//     OrExpr    = AndExpr { "||"   AndExpr } .
//     AndExpr   = CmpExpr { "&&"   CmpExpr } .
//     CmpExpr   = AddExpr { cmp_op AddExpr } .
//     AddExpr   = MulExpr { add_op MulExpr } .
//     MulExpr   = PowExpr { mul_op PowExpr } .
//
// parseBinOp avoids the need for multiple nonterminal functions by providing
// context for operator precedence in recursive calls. inPrec specifies the
// incoming operator precedence. On the first call to parseBinO, inPrec should
// be 1.
//
// parseBinOp handles left-associative operators, so PowExpr is handled by
// parsePowExpr.
func (p *parser) parseBinOp(inPrec int) ast.Expr {
	// The EBNF documented by the function can be generalized into:
	//
	//     CurPrecExpr = NextPrecExpr { cur_prec_op NextPrecExpr } .
	//
	// The code below implements this grammar, continually collecting everything
	// with the same precedence into the LHS of the expression while recursively
	// calling parseBinOp for higher-precedence operaions.

	lhs := p.parsePowExpr()

	for {
		tok, pos, prec := p.tok, p.pos, tokPrec(p.tok)
		if prec < inPrec {
			// The next operator is lower precedence; drop up a level in our stack.
			return lhs
		}
		p.next() // Consume the operator

		// Recurse with a higher-precedence level, which ensures operators with the
		// same precedence don't get handled in the recursive call.
		rhs := p.parseBinOp(prec + 1)

		lhs = &ast.BinaryExpr{
			Left:    lhs,
			Kind:    tok,
			KindPos: pos,
			Right:   rhs,
		}
	}
}

func tokPrec(t token.Token) int {
	switch t {
	case token.OR:
		return 1
	case token.AND:
		return 2
	case token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE:
		return 3
	case token.ADD, token.SUB:
		return 4
	case token.MUL, token.DIV, token.MOD:
		return 5
	case token.POW:
		return 6
	}
	return 0 // Lowest precedence
}

// parsePowExpr is like parseBinOp but handles the right-associative pow
// operator.
//
//   PowExpr = UnaryExpr [ "^" PowExpr ] .
func (p *parser) parsePowExpr() ast.Expr {
	lhs := p.parseUnaryExpr()

	if p.tok == token.POW {
		pos := p.pos
		p.next() // Consume "^"

		return &ast.BinaryExpr{
			Left:    lhs,
			Kind:    token.POW,
			KindPos: pos,
			Right:   p.parsePowExpr(),
		}
	}

	return lhs
}

// parseUnaryExpr parses a unary expression.
//
//     UnaryExpr = OperExpr | unary_op UnaryExpr .
//
//     OperExpr   = PrimaryExpr { AccessExpr | IndexExpr | CallExpr } .
//     AccessExpr = "." identifier .
//     IndexExpr  = "[" Expression "]"
//     CallExpr   = "(" [ ExpressionList ] ")"
func (p *parser) parseUnaryExpr() ast.Expr {
	if isUnaryOp(p.tok) {
		op, pos := p.tok, p.pos
		p.next() // consume op

		return &ast.UnaryExpr{
			Kind:       op,
			KindPos:    pos,
			Expression: p.parseUnaryExpr(),
		}
	}

	primary := p.parsePrimaryExpr()

NextOper:
	for {
		switch p.tok {
		case token.DOT: // AccessExpr
			p.expect(token.DOT)
			identPos, _, ident := p.expect(token.IDENT)

			primary = &ast.AccessExpr{
				Value:   primary,
				Name:    ident,
				NamePos: identPos,
			}

		case token.LBRACKET: // IndexExpr
			lBracket, _, _ := p.expect(token.LBRACKET)
			index := p.ParseExpression()
			rBracket, _, _ := p.expect(token.RBRACKET)

			primary = &ast.IndexExpr{
				Value:    primary,
				LBracket: lBracket,
				Index:    index,
				RBracket: rBracket,
			}

		case token.LPAREN: // CallExpr
			var args []ast.Expr

			lParen, _, _ := p.expect(token.LPAREN)
			if p.tok != token.RPAREN {
				args = p.parseExpressionList(token.RPAREN)
				p.allow(token.COMMA) // optional final comma
			}
			rParen, _, _ := p.expect(token.RPAREN)

			primary = &ast.CallExpr{
				Value:  primary,
				LParen: lParen,
				Args:   args,
				RParen: rParen,
			}

		default:
			break NextOper
		}
	}

	return primary
}

func isUnaryOp(t token.Token) bool {
	switch t {
	case token.NOT, token.SUB:
		return true
	}
	return false
}

// parsePrimaryExpr parses a primary expression.
//
//     PrimaryExpr = LiteralValue | ArrayExpr | ObjectExpr .
//
//     LiteralValue = identifier | string | number | float | bool | null |
//                    "(" Expression ")" .
//
//     ArrayExpr = "[" [ ExpressionList ] "]" .
//     ObjectExpr = "{" [ FieldList ] "}" .
func (p *parser) parsePrimaryExpr() ast.Expr {
	switch p.tok {
	case token.IDENT:
		res := &ast.IdentifierExpr{
			Name:    p.lit,
			NamePos: p.pos,
		}
		p.next()
		return res

	case token.STRING, token.NUMBER, token.FLOAT, token.BOOL, token.NULL:
		res := &ast.LiteralExpr{
			Kind:     p.tok,
			Value:    p.lit,
			ValuePos: p.pos,
		}
		p.next()
		return res

	case token.LPAREN:
		lParen, _, _ := p.expect(token.LPAREN)
		expr := p.ParseExpression()
		rParen, _, _ := p.expect(token.RPAREN)

		return &ast.ParenExpr{
			LParen: lParen,
			Inner:  expr,
			RParen: rParen,
		}

	case token.LBRACKET:
		var res ast.ArrayExpr

		res.LBracket, _, _ = p.expect(token.LBRACKET)
		if p.tok != token.RBRACKET {
			res.Elements = p.parseExpressionList(token.RBRACKET)
		}
		res.RBracket, _, _ = p.expect(token.RBRACKET)

		return &res

	case token.LCURLY:
		var res ast.ObjectExpr

		res.LCurly, _, _ = p.expect(token.LCURLY)
		if p.tok != token.RCURLY {
			res.Fields = p.parseFieldList(token.RCURLY)
		}
		res.RCurly, _, _ = p.expect(token.RCURLY)

		return &res
	}

	p.addError(fmt.Sprintf("expected expression, got %s", p.tok))
	res := &ast.LiteralExpr{Kind: token.NULL, Value: "null", ValuePos: p.pos}
	p.advanceSet(statementEnd) // Eat up the rest of the line
	return res
}

var statementEnd = map[token.Token]struct{}{
	token.TERMINATOR: {},
	token.RPAREN:     {},
	token.RCURLY:     {},
	token.RBRACKET:   {},
	token.COMMA:      {},
}

// parseExpressionList parses a list of expressions.
//
//     ExpressionList = Expression { "," Expression } [ "," ].
func (p *parser) parseExpressionList(until token.Token) []ast.Expr {
	var exprs []ast.Expr

	for p.tok != until && p.tok != token.EOF {
		exprs = append(exprs, p.ParseExpression())

		if p.tok == until {
			break
		}
		if p.tok != token.COMMA {
			p.addError("missing ',' in expression list")
		}
		p.next()
	}

	return exprs
}

// parseFieldList parses a list of fields in an object.
//
//    FieldList = Field { "," Field } [ "," ] .
func (p *parser) parseFieldList(until token.Token) []*ast.ObjectField {
	var fields []*ast.ObjectField

	for p.tok != until && p.tok != token.EOF {
		fields = append(fields, p.parseField())

		if p.tok == until {
			break
		}
		if p.tok != token.COMMA {
			p.addError("missing ',' in field list")
		}
		p.next()
	}

	return fields
}

// parseField parses a field in an object.
//
//     Field = (string | identifier) "=" Expression .
func (p *parser) parseField() *ast.ObjectField {
	if p.tok != token.STRING && p.tok != token.IDENT {
		p.addError(fmt.Sprintf("Expected field name, got %s", p.tok))
		p.advanceSet(fieldStarter)
		return nil
	}

	field := ast.ObjectField{
		Name:    p.lit,
		NamePos: p.pos,
	}
	if p.tok == token.STRING && len(field.Name) > 2 {
		// If the field name is from a string literal, we need to remove the
		// surrounding quotes from the name.
		field.Name = field.Name[1 : len(field.Name)-1]
		field.Quoted = true // Mark that it was surrounded in quotes
	}
	p.next() // Consume field name

	p.expect(token.ASSIGN)

	field.Value = p.ParseExpression()
	return &field
}

var fieldStarter = map[token.Token]struct{}{
	token.STRING: {},
	token.IDENT:  {},
}
