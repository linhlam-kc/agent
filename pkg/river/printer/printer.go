package printer

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/grafana/agent/pkg/river/ast"
	"github.com/grafana/agent/pkg/river/token"
)

// Config configures behavior of the printer.
type Config struct {
	Indent int // Indentation to apply to all emitted code. Default 0.
}

// Print pretty-prints the specified node to w. The Node type must be an
// *ast.File, []ast.Stmt, or a type that implements ast.Stmt or ast.Expr.
func Print(w io.Writer, node interface{}) (err error) {
	var p printer
	p.Init(&Config{})

	if err = (&walker{p: &p}).Walk(node); err != nil {
		return
	}

	w = tabwriter.NewWriter(w, 0, 8, 1, ' ', tabwriter.DiscardEmptyColumns|tabwriter.TabIndent|tabwriter.StripEscape)

	if _, err = w.Write(p.output); err != nil {
		return
	}
	if tw, _ := w.(*tabwriter.Writer); tw != nil {
		// Flush tabwriter if defined
		err = tw.Flush()
	}

	return
}

// The printer writes lexical tokens and whitespace to an internal buffer.
//
// Internally, printer depends on a tabwriter for formatting text and aligning
// runs of characters. Horizontal '\t' and vertical '\v' tab characters are
// used to introduce new columns in the row. Runs of characters are stopped by
// either introducing a linefeed '\f' or by having a line with a different
// number of tab stops from the previous line. See the text/tabwriter package
// for more information on the algorithm it uses for formatting text.
type printer struct {
	cfg Config

	// State variables

	output      []byte
	indent      int          // Current indentation level
	whitespace  []whitespace // Delayed whitespace to print
	impliedTerm bool         // When set, a linebreak implies a terminator
	lastTok     token.Token  // Last token printed (token.ILLEGAL if it's whitespace)

	pos  token.Position // Current position in AST space
	out  token.Position // Current position in output space
	last token.Position // Value of pos after calling writeString
}

func (p *printer) Init(cfg *Config) {
	p.cfg = *cfg
	p.pos = token.Position{Line: 1, Column: 1}
	p.out = token.Position{Line: 1, Column: 1}
	p.whitespace = make([]whitespace, 0, 16) // whitespace sequences are short
}

// Write writes a list of writable arguments to the printer.
//
// Whitespace is accumulated until the next non-whitespace token appears.
// Comments which need to appear before that token are printed first,
// accounting for whitespace for comment placement. After comments, any
// leftover whitespace is printed, followed by the actual token.
func (p *printer) Write(args ...interface{}) {
	for _, arg := range args {
		var (
			data        string
			isLit       bool
			impliedTerm bool
		)

		switch x := arg.(type) {
		case whitespace:
			if x == wsIgnore {
				continue
			}
			i := len(p.whitespace)
			if i == cap(p.whitespace) {
				p.writeWritespace(i)
				i = 0
			}
			p.whitespace = p.whitespace[0 : i+1]
			p.whitespace[i] = x
			if x == wsNewline || x == wsFormfeed {
				// Newlines affect the current state (p.impliedTerm) and not the state
				// after printing arg (impliedTerm) because comments can be interspered
				// before the arg in this case.
				p.impliedTerm = false // TODO(rfratto): is this necessary?
			}
			p.lastTok = token.ILLEGAL
			continue

		case *ast.IdentifierExpr:
			data = x.Name
			impliedTerm = true
			p.lastTok = token.IDENT

		case *ast.LiteralExpr:
			data = x.Value
			isLit = true
			impliedTerm = true
			p.lastTok = x.Kind

		case token.Token:
			s := x.String()
			if mayCombine(p.lastTok, s[0]) {
				if len(p.whitespace) != 0 {
					panic("whitespace buffer not empty")
				}
				p.whitespace = p.whitespace[0:1]
				p.whitespace[0] = ' '
			}
			data = s

			// Some keywords followed by a newline imply a terminator.
			switch x {
			case token.RBRACKET, token.RPAREN, token.RCURLY:
				impliedTerm = true
			}
			p.lastTok = x

		case token.Pos:
			// TODO(rfratto): do we have a need for writing token.Pos? How does Go
			// use it?
			panic(fmt.Sprintf("printer: unsupported argument %v (%T)\n", arg, arg))

		default:
			panic(fmt.Sprintf("printer: unsupported argument %v (%T)\n", arg, arg))
		}

		next := p.pos // Estimated/accurate position of next item
		wroteNewline, dropedFF := p.flush(next, p.lastTok)

		// Intersperse extra newlines if present in the source and if they don't
		// cause extra semicolons. This should NOT be done in flush as it will
		// cause extra newlines at the end of a file.
		if !p.impliedTerm {
			n := nlimit(next.Line - p.pos.Line)
			// Don't exceed maxNewlines if we already wrote one.
			if wroteNewline && n == maxNewlines {
				n = maxNewlines - 1
			}
			if n > 0 {
				ch := byte('\n')
				if dropedFF {
					ch = '\f' // Use formfeed since we dropped one before
				}
				p.writeByte(ch, n)
				impliedTerm = false
			}
		}

		// TODO(rfratto): record line number just written? (go/printer does it)
		p.writeString(next, data, isLit)
		p.impliedTerm = impliedTerm
	}
}

