package token

// Pos is a byte offset within an individual File.
type Pos int

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

// NewFile creates a new File for storing position information.
func NewFile(filename string) *File {
	return &File{
		filename: filename,
	}
}
