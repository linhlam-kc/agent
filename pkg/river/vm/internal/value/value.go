package value

import (
	"fmt"
	"reflect"
	"strconv"
)

// Value represents a River value.
type Value struct {
	v reflect.Value
	k Kind
}

// Null is the null value. It is not valid for using in expressions.
var Null = Value{}

// Uint returns a Value from a uint64.
func Uint(u uint64) Value { return Value{v: reflect.ValueOf(u), k: KindNumber} }

// Int returns a Value from an int64.
func Int(i int64) Value { return Value{v: reflect.ValueOf(i), k: KindNumber} }

// Float returns a Value from a floating-point number.
func Float(f float64) Value { return Value{v: reflect.ValueOf(f), k: KindNumber} }

// String returns a Value from a string.
func String(s string) Value { return Value{v: reflect.ValueOf(s), k: KindString} }

// Bool returns a Value from a bool.
func Bool(b bool) Value { return Value{v: reflect.ValueOf(b), k: KindBool} }

// Object returns a new Value from m. A copy of m is made for producing the
// Value.
func Object(m map[string]Value) Value {
	raw := reflect.MakeMapWithSize(reflect.TypeOf(map[string]interface{}(nil)), len(m))

	for k, v := range m {
		raw.SetMapIndex(reflect.ValueOf(k), v.v)
	}

	return Value{v: raw, k: KindObject}
}

// Array creates an array from the given values.
func Array(vv ...Value) Value {
	if len(vv) == 0 {
		return Encode([]interface{}(nil))
	}

	arrayType := reflect.SliceOf(emptyInterface)
	raw := reflect.MakeSlice(arrayType, len(vv), len(vv))

	for i, v := range vv {
		raw.Index(i).Set(v.v)
	}

	return Value{v: raw, k: KindArray}
}

// Func makes a new function Value from f. f must be a function with exactly
// one return argument.
func Func(f interface{}) Value {
	rf := reflect.ValueOf(f)
	if rf.Type().Kind() != reflect.Func {
		panic("river/vm: Func called with non-function type")
	}

	if rf.Type().NumOut() != 1 {
		panic(fmt.Sprintf("river/vm: Func called with function that has %d output arguments, but exactly 1 is required", rf.Type().NumOut()))
	}

	return Value{v: rf, k: KindFunction}
}

// Capsule creates a new Capsule value from v.
func Capsule(v interface{}) Value {
	return Value{v: reflect.ValueOf(v), k: KindCapsule}
}

// Kind returns the Kind of the value.
func (v Value) Kind() Kind { return v.k }

// Len returns the length of v. Panics if the Kind of Value is not KindArray or
// KindObject.
func (v Value) Len() int {
	switch v.k {
	case KindArray:
		return v.v.Len()
	case KindObject:
		switch v.v.Kind() {
		case reflect.Struct:
			return getCachedTags(v.v.Type()).Len()
		case reflect.Map:
			return v.v.Len()
		default:
			panic("river/vm: unexpected object value " + v.v.Kind().String())
		}
	default:
		panic("river/vm: Len called on non-array and non-object value")
	}
}

// Keys returns the keys in v, in unspecified order. It panics if the
// Kind of v is not KindObject.
func (v Value) Keys() []string {
	if v.k != KindObject {
		panic("river/vm: MapKeys called on non-object value")
	}

	switch v.v.Kind() {
	case reflect.Struct:
		ff := getCachedTags(v.v.Type())
		return ff.Keys()

	case reflect.Map:
		// TODO(rfratto): optimize?
		reflectKeys := v.v.MapKeys()
		res := make([]string, len(reflectKeys))
		for i, rk := range reflectKeys {
			res[i] = rk.String()
		}
		return res

	default:
		panic("river/vm: unexpected object value " + v.v.Kind().String())
	}
}

// Key returns the value for a key in v. It panics if the Kind of v is not
// KindObject. ok will be false if the key did not exist in the object.
func (v Value) Key(key string) (index Value, ok bool) {
	if v.k != KindObject {
		panic("river/vm: MapIndex called on non-object value")
	}

	switch v.v.Kind() {
	case reflect.Struct:
		// TODO(rfratto): optimize
		ff := getCachedTags(v.v.Type())
		f, foundField := ff.Get(key)
		if !foundField {
			return
		}
		return makeValue(v.v.Field(f.Index)), true

	case reflect.Map:
		val := v.v.MapIndex(reflect.ValueOf(key))
		if !val.IsValid() || val.IsZero() {
			return
		}
		return makeValue(val), true

	default:
		panic("river/vm: unexpected object value " + v.v.Kind().String())
	}
}

// makeValue converts a reflect value into a Value. makeValue will unwrap Any
// values into their concrete form.
func makeValue(v reflect.Value) Value {
	for v.Kind() == reflect.Pointer || v.Type() == emptyInterface {
		v = v.Elem()
	}
	return Value{v: v, k: kindFromType(v.Type())}
}

