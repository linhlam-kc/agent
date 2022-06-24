package value

import (
	"fmt"
	"reflect"
)

type capsuleMarker interface {
	RiverCapsule()
}

// Special types to compare against
var (
	capsuleMarkerType = reflect.TypeOf((*capsuleMarker)(nil)).Elem()
	emptyInterface    = reflect.TypeOf((*interface{})(nil)).Elem()
	stringType        = reflect.TypeOf(string(""))
)

// Kind represents a category of River value.
type Kind uint8

// Supported Kind values. The zero value, KindInvalid, isn't valid.
const (
	KindInvalid Kind = iota
	KindAny
	KindNumber
	KindString
	KindBool
	KindArray
	KindMap
	KindObject
	KindFunction
	KindCapsule
)

var kindStrings = [...]string{
	KindInvalid:  "invalid",
	KindAny:      "any",
	KindNumber:   "number",
	KindString:   "string",
	KindBool:     "bool",
	KindArray:    "array",
	KindMap:      "map",
	KindObject:   "object",
	KindFunction: "function",
	KindCapsule:  "capsule",
}

// String returns the name of k.
func (k Kind) String() string {
	if int(k) < len(kindStrings) {
		return kindStrings[k]
	}
	return fmt.Sprintf("Kind(%d)", k)
}

// GoString returns the `%#v` format of k.
func (k Kind) GoString() string { return k.String() }

// kindFromType maps a reflect.Type to a Kind.
func kindFromType(t reflect.Type) (k Kind) {
	if t.Implements(capsuleMarkerType) {
		return KindCapsule
	}

	// Deference pointers
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Invalid:
		return KindInvalid

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return KindNumber
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return KindNumber
	case reflect.Float32, reflect.Float64:
		return KindNumber

	case reflect.String:
		return KindString

	case reflect.Bool:
		return KindBool

	case reflect.Array, reflect.Slice:
		return KindArray

	case reflect.Map:
		if t.Key() != stringType {
			// Maps must be keyed by string. Anything else is forced to be a Capsule.
			return KindCapsule
		}
		return KindMap

	case reflect.Struct:
		tfs := getCachedTags(t)
		if len(tfs) == 0 {
			// Structs must have at least one river tag to be used as an object.
			return KindCapsule
		}
		return KindObject

	case reflect.Func:
		// TODO(rfratto): require argument?
		return KindFunction

	case reflect.Interface:
		if t == emptyInterface {
			return KindAny
		}
		return KindCapsule

	default:
		return KindCapsule
	}
}
