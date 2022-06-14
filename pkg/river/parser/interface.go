package parser

import "github.com/grafana/agent/pkg/river/ast"

// ParseFile parses an entire River configuration file. The data parameter
// should hold the file content to parse, while the filename parameter is used
// for reporting errors.
//
// If an error was encountered during parsing, the returned AST will be nil
// and err will be a token.ErrorList with all the errors encountered during
// parsing.
func ParseFile(filename string, data []byte) (f *ast.File, err error) {
	p := newParser(filename, data)

	f = p.ParseFile()

	if len(p.errors) > 0 {
		err = p.errors
		f = nil
	}
	return
}

// ParseExpression parses a single River expression from expr.
//
// If an error was encountered during parsing, the returned Expr will be nil
// and err will be a token.ErrorList with all the errors encountered during
// parsing.
func ParseExpression(expr string) (e ast.Expr, err error) {
	p := newParser("", []byte(expr))

	e = p.ParseExpression()

	if len(p.errors) > 0 {
		err = p.errors
		e = nil
	}
	return
}
