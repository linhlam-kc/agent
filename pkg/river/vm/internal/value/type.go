package value

import (
	"fmt"
	"reflect"
)

// Type represents a static type for a River value.
type Type struct {
	ty reflect.Type
	k  Kind
}

// Equals returns true if u == t.
func (t Type) Equals(u Type) bool {
	if t.k != u.k {
		return false
	}

	// Shortcut: if the Go types are exactly the same we don't have to do
	// anything else.
	if t.ty == u.ty {
		return true
	}

	switch t.k {
	case KindInvalid, KindNumber, KindString, KindBool:
		return true

	case KindArray, KindMap:
		return t.Elem().Equals(u.Elem())

	case KindFunction:
		// Functions are equal if they have equal arguments and the return value.
		if t.ty.NumIn() != u.ty.NumIn() {
			return false
		}
		for i := 0; i < t.ty.NumIn(); i++ {
			if !t.Arg(i).Equals(u.Arg(i)) {
				return false
			}
		}
		return t.Returns().Equals(u.Returns())

	case KindObject:
		// Objects are equal if they have the exact same set of keys.
		// TODO(rfratto): should we allow the order of keys to be different?
		if t.NumKeys() != u.NumKeys() {
			return false
		}
		for i := 0; i < t.NumKeys(); i++ {
			var (
				tKey = t.Key(i)
				uKey = u.Key(i)
			)

			if tKey.Name != uKey.Name {
				return false
			} else if !tKey.Type.Equals(uKey.Type) {
				return false
			}
		}
		return true

	case KindCapsule:
		return t.ty == u.ty

	default:
		panic(fmt.Sprintf("river/vm: Equals called with unrecognized Kind %s", t.k))
	}
}

// String returns the string representation of t.
func (t Type) String() string {
	// TODO(rfratto): give more helpful names for capsules/functions/objects
	switch t.k {
	default:
		return t.k.String()
	}
}

// Kind returns the specific kind of this type.
func (t Type) Kind() Kind { return t.k }

// Elem returns this type's element type. It panics if the type's Kind is not
// KindArray or KindMap.
//
// The Elem type of an array is the array element type. The Elem type of an map
// is the value of the map (the key is always a string).
func (t Type) Elem() Type {
	if t.k != KindArray && t.k != KindMap {
		panic("river/vm: Elem called on non-array and non-map type")
	}

	innerTy := t.ty.Elem()
	return Type{
		ty: innerTy,
		k:  kindFromType(innerTy),
	}
}

// NumArgs returns the number of input arguments for this type. It panics if
// the type's Kind is not KindFunction.
func (t Type) NumArgs() int {
	if t.k != KindFunction {
		panic("river/vm: NumArgs called on non-function type")
	}

	return t.ty.NumIn()
}

// Arg returns the type of the i'th argument for this type. It panics if the
// type's Kind is not KindFunction or if i is not in the range [0, NumArgs()).
func (t Type) Arg(i int) Type {
	if t.k != KindFunction {
		panic("river/vm: Arg called on non-function type")
	}

	args := t.ty.NumIn()
	if i < 0 || i >= args {
		panic(fmt.Sprintf("river/vm: Arg index %d out of range [0, %d)", i, args))
	}

	argTy := t.ty.In(i)
	return Type{
		ty: argTy,
		k:  kindFromType(argTy),
	}
}

// Returns reports the type of the return argument for this type. It panics if
// the type's Kind is not KindFunction.
func (t Type) Returns() Type {
	if t.k != KindFunction {
		panic("river/vm: Returns called on non-function type")
	}

	if t.ty.NumOut() != 1 {
		panic("river/vm: Function type must have exactly 1 return value")
	}

	retTy := t.ty.Out(0)
	return Type{
		ty: retTy,
		k:  kindFromType(retTy),
	}
}

// NumKeys returns the number of keys for this type. It panics if this type's
// Kind is not KindObject.
func (t Type) NumKeys() int {
	if t.k != KindObject {
		panic("river/vm: NumKeys called on non-object type")
	}

	return len(getCachedTags(t.ty))
}

// Key returns the i'th key for this type. It panics if the type's Kind is not
// KindObject or if i is not in the range [0, NumKeys()].
func (t Type) Key(i int) ObjectKeyType {
	if t.k != KindObject {
		panic("river/vm: Key called on non-object type")
	}

	keys := getCachedTags(t.ty)
	if i < 0 || i >= len(keys) {
		panic(fmt.Sprintf("river/vm: Key index %d out of range [0, %d)", i, len(keys)))
	}

	keyTy := t.ty.Field(keys[i].Index).Type

	return ObjectKeyType{
		Name: keys[i].Name,
		Type: Type{
			ty: keyTy,
			k:  kindFromType(keyTy),
		},
	}
}

// ObjectKeyType is an individual key within an object.
type ObjectKeyType struct {
	Name string
	Type Type
}
