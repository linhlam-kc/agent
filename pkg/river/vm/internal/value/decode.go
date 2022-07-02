package value

import (
	"encoding"
	"fmt"
	"reflect"
)

// Unmarshaler allows types to implement custom unmarshal methods.
type Unmarshaler interface {
	// UnmarshalRiver will be called when the type is about to be decoded.
	UnmarshalRiver(unmarshal func(v interface{}) error) error
}

var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// Decode assigns a Value to a Go value. Decode will attempt to convert val to
// the type expected by target for assignment. If val cannot be converted, an
// error is returned.
//
// Decode will panic if target is not a pointer to a value.
func Decode(val Value, target interface{}) error {
	rt := reflect.ValueOf(target)
	if rt.Kind() != reflect.Pointer {
		panic("river/vm: Decode called with non-pointer")
	}
	return decode(val, rt)
}

// NOTE(rfratto): when we report errors from type mismatches in all decode
// functions below, we always say that the Go type is the expected type. This
// means if we're trying to decode into a Go string, but we have a River bool,
// the user would see they they should've provided a string instead of a bool.

func decode(val Value, rt reflect.Value) error {
	// Before decoding, we temporarily take the addr of rt so we can check to see
	// if it implements supported interfaces.
	if rt.CanAddr() {
		rt = rt.Addr()
	}

	// TODO(rfratto): is this at the right level? Does it need to be before/after
	// deferencing?
	if rt.Type().Implements(unmarshalerType) {
		return rt.Interface().(Unmarshaler).UnmarshalRiver(func(v interface{}) error {
			rt := reflect.ValueOf(v)
			if rt.Kind() != reflect.Pointer {
				return fmt.Errorf("unmarshal called with non-pointer type")
			}
			return decode(val, rt)
		})
	} else if rt.Type().Implements(textUnmarshalerType) {
		var s string
		err := decode(val, reflect.ValueOf(&s))
		if err != nil {
			return err
		}
		return rt.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
	}

	// Fully deference rt and allocate pointers as necessary.
	for rt.Kind() == reflect.Pointer {
		if rt.IsNil() {
			rt.Set(reflect.New(rt.Type().Elem()))
		}
		rt = rt.Elem()
	}

	// Fastest case: rt is directly assignable.
	switch rt.Type() {
	case val.v.Type(): // rt == val.v
		rt.Set(cloneValue(val.v))
		return nil
	case emptyInterface: // rt == interface{}
		rt.Set(cloneValue(val.v))
		return nil
	}

	// Slower cases: we'll individually examine our kinds and try to do what we
	// can.
	switch val.Kind() {
	case KindInvalid:
		panic("river/vm: Deocde called with invalid value")
	case KindNumber:
		convVal, err := convertBasicValue(val.v, rt.Type())
		if err != nil {
			return fmt.Errorf("%s expected, got number", kindFromType(rt.Type()))
		}
		rt.Set(convVal)
	case KindString:
		convVal, err := convertBasicValue(val.v, rt.Type())
		if err != nil {
			return fmt.Errorf("%s expected, got string", kindFromType(rt.Type()))
		}
		rt.Set(convVal)
	case KindBool:
		if rt.Type().Kind() != reflect.Bool {
			return fmt.Errorf("%s expected, got bool", kindFromType(rt.Type()))
		}
		rt.Set(reflect.ValueOf(val.v.Bool()))
	case KindArray:
		return decodeArray(val, rt)
	case KindObject:
		return decodeObject(val, rt)
	case KindFunction:
		// Function types must have the exact same signature, which would've been
		// handled in the best case statement above. If we've hit this point, the
		// types are incompatible.
		//
		// TODO(rfratto): this seems wrong. How will a user-defined function ever
		// exactly match the signature of a Go function?
		return fmt.Errorf("cannot assign function type %s to %s", val.v.Type(), rt.Type())
	case KindCapsule:
		// Capsule types require the Go types to be exactly the same, which
		// would've been handled in the best case statement above. If we've hit
		// this point, the types are incompatible.
		return fmt.Errorf("cannot assign type %s to %s", val.v.Type(), rt.Type())
	default:
		panic("river/vm: unexpected kind " + val.Kind().String())
	}

	return nil
}

func unwrapAny(val Value) Value {
	unwrapped := val.v.Elem()
	return Value{
		v: unwrapped,
		k: kindFromType(unwrapped.Type()),
	}
}

