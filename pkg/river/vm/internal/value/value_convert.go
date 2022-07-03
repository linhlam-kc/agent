package value

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// convertValue converts a Value to a Value of a different Kind. The only valid
// conversions between kinds are between numbers and strings.
func convertValue(val Value, toKind Kind) (Value, error) {
	fromKind := val.Kind()

	if fromKind == toKind {
		// no-op: val is already the right kind.
		return val, nil
	}

	switch fromKind {
	case KindNumber:
		switch toKind {
		case KindString: // number -> string
			strVal := newNumberValue(val.v).ToString()
			return makeValue(reflect.ValueOf(strVal)), nil
		}

	case KindString:
		sourceStr := val.v.String()

		switch toKind {
		case KindNumber: // string -> number
			switch {
			case sourceStr == "":
				return Null, fmt.Errorf("cannot convert string %q to number", sourceStr)

			case sourceStr[0] == '-':
				// String starts with a -; parse as a signed int.
				val, err := strconv.ParseInt(sourceStr, 10, 64)
				if err != nil {
					return Null, fmt.Errorf("cannot convert string %q to number: %w", sourceStr, err)
				}
				return Int(val), nil
			case strings.ContainsAny(sourceStr, ".eE"):
				// String contains something that a floating-point number would use;
				// convert.
				val, err := strconv.ParseFloat(sourceStr, 64)
				if err != nil {
					return Null, fmt.Errorf("cannot convert string %q to number: %w", sourceStr, err)
				}
				return Float(val), nil
			default:
				// Otherwise, treat the number as an unsigned int.
				val, err := strconv.ParseUint(sourceStr, 10, 64)
				if err != nil {
					return Null, fmt.Errorf("cannot convert string %q to number: %w", sourceStr, err)
				}
				return Uint(val), nil
			}
		}
	}

	return Null, fmt.Errorf("cannot assign %s to %s", fromKind, toKind)
}

func convertNumber(v reflect.Value, target reflect.Type) reflect.Value {
	nval := newNumberValue(v)

	switch target.Kind() {
	case reflect.Int:
		return reflect.ValueOf(int(nval.Int()))
	case reflect.Int8:
		return reflect.ValueOf(int8(nval.Int()))
	case reflect.Int16:
		return reflect.ValueOf(int16(nval.Int()))
	case reflect.Int32:
		return reflect.ValueOf(int32(nval.Int()))
	case reflect.Int64:
		return reflect.ValueOf(nval.Int())
	case reflect.Uint:
		return reflect.ValueOf(uint(nval.Uint()))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(nval.Uint()))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(nval.Uint()))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(nval.Uint()))
	case reflect.Uint64:
		return reflect.ValueOf(nval.Uint())
	case reflect.Float32:
		return reflect.ValueOf(float32(nval.Float()))
	case reflect.Float64:
		return reflect.ValueOf(nval.Float())
	}

	panic("unsupported number conversion")
}