// Index returns index i of the Value. Panics if the Kind of Value is not
// KindArray.
func (v Value) Index(i int) Value {
	if v.k != KindArray {
		panic("river/vm: Index called on non-array value")
	}
	return makeValue(v.v.Index(i))
}

// Int returns an int value for v. It panics if v is not a number.
func (v Value) Int() int64 {
	if v.k != KindNumber {
		panic("river/vm: Int called on non-number type")
	}
	switch makeNumberKind(v.v.Kind()) {
	case numberKindInt:
		return v.v.Int()
	case numberKindUint:
		return int64(v.v.Uint())
	case numberKindFloat:
		return int64(v.v.Float())
	default:
		panic("unrecognized number kind")
	}
}

// Uint returns an uint value for v. It panics if v is not a number.
func (v Value) Uint() uint64 {
	if v.k != KindNumber {
		panic("river/vm: Uint called on non-number type")
	}
	switch makeNumberKind(v.v.Kind()) {
	case numberKindInt:
		return uint64(v.v.Int())
	case numberKindUint:
		return v.v.Uint()
	case numberKindFloat:
		return uint64(v.v.Float())
	default:
		panic("unrecognized number kind")
	}
}

// Float returns a float value for v. It panics if v is not a number.
func (v Value) Float() float64 {
	if v.k != KindNumber {
		panic("river/vm: Uint called on non-number type")
	}
	switch makeNumberKind(v.v.Kind()) {
	case numberKindInt:
		return float64(v.v.Int())
	case numberKindUint:
		return float64(v.v.Uint())
	case numberKindFloat:
		return v.v.Float()
	default:
		panic("unrecognized number kind")
	}
}

// Call calls a function value with the provided arguments. It panics if v is
// not a function.
func (v Value) Call(args ...Value) (Value, error) {
	if v.k != KindFunction {
		panic("river/vm: Call called on non-function type")
	}

	var (
		variadic     = v.v.Type().IsVariadic()
		expectedArgs = v.v.Type().NumIn()
	)

	if variadic && len(args) < expectedArgs-1 {
		return Null, fmt.Errorf("expected %d args, got %d", expectedArgs-1, len(args))
	} else if !variadic && len(args) != expectedArgs {
		return Null, fmt.Errorf("expected %d args, got %d", expectedArgs, len(args))
	}

	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		var argVal reflect.Value
		if variadic && i >= expectedArgs-1 {
			argType := v.v.Type().In(expectedArgs - 1).Elem()
			argVal = reflect.New(argType).Elem()
		} else {
			argType := v.v.Type().In(i)
			argVal = reflect.New(argType).Elem()
		}

		if err := decode(arg, argVal); err != nil {
			return Null, err
		}

		reflectArgs[i] = argVal
	}

	outs := v.v.Call(reflectArgs)
	if len(outs) != 1 {
		panic("river/vm: functions without 1 return value are unsupported")
	}

	return makeValue(outs[0]), nil
}

// fitNumberTypes determines which type can be used for operations on values.
// The precedence order is: float64, float32, int64, int32, int, int16, int8,
// uint64, uint32, uint, uint16, uint8.
//
// All other kinds are given no predence. This means if fitNumberTypes is
// called with a string and a number, the number type will always be returned.
func fitNumberTypes(a, b reflect.Type) reflect.Type {
	aPrec, bPrec := kindPrec[a.Kind()], kindPrec[b.Kind()]
	if aPrec > bPrec {
		return a
	}
	return b
}

// kindPrec is a mapping of fitNumberTypes precedence from lowest to highest
var kindPrec = map[reflect.Kind]int{
	reflect.Uint8:   0,
	reflect.Uint16:  1,
	reflect.Uint:    2,
	reflect.Uint32:  3,
	reflect.Uint64:  4,
	reflect.Int8:    5,
	reflect.Int16:   6,
	reflect.Int:     7,
	reflect.Int32:   8,
	reflect.Int64:   9,
	reflect.Float32: 10,
	reflect.Float64: 11,
}