func decodeArray(val Value, rt reflect.Value) error {
	switch rt.Kind() {
	case reflect.Slice:
		res := reflect.MakeSlice(rt.Type(), val.v.Len(), val.v.Len())
		for i := 0; i < val.v.Len(); i++ {
			// Decode the original elements into the new elements.
			if err := decode(val.Index(i), res.Index(i)); err != nil {
				return err
			}
		}
		rt.Set(res)

	case reflect.Array:
		res := reflect.New(rt.Type()).Elem()
		for i := 0; i < val.v.Len(); i++ {
			// Stop processing elements if the target array is too short.
			if i >= res.Len() {
				break
			}
			if err := decode(val.Index(i), res.Index(i)); err != nil {
				return err
			}
		}
		rt.Set(res)

	default:
		// Special case: []byte to string
		if val.v.Type() == byteSliceType && rt.Type() == stringType {
			rt.Set(val.v.Convert(stringType))
			return nil
		}

		return fmt.Errorf("expected %s, got array", kindFromType(rt.Type()))
	}

	return nil
}

func decodeObject(val Value, rt reflect.Value) error {
	switch val.v.Kind() {
	case reflect.Struct:
		return decodeStructObject(val, rt)
	case reflect.Map:
		return decodeMapObject(val, rt)
	default:
		panic(fmt.Sprintf("river/vm: unexpected object type %s", val.v.Kind()))
	}
}

func decodeStructObject(val Value, rt reflect.Value) error {
	switch rt.Kind() {
	case reflect.Struct:
		// TODO(rfratto): can we find a way to encode optional keys that aren't
		// set?
		sourceTags := getCachedTags(val.v.Type())
		targetTags := getCachedTags(rt.Type())

		for i := 0; i < sourceTags.Len(); i++ {
			key := sourceTags.Index(i)
			keyValue, _ := val.Key(key.Name)

			// Find the equivalent key in the Go struct.
			target, ok := targetTags.Get(key.Name)
			if !ok {
				return fmt.Errorf("unsupported key %q", key.Name)
			}

			if err := decode(keyValue, rt.Field(target.Index)); err != nil {
				return err
			}
		}

	case reflect.Map:
		if rt.Type().Key() != stringType {
			// Maps with non-string types are treated as capsules and can't be
			// decoded from objects.
			return fmt.Errorf("expected %s, got object", kindFromType(rt.Type()))
		}

		res := reflect.MakeMapWithSize(rt.Type(), val.Len())

		sourceTags := getCachedTags(val.v.Type())

		for i := 0; i < sourceTags.Len(); i++ {
			keyName := sourceTags.Index(i).Name
			keyValue, _ := val.Key(keyName)

			// Create a new value to hold the entry and decode into it.
			entry := reflect.New(rt.Type().Elem()).Elem()
			if err := decode(keyValue, entry); err != nil {
				return err
			}

			// Then set the map index.
			res.SetMapIndex(reflect.ValueOf(keyName), entry)
		}
		rt.Set(res)

	default:
		return fmt.Errorf("expected %s, got map", kindFromType(rt.Type()))
	}

	return nil
}

func decodeMapObject(val Value, rt reflect.Value) error {
	switch rt.Kind() {
	case reflect.Struct:
		// TODO(rfratto): can we find a way to encode optional keys that aren't
		// set?
		targetTags := getCachedTags(rt.Type())

		// TODO(rfratto): we need to iterate over the map
		for _, key := range val.Keys() {
			// We ignore the ok value below because we know it exists in the map.
			value, _ := val.Key(key)

			// Find the equivalent key in the Go struct.
			target, ok := targetTags.Get(key)
			if !ok {
				return fmt.Errorf("unsupported key %q", key)
			}

			if err := decode(value, rt.Field(target.Index)); err != nil {
				return err
			}
		}

	case reflect.Map:
		if rt.Type().Key() != stringType {
			// Maps with non-string types are treated as capsules and can't be
			// decoded from maps.
			return fmt.Errorf("expected %s, got object", kindFromType(rt.Type()))
		}

		res := reflect.MakeMapWithSize(rt.Type(), val.Len())

		for _, key := range val.Keys() {
			// We ignore the ok value below because we know it exists in the map.
			value, _ := val.Key(key)

			// Create a new value to hold the entry and decode into it.
			entry := reflect.New(rt.Type().Elem()).Elem()
			if err := decode(value, entry); err != nil {
				return err
			}

			// Then set the map index.
			res.SetMapIndex(reflect.ValueOf(key), entry)
		}
		rt.Set(res)

	default:
		return fmt.Errorf("expected %s, got map", kindFromType(rt.Type()))
	}

	return nil
}
