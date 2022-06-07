package token

// Token is an individual lexical token.
type Token int

// List of all tokens.
const (
	ILLEGAL Token = iota
	EOF
	COMMENT

	IDENT  // foobar
	NUMBER // 1234
	FLOAT  // 1234.0
	STRING // "foobar"
	BOOL   // true
	NULL   // null

	OR  // ||
	AND // &&
	NOT // !

	ASSIGN // =
	EQ     // ==
	NEQ    // !=
	LT     // <
	LTE    // <=
	GT     // >
	GTE    // >=

	ADD // +
	SUB // -
	MUL // *
	DIV // /
	MOD // %
	POW // ^

	LCURLY   // {
	RCURLY   // }
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]
	COMMA    // ,
	DOT      // .

	TERMINATOR // \n
)

var tokenNames = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:  "IDENT",
	NUMBER: "NUMBER",
	STRING: "STRING",
	BOOL:   "BOOL",
	NULL:   "NULL",

	OR:  "||",
	AND: "&&",
	NOT: "!",

	ASSIGN: "=",
	EQ:     "==",
	NEQ:    "!=",
	LT:     "<",
	LTE:    "<=",
	GT:     ">",
	GTE:    ">=",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	DIV: "/",
	MOD: "%",
	POW: "^",

	LCURLY:   "{",
	RCURLY:   "}",
	LPAREN:   "(",
	RPAREN:   ")",
	LBRACKET: "[",
	RBRACKET: "]",
	COMMA:    ",",
	DOT:      ".",

	TERMINATOR: "TERMINATOR",
}

// Lookup maps a string to its keyword token or IDENT if it's not a keyword.
func Lookup(ident string) Token {
	switch ident {
	case "true", "false":
		return BOOL
	case "null":
		return NULL
	}
	return IDENT
}

// String returns the string representation corresponding to the token.
func (t Token) String() string {
	if int(t) >= len(tokenNames) {
		return "ILLEGAL"
	}

	name := tokenNames[t]
	if name == "" {
		return "ILLEGAL"
	}
	return name
}

// GoString returns the %#v format of t.
func (t Token) GoString() string { return t.String() }
