package token

import "fmt"

// Pos is a byte offset within an individual File.
type Pos int

// Position holds full position information for a location within an individual
// file.
type Position struct {
	Filename string // Filename (if any)
	Offset   int    // Byte offset (starting at 0)
	Line     int    // Line number (starting at 1)
	Column   int    // Offset from start of line (starting at 1)
}

// IsValid reports whether the position is valid. Valid positions must have a
// Line value of at least 1.
func (pos *Position) IsValid() bool {
	return pos.Line > 1
}

// String returns a string in one of the following forms:
//
//     file:line:column   Valid position with file name
//     file:line          Valid position with file name but no column
//     line:column        Valid position with no file name
//     line               Valid position with no file name or column
//     file               Invalid position with file name
//     -                  Invalid position with no file name
func (pos Position) String() string {
	s := pos.Filename

	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d", pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}

	if s == "" {
		s = "-"
	}
	return s
}

// File holds position information for a specific file.
type File struct {
	filename string
}

// Name returns the name of the file.
func (f *File) Name() string { return f.filename }

// AddLine tracks a new line from a byte offset.
func (f *File) AddLine(offset int) {
	// TODO(rfratto): impl
}

// PositionFor returns a Position from an offset.
func (f *File) PositionFor(p Pos) Position {
	// TODO(rfratto): impl
	return Position{
		Filename: f.filename,
		Offset:   int(p),
	}
}

// NewFile creates a new File for storing position information.
func NewFile(filename string) *File {
	return &File{
		filename: filename,
	}
}