// mayCombine returns true if two tokes must not be combined, because combining
// them would format in a different token sequence being generated.
func mayCombine(prev token.Token, next byte) (b bool) {
	switch prev {
	case token.NUMBER:
		return next == '.' // 1.
	case token.DIV:
		return next == '*' // /*
	default:
		return false
	}
}

// flush prints any pending comments and whitespace occurring textually before
// the position of the next token tok. The flush result indicates if a newline
// was written or if a formfeed \f character was dropped from the whitespace
// buffer.
func (p *printer) flush(next token.Position, tok token.Token) (wroteNewline, droppedFF bool) {
	// TODO(rfratto): check if we need to inject comments before next.
	p.writeWritespace(len(p.whitespace)) // Write all remaining whitespace.
	return
}

// nlimit limits n to maxNewlines.
func nlimit(n int) int {
	if n > maxNewlines {
		n = maxNewlines
	}
	return n
}

const maxNewlines = 2 // Maximum number of newlines between text blocks

// writeString writes the literal string s into the printer's output.
// Formatting characters in s such as '\t' and '\n' will be interpreted by
// underlying tabwriter unless isLit is set.
func (p *printer) writeString(pos token.Position, s string, isLit bool) {
	if p.out.Column == 1 {
		// We haven't written any text to this line yet; prepend our indentation
		// for the line.
		p.writeIndent()
	}

	if pos.IsValid() {
		// Update p.pos if pos is valid. This is done *after* handling indentation
		// since we want to interpret pos as the literal position for s (and
		// writeIndent will update p.pos).
		p.pos = pos
	}

	if isLit {
		// Wrap our literal string in tabwriter.Escape if it's meant to be written
		// without interpretation by the tabwriter.
		p.output = append(p.output, tabwriter.Escape)

		defer func() {
			p.output = append(p.output, tabwriter.Escape)
		}()
	}

	p.output = append(p.output, s...)

	for i := 0; i < len(s); i++ {
		if ch := s[i]; ch == '\n' || ch == '\f' {
			// TODO(rfratto): In the future, elements may cross line barriers (e.g.,
			// heredoc strings), which will require careful handling with how p.pos
			// gets updated and whether it's safe to inject whitespace in writeByte
			// while processing a multiline element.
			panic("printer: handling for elements which cross line barriers is not yet implemented")
		}
	}
	p.pos.Offset += len(s)
	p.pos.Column += len(s)
	p.out.Column += len(s)

	p.last = pos
}

func (p *printer) writeIndent() {
	depth := p.cfg.Indent + p.indent
	for i := 0; i < depth; i++ {
		p.output = append(p.output, '\t')
	}

	p.pos.Offset += depth
	p.pos.Column += depth
	p.out.Column += depth
}

// writeByte writes ch n times to the output, updating the position of the
// printer. writeByte is only used for writing whitespace characters.
func (p *printer) writeByte(ch byte, n int) {
	if p.out.Column == 1 {
		p.writeIndent()
	}

	for i := 0; i < n; i++ {
		p.output = append(p.output, ch)
	}

	// Update positions.
	p.pos.Offset += n
	if ch == '\n' || ch == '\f' {
		p.pos.Line += n
		p.out.Line += n
		p.pos.Column = 1
		p.out.Column = 1
		return
	}
	p.pos.Column += n
	p.out.Column += n
}

// writeWhitespace writes the first n whitespace entries in the whitespace
// buffer.
//
// writeWritespace is only safe to be called when len(p.whitespace) >= n.
func (p *printer) writeWritespace(n int) {
	for i := 0; i < n; i++ {
		switch ch := p.whitespace[i]; ch {
		case wsIgnore: // no-op
		case wsIndent:
			p.indent++
		case wsUnindent:
			p.indent--
			if p.indent < 0 {
				panic("printer: negative indentation")
			}
		default:
			p.writeByte(byte(ch), 1)
		}
	}

	// Shift remaining entries down
	l := copy(p.whitespace, p.whitespace[n:])
	p.whitespace = p.whitespace[:l]
}

type whitespace byte

const (
	wsIgnore   = whitespace(0)
	wsBlank    = whitespace(' ')
	wsVTab     = whitespace('\v')
	wsNewline  = whitespace('\n')
	wsFormfeed = whitespace('\f')
	wsIndent   = whitespace('>')
	wsUnindent = whitespace('<')
)
