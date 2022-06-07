package scanner

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/grafana/agent/pkg/river/token"
)

// EBNF for the scanner:
//
//   IDENT  = letter { letter | number }
//   NULL   = "null"
//   BOOL   = "true" | "false"
//   NUMBER = [ digit ]
//   FLOAT  = [ digit ] "." digit { digit } [ "e" ("+" | "-") digit { digit } ]
//   STRING = '"' { character | escape_sequence } '"'
//   OR     = "||"
//   AND    = "&&"
//   NOT    = "!"
//   NEQ    = "!="
//   ASSIGN = "="
//   EQ     = "=="
//   LT     = "<"
//   LTE    = "<="
//   GT     = ">"
//   GTE    = ">="
//   ADD    = "+"
//   SUB    = "-"
//   MUL    = "*"
//   DIV    = "/"
//   MOD    = "%"
//   POW    = "^"
//   LCURLY = "{"
//   RCURLY = "}"
//   LPAREN = "("
//   RPAREN = ")"
//   LBRACK = "["
//   RBRACK = "]"
//   COMMA  = ","
//   DOT    = "."

// ErrorHandler is invoked whenever there is an error.
type ErrorHandler func(pos token.Pos, msg string)

// Mode is a set of bitwise flags which control scanner behavior.
type Mode uint

const (
	// IncludeComments will cause comments to be returned as comment tokens.
	// Otherwise, comments are ignored.
	IncludeComments Mode = 1 << iota

	// Avoids automatic insertion of terminators (for testing only).
	dontInsertTerms
)

const (
	bom = 0xFEFF // byte order mark, permitted as very first character
	eof = -1     // end of file
)

// Scanner holds the internal state for the tokenizer while processing configs.
type Scanner struct {
	file  *token.File  // Config file handle
	input []byte       // Input config
	err   ErrorHandler // Error reporting (may be nil)
	mode  Mode

	// scanning state:

	ch         rune // Current character
	offset     int  // Position offset of ch
	readOffset int  // Position offset of first character after ch
	insertTerm bool // Insert a terminator before the next newline
	numErrors  int
}

// New creates a new scanner to tokenize the provided input config. The
// scanner uses the provided file for adding line information for each token.
// The mode parameter determines how to customize scanner behavior.
//
// Calls to Scan will invoke the error handler eh when a lexical error is found
// if eh is not nil.
func New(file *token.File, input []byte, eh ErrorHandler, mode Mode) *Scanner {
	s := &Scanner{
		file:  file,
		input: input,
		err:   eh,
		mode:  mode,
	}

	// Preload first character
	s.next()
	if s.ch == bom {
		s.next() // ignore BOM if it's the first character
	}
	return s
}

// next advances the scanner and reads the next Unicode character into s.ch.
// s.ch < 0 indicates end of file.
func (s *Scanner) next() {
	if s.readOffset >= len(s.input) {
		s.offset = len(s.input)
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		s.ch = eof
		return
	}

	s.offset = s.readOffset
	if s.ch == '\n' {
		s.file.AddLine(s.offset)
	}

	r, width := rune(s.input[s.offset]), 1
	switch {
	case r == 0:
		s.onError(s.offset, "illegal character NUL")
	case r >= utf8.RuneSelf: // not ASCII, get the unicode char
		r, width = utf8.DecodeRune(s.input[s.offset:])
		if r == utf8.RuneError && width == 1 {
			s.onError(s.offset, "illegal UTF-8 encoding")
		} else if r == bom && s.offset > 0 {
			s.onError(s.offset, "illegal byte order mark")
		}
	}
	s.readOffset += width
	s.ch = r
}

func (s *Scanner) onError(offset int, msg string) {
	if s.err != nil {
		s.err(token.Pos(offset), msg)
	}
	s.numErrors++
}

