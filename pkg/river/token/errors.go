package token

import (
	"fmt"
)

// Error is a reusable error for errors encountered during scanning, parsing,
// or evaluation.
type Error struct {
	Position Position // Starting position of the error
	Message  string
}

// Error implements error.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Position, e.Message)
}

// ErrorList is a list of Error.
type ErrorList []*Error

// Add adds a new Error into the ErrorList.
func (l *ErrorList) Add(e *Error) { *l = append(*l, e) }

// Error implements error.
func (l ErrorList) Error() string {
	switch len(l) {
	case 0:
		return "no errors"
	case 1:
		return l[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", l[0], len(l)-1)
}