func convertBasicValue(v reflect.Value, target reflect.Type) (reflect.Value, error) {
	fromKind, toKind := v.Type().Kind(), target.Kind()

	if v.Type() == target {
		return v, nil
	} else if target == emptyInterface {
		return v, nil
	}

	switch fromKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number := v.Int()
		switch toKind {
		case reflect.Int:
			return reflect.ValueOf(int(number)), nil
		case reflect.Int8:
			return reflect.ValueOf(int8(number)), nil
		case reflect.Int16:
			return reflect.ValueOf(int16(number)), nil
		case reflect.Int32:
			return reflect.ValueOf(int32(number)), nil
		case reflect.Int64:
			return reflect.ValueOf(number), nil
		case reflect.Uint:
			return reflect.ValueOf(uint(number)), nil
		case reflect.Uint8:
			return reflect.ValueOf(uint8(number)), nil
		case reflect.Uint16:
			return reflect.ValueOf(uint16(number)), nil
		case reflect.Uint32:
			return reflect.ValueOf(uint32(number)), nil
		case reflect.Uint64:
			return reflect.ValueOf(uint64(number)), nil
		case reflect.Float32:
			return reflect.ValueOf(float32(number)), nil
		case reflect.Float64:
			return reflect.ValueOf(float64(number)), nil
		case reflect.String:
			return reflect.ValueOf(strconv.FormatInt(number, 10)), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number := v.Uint()
		switch toKind {
		case reflect.Int:
			return reflect.ValueOf(int(number)), nil
		case reflect.Int8:
			return reflect.ValueOf(int8(number)), nil
		case reflect.Int16:
			return reflect.ValueOf(int16(number)), nil
		case reflect.Int32:
			return reflect.ValueOf(int32(number)), nil
		case reflect.Int64:
			return reflect.ValueOf(int64(number)), nil
		case reflect.Uint:
			return reflect.ValueOf(uint(number)), nil
		case reflect.Uint8:
			return reflect.ValueOf(uint8(number)), nil
		case reflect.Uint16:
			return reflect.ValueOf(uint16(number)), nil
		case reflect.Uint32:
			return reflect.ValueOf(uint32(number)), nil
		case reflect.Uint64:
			return reflect.ValueOf(number), nil
		case reflect.Float32:
			return reflect.ValueOf(float32(number)), nil
		case reflect.Float64:
			return reflect.ValueOf(float64(number)), nil
		case reflect.String:
			return reflect.ValueOf(strconv.FormatUint(number, 10)), nil
		}

	case reflect.Float32, reflect.Float64:
		number := v.Float()
		switch toKind {
		case reflect.Int:
			return reflect.ValueOf(int(number)), nil
		case reflect.Int8:
			return reflect.ValueOf(int8(number)), nil
		case reflect.Int16:
			return reflect.ValueOf(int16(number)), nil
		case reflect.Int32:
			return reflect.ValueOf(int32(number)), nil
		case reflect.Int64:
			return reflect.ValueOf(int64(number)), nil
		case reflect.Uint:
			return reflect.ValueOf(uint(number)), nil
		case reflect.Uint8:
			return reflect.ValueOf(uint8(number)), nil
		case reflect.Uint16:
			return reflect.ValueOf(uint16(number)), nil
		case reflect.Uint32:
			return reflect.ValueOf(uint32(number)), nil
		case reflect.Uint64:
			return reflect.ValueOf(uint64(number)), nil
		case reflect.Float32:
			return reflect.ValueOf(float32(number)), nil
		case reflect.Float64:
			return reflect.ValueOf(number), nil
		case reflect.String:
			return reflect.ValueOf(strconv.FormatFloat(number, 'f', -1, 64)), nil
		}

	case reflect.String:
		text := v.String()

		var (
			err error

			signed   int64
			unsigned uint64
			float    float64
		)
		switch toKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			signed, err = strconv.ParseInt(text, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert %s to number: %w", text, err)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			unsigned, err = strconv.ParseUint(text, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert %s to number: %w", text, err)
			}
		case reflect.Float32, reflect.Float64:
			float, err = strconv.ParseFloat(text, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert %s to number: %w", text, err)
			}
		}

		switch toKind {
		case reflect.Int:
			return reflect.ValueOf(int(signed)), nil
		case reflect.Int8:
			return reflect.ValueOf(int8(signed)), nil
		case reflect.Int16:
			return reflect.ValueOf(int16(signed)), nil
		case reflect.Int32:
			return reflect.ValueOf(int32(signed)), nil
		case reflect.Int64:
			return reflect.ValueOf(signed), nil
		case reflect.Uint:
			return reflect.ValueOf(uint(unsigned)), nil
		case reflect.Uint8:
			return reflect.ValueOf(uint8(unsigned)), nil
		case reflect.Uint16:
			return reflect.ValueOf(uint16(unsigned)), nil
		case reflect.Uint32:
			return reflect.ValueOf(uint32(unsigned)), nil
		case reflect.Uint64:
			return reflect.ValueOf(unsigned), nil
		case reflect.Float32:
			return reflect.ValueOf(float32(float)), nil
		case reflect.Float64:
			return reflect.ValueOf(float), nil
		case reflect.String:
			return reflect.ValueOf(text), nil
		}
	}

	if v.CanConvert(target) {
		return v.Convert(target), nil
	}
	return reflect.Value{}, fmt.Errorf("expected %s, got %s", kindFromType(target), kindFromType(v.Type()))
}