// Scan scans the next token and returns the token position, the token, and its
// literal string (if applicable). The end of the input is indicated by
// token.EOF.
//
// If the returned token is a literal (such as token.STRING), then lit contains
// the corresponding value.
//
// If the returned token is a keyword, lit is the keyword that was scanned.
//
// If the returned token is token.TERMINATOR, lit will contain "\n".
//
// If the returned token is token.ILLEGAL, lit contains the offending
// character.
//
// Otherwise, lit will be an empty string.
//
// For more tolerant parsing, Scan returns a valid token whenever possible even
// when a syntax error was encountered. Callers must check NumErrors() or the
// number of times the installed ErrorHandler was invoked to ensure there were
// no errors found during scanning.
//
// Scan will inject line information to the file provided to NewScanner.
// Returned token positions are relative to that file.
func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
scanAgain:
	s.skipWhitespace()

	// Current token start
	pos = token.Pos(s.offset)

	// Determine token value
	insertTerm := false
	switch ch := s.ch; {
	case isLetter(ch):
		lit = s.scanIdentifier()
		if len(lit) > 1 { // Identifiers are always > 1 char
			tok = token.Lookup(lit)
			switch tok {
			case token.IDENT, token.NULL, token.BOOL:
				insertTerm = true
			}
		} else {
			insertTerm = true
			tok = token.IDENT
		}

	case isDecimal(ch) || (ch == '.' && isDecimal(rune(s.peek()))):
		insertTerm = true
		tok, lit = s.scanNumber()

	default:
		s.next() // Make progress

		switch ch {
		case eof:
			if s.insertTerm {
				s.insertTerm = false // Consumed EOF
				return pos, token.TERMINATOR, "\n"
			}
			tok = token.EOF

		case '\n':
			// We can only reach here if s.insertTerm is true; skipWhitespace will
			// otherwise skip all other newlines.
			s.insertTerm = false // Consumed newline
			return pos, token.TERMINATOR, "\n"

		case '"':
			insertTerm = true
			tok = token.STRING
			lit = s.scanString()

		case '|':
			if s.ch != '|' {
				s.onError(s.offset, "missing second | in logical OR")
			} else {
				s.next() // consume second '|'
			}
			tok = token.OR
		case '&':
			if s.ch != '&' {
				s.onError(s.offset, "missing second & in logical AND")
			} else {
				s.next() // consume second '&'
			}
			tok = token.AND

		case '#': // Comment
			if s.insertTerm {
				// We're expecting a terminator. Reset the scanner back to the start of
				// the comment and force emit a terminator.
				s.ch = '#'
				s.offset = int(pos)
				s.readOffset = s.offset + 1
				s.insertTerm = false // consumed newline
				return pos, token.TERMINATOR, "\n"
			}
			comment := s.scanComment()
			if s.mode&IncludeComments == 0 {
				// Skip comment
				s.insertTerm = false // newline consumed
				goto scanAgain
			}
			tok = token.COMMENT
			lit = comment

		case '!': // !, !=
			tok = s.switch2(token.NOT, token.NEQ, '=')
		case '=': // =, ==
			tok = s.switch2(token.ASSIGN, token.EQ, '=')
		case '<': // <, <=
			tok = s.switch2(token.LT, token.LTE, '=')
		case '>': // >, >=
			tok = s.switch2(token.GT, token.GTE, '=')
		case '+':
			tok = token.ADD
		case '-':
			tok = token.SUB
		case '*':
			tok = token.MUL
		case '/':
			// NOTE(rfratto): //- and /*-style comments currently aren't included in
			// favor of only supporting #-style comments.
			tok = token.DIV

		case '%':
			tok = token.MOD
		case '^':
			tok = token.POW
		case '{':
			tok = token.LCURLY
		case '}':
			tok = token.RCURLY
		case '(':
			tok = token.LPAREN
		case ')':
			tok = token.RPAREN
		case '[':
			tok = token.LBRACKET
		case ']':
			tok = token.RBRACKET
		case ',':
			tok = token.COMMA
		case '.': // Fractions starting with '.' are handled by outer switch
			tok = token.DOT

		default:
			// s.next() reports invalid BOMs so we don't repeat the error
			if ch != bom {
				s.onError(int(pos), fmt.Sprintf("illegal character %#U", ch))
			}
			insertTerm = s.insertTerm // Preserve previous s.insertTerm
			tok = token.ILLEGAL
			lit = string(ch)
		}
	}

	if s.mode&dontInsertTerms == 0 {
		s.insertTerm = insertTerm
	}
	return
}

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\r' || (s.ch == '\n' && !s.insertTerm) {
		s.next()
	}
}

func isLetter(ch rune) bool {
	// We check for ASCII first as an optimization, and leave checking unicode
	// (the slowest) to the very last.
	return (lower(ch) >= 'a' && lower(ch) <= 'z') ||
		ch == '_' ||
		(ch >= utf8.RuneSelf && unicode.IsLetter(ch))
}

