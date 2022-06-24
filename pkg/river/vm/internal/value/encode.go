package value

import (
	"reflect"
)

// Encode creates a new Value from v. Deep copies are made of arrays, slices,
// and maps, but not to pointers of those types.
func Encode(v interface{}) Value {
	rv := reflect.ValueOf(v)
	rv = cloneValue(rv) // Clone prior to deferencing to allow pointers to clonable values to persist.

	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	return makeValue(rv)
}

func cloneValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Array:
		return cloneArray(v)
	case reflect.Slice:
		return cloneSlice(v)
	case reflect.Map:
		return cloneMap(v)
	}

	return v
}

func cloneArray(in reflect.Value) reflect.Value {
	res := reflect.New(in.Type()).Elem()
	for i := 0; i < in.Len(); i++ {
		res.Index(i).Set(cloneValue(in.Index(i)))
	}
	return res
}

func cloneSlice(in reflect.Value) reflect.Value {
	res := reflect.MakeSlice(in.Type(), in.Len(), in.Len())
	for i := 0; i < in.Len(); i++ {
		res.Index(i).Set(cloneValue(in.Index(i)))
	}
	return res
}

func cloneMap(in reflect.Value) reflect.Value {
	res := reflect.MakeMapWithSize(in.Type(), in.Len())
	iter := in.MapRange()
	for iter.Next() {
		res.SetMapIndex(cloneValue(iter.Key()), cloneValue(iter.Value()))
	}
	return res
}
