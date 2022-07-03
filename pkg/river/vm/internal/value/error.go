package value

import "fmt"

// TypeError is used for reporting on a value having an unexpected type.
type TypeError struct {
	// Value which caused the error.
	Value    Value
	Expected Kind
}

// Error returns the string form of the TypeError.
func (te TypeError) Error() string {
	return fmt.Sprintf("expected %s, got %s", te.Expected, te.Value.Kind())
}

// MissingKeyError is used for reporting that a value is missing a key.
type MissingKeyError struct {
	Value   Value
	Missing string
}

// Error returns the string form of the MissingKeyError.
func (mke MissingKeyError) Error() string {
	return fmt.Sprintf("key %q does not exist", mke.Missing)
}

// ElementError is used to report on an error inside of an array.
type ElementError struct {
	Value Value // The Array value
	Index int   // The index of the element with the issue
	Inner error // The error from the element
}

// Error returns the string form of the ElementError.
func (ee ElementError) Error() string {
	return fmt.Sprintf("index %d: %s", ee.Index, ee.Inner)
}

// FieldError is used to report on an invalid field inside an object.
type FieldError struct {
	Value Value  // The Object value
	Field string // The field name with the issue
	Inner error  // The error from the field
}

// Error returns the string form of the ElementError.
func (fe FieldError) Error() string {
	return fmt.Sprintf("field %s: %s", fe.Field, fe.Inner)
}
