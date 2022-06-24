package value

import (
	"fmt"
	"reflect"
)

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
	// Fully deference rt and allocate pointers as necessary.
	for rt.Kind() == reflect.Pointer {
		if rt.IsNil() {
			rt.Set(reflect.New(rt.Type().Elem()))
		}
		rt = rt.Elem()
	}

	// Best case: rt is directly assignable because the underlying value of val
	// and rt match.
	if val.v.Type() == rt.Type() {
		rt.Set(cloneValue(val.v))
		return nil
	}

	// Slower cases: we'll individually examine our kinds and try to do what we
	// can.
	switch val.Kind() {
	case KindInvalid:
		panic("river/vm: Deocde called with invalid value")
	case KindAny:
		// Unwrap and try again.
		unwrapped := val.v.Elem()
		unwrappedVal := Value{
			v: unwrapped,
			k: kindFromType(unwrapped.Type()),
		}
		return decode(unwrappedVal, rt)

	case KindNumber:
		convVal, err := convertBasicValue(val.v, rt.Type())
		if err != nil {
			return fmt.Errorf("%s expected, got number", kindFromType(rt.Type()))
		}
		rt.Set(convVal)
	case KindString:
		convVal, err := convertBasicValue(val.v, rt.Type())
		if err != nil {
			return fmt.Errorf("%s expected, got number", kindFromType(rt.Type()))
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
	case KindMap:
		return decodeMap(val, rt)
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
		return fmt.Errorf("expected %s, got array", kindFromType(rt.Type()))
	}

	return nil
}

func decodeObject(val Value, rt reflect.Value) error {
	switch rt.Kind() {
	case reflect.Struct:
		// TODO(rfratto): can we find a way to encode optional keys that aren't
		// set?
		targetTags := getCachedTags(rt.Type())

		keys := val.Type().NumKeys()
		for i := 0; i < keys; i++ {
			key := val.Type().Key(i)
			keyValue := val.Key(i)

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

		res := reflect.MakeMapWithSize(rt.Type(), val.Type().NumKeys())

		keys := val.Type().NumKeys()
		for i := 0; i < keys; i++ {
			keyName := val.Type().Key(i).Name
			keyValue := val.Key(i)

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

func decodeMap(val Value, rt reflect.Value) error {
	switch rt.Kind() {
	case reflect.Struct:
		// TODO(rfratto): can we find a way to encode optional keys that aren't
		// set?
		targetTags := getCachedTags(rt.Type())

		// TODO(rfratto): we need to iterate over the map
		for _, key := range val.MapKeys() {
			// We ignore the ok value below because we know it exists in the map.
			value, _ := val.MapIndex(key)

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

		for _, key := range val.MapKeys() {
			// We ignore the ok value below because we know it exists in the map.
			value, _ := val.MapIndex(key)

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