// scanIdentifier reads the string of valid identifier characters starting at
// s.offset. It must only be called when s.ch is a valid letter.
//
// scanIdentifier is highly optimized for identifiers and modifications must be
// made carefully.
func (s *Scanner) scanIdentifier() string {
	off := s.offset

	// Optimize for common case of ASCII identifiers.
	//
	// Ranging over s.input[s.readOffset:] avoids bounds checks and avoids
	// conversions to runes.
	//
	// We'll fall back to the slower path if we find a non-ASCII character.
	for readOffset, b := range s.input[s.readOffset:] {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || (b >= '0' && b <= '9') {
			// Common case: ASCII character; don't assign a rune.
			continue
		}
		s.readOffset += readOffset
		if b > 0 && b < utf8.RuneSelf {
			// Optimization: ASCII character that's not a letter or number. We avoid
			// the call to s.next() and the corresponding setup.
			//
			// This optimization only works because we know that s.ch is never '\n'.
			s.ch = rune(b)
			s.offset = s.readOffset
			s.readOffset++
			goto exit
		}

		// The preceding character is valid for an identifier because
		// scanIdentifier is only called when s.ch is a letter; calling s.next() at
		// s.readOffset will reset the scanner state.
		s.next()
		for isLetter(s.ch) || isDigit(s.ch) {
			s.next()
		}
		goto exit
	}

	s.offset = len(s.input)
	s.readOffset = len(s.input)
	s.ch = eof

exit:
	return string(s.input[off:s.offset])
}

func isDigit(ch rune) bool {
	return isDecimal(ch) || (ch >= utf8.RuneSelf && unicode.IsDigit(ch))
}

// peek gets the next byte after the current character without advancing the
// scanner. Returns 0 if the scanner is at EOF.
func (s *Scanner) peek() byte {
	if s.readOffset < len(s.input) {
		return s.input[s.readOffset]
	}
	return 0
}

// lower returns the lowercase of ch if ch is an ASCII letter.
func lower(ch rune) rune     { return ('a' - 'A') | ch }
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }

func (s *Scanner) scanNumber() (token.Token, string) {
	tok := token.NUMBER
	off := s.offset

	// integer part of number
	if s.ch != '.' {
		s.digits()
	}

	// fractional part
	if s.ch == '.' {
		tok = token.FLOAT

		s.next()
		s.digits()
	}

	// exponent
	if lower(s.ch) == 'e' {
		tok = token.FLOAT

		s.next()
		if s.ch == '+' || s.ch == '-' {
			s.next()
		}

		if s.digits() == 0 {
			s.onError(off, "exponent has no digits")
		}
	}

	return tok, string(s.input[off:s.offset])
}

// digits scans a set of digits.
func (s *Scanner) digits() (count int) {
	for isDecimal(s.ch) {
		s.next()
		count++
	}
	return
}

func (s *Scanner) scanString() string {
	// subtract 1 to account for the opening '"' which was already consumed by
	// the scanner forcing progress.
	off := s.offset - 1

	for {
		ch := s.ch
		if ch == '\n' || ch == eof {
			s.onError(off, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape()
		}
	}

	return string(s.input[off:s.offset])
}

// scanEscape parses an escape sequence. In case of a syntax error, scanEscape
// stops at the offending character without consuming it.
func (s *Scanner) scanEscape() {
	off := s.offset

	var (
		n         int
		base, max uint32
	)

	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
		s.next()
		return
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.ch == eof {
			msg = "escape sequence not terminated"
		}
		s.onError(off, msg)
		return
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch == eof {
				msg = "escape sequence not terminated"
			}
			s.onError(off, msg)
			return
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || x >= 0xD800 && x < 0xE000 {
		s.onError(off, "escape sequence is invalid Unicode code point")
	}
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= lower(ch) && lower(ch) <= 'f':
		return int(lower(ch) - 'a' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanComment() string {
	// The '#' was already consumed; s.ch is the next character in the comment
	// sequence.

	var (
		off   = s.offset - 1 // Offset of initial '#'
		numCR = 0
	)

	for s.ch != '\n' && s.ch != eof {
		if s.ch == '\r' {
			numCR++
		}
		s.next()
	}

	lit := s.input[off:s.offset]

	// On Windows, a single comment line may end in "\r\n". We want to remove the
	// final \r.
	if numCR > 0 && len(lit) >= 1 && lit[len(lit)-1] == '\r' {
		lit = lit[:len(lit)-1]
		numCR--
	}

	if numCR > 0 {
		lit = stripCR(lit)
	}

	return string(lit)
}

func stripCR(b []byte) []byte {
	c := make([]byte, len(b))
	i := 0

	for _, ch := range b {
		if ch != '\r' {
			c[i] = ch
			i++
		}
	}

	return c[:i]
}

// switch2 returns tok1 if s.ch is next, tok0 otherwise. The scanner will be
// advanced if tok1 is returned.
func (s *Scanner) switch2(tok0, tok1 token.Token, next rune) token.Token {
	if s.ch == next {
		s.next()
		return tok1
	}
	return tok0
}
