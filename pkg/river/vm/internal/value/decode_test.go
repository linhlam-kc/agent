package value_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/agent/pkg/river/vm/internal/value"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	// Declare some types to use for testing. Person2 is used as a struct
	// equivalent to Person, but with a different Go type to force casting.
	type Person struct {
		Name string `river:"name,attr"`
	}

	type Person2 struct {
		Name string `river:"name,attr"`
	}

	tt := []struct {
		input, expect interface{}
	}{
		// Non-number primitives. Non-number primitives can only ever be one Go
		// type, so they are the simplest to test.
		{string("Hello!"), string("Hello!")},
		{bool(true), bool(true)},

		// Number primitives. Number primitives have many Go types they can be
		// converted to. We do an exhaustive list of conversions below.
		{int(15), int(15)},
		{int(15), int8(15)},
		{int(15), int16(15)},
		{int(15), int32(15)},
		{int(15), int64(15)},
		{int(15), uint(15)},
		{int(15), uint8(15)},
		{int(15), uint16(15)},
		{int(15), uint32(15)},
		{int(15), uint64(15)},
		{int(15), float32(15)},
		{int(15), float64(15)},
		{int(15), string("15")},

		{int8(15), int(15)},
		{int8(15), int8(15)},
		{int8(15), int16(15)},
		{int8(15), int32(15)},
		{int8(15), int64(15)},
		{int8(15), uint(15)},
		{int8(15), uint8(15)},
		{int8(15), uint16(15)},
		{int8(15), uint32(15)},
		{int8(15), uint64(15)},
		{int8(15), float32(15)},
		{int8(15), float64(15)},
		{int8(15), string("15")},

		{int16(15), int(15)},
		{int16(15), int8(15)},
		{int16(15), int16(15)},
		{int16(15), int32(15)},
		{int16(15), int64(15)},
		{int16(15), uint(15)},
		{int16(15), uint8(15)},
		{int16(15), uint16(15)},
		{int16(15), uint32(15)},
		{int16(15), uint64(15)},
		{int16(15), float32(15)},
		{int16(15), float64(15)},
		{int16(15), string("15")},

		{int32(15), int(15)},
		{int32(15), int8(15)},
		{int32(15), int16(15)},
		{int32(15), int32(15)},
		{int32(15), int64(15)},
		{int32(15), uint(15)},
		{int32(15), uint8(15)},
		{int32(15), uint16(15)},
		{int32(15), uint32(15)},
		{int32(15), uint64(15)},
		{int32(15), float32(15)},
		{int32(15), float64(15)},
		{int32(15), string("15")},

		{int64(15), int(15)},
		{int64(15), int8(15)},
		{int64(15), int16(15)},
		{int64(15), int32(15)},
		{int64(15), int64(15)},
		{int64(15), uint(15)},
		{int64(15), uint8(15)},
		{int64(15), uint16(15)},
		{int64(15), uint32(15)},
		{int64(15), uint64(15)},
		{int64(15), float32(15)},
		{int64(15), float64(15)},
		{int64(15), string("15")},

		{uint(15), int(15)},
		{uint(15), int8(15)},
		{uint(15), int16(15)},
		{uint(15), int32(15)},
		{uint(15), int64(15)},
		{uint(15), uint(15)},
		{uint(15), uint8(15)},
		{uint(15), uint16(15)},
		{uint(15), uint32(15)},
		{uint(15), uint64(15)},
		{uint(15), float32(15)},
		{uint(15), float64(15)},
		{uint(15), string("15")},

		{uint8(15), int(15)},
		{uint8(15), int8(15)},
		{uint8(15), int16(15)},
		{uint8(15), int32(15)},
		{uint8(15), int64(15)},
		{uint8(15), uint(15)},
		{uint8(15), uint8(15)},
		{uint8(15), uint16(15)},
		{uint8(15), uint32(15)},
		{uint8(15), uint64(15)},
		{uint8(15), float32(15)},
		{uint8(15), float64(15)},
		{uint8(15), string("15")},

		{uint16(15), int(15)},
		{uint16(15), int8(15)},
		{uint16(15), int16(15)},
		{uint16(15), int32(15)},
		{uint16(15), int64(15)},
		{uint16(15), uint(15)},
		{uint16(15), uint8(15)},
		{uint16(15), uint16(15)},
		{uint16(15), uint32(15)},
		{uint16(15), uint64(15)},
		{uint16(15), float32(15)},
		{uint16(15), float64(15)},
		{uint16(15), string("15")},

		{uint32(15), int(15)},
		{uint32(15), int8(15)},
		{uint32(15), int16(15)},
		{uint32(15), int32(15)},
		{uint32(15), int64(15)},
		{uint32(15), uint(15)},
		{uint32(15), uint8(15)},
		{uint32(15), uint16(15)},
		{uint32(15), uint32(15)},
		{uint32(15), uint64(15)},
		{uint32(15), float32(15)},
		{uint32(15), float64(15)},
		{uint32(15), string("15")},

		{uint64(15), int(15)},
		{uint64(15), int8(15)},
		{uint64(15), int16(15)},
		{uint64(15), int32(15)},
		{uint64(15), int64(15)},
		{uint64(15), uint(15)},
		{uint64(15), uint8(15)},
		{uint64(15), uint16(15)},
		{uint64(15), uint32(15)},
		{uint64(15), uint64(15)},
		{uint64(15), float32(15)},
		{uint64(15), float64(15)},
		{uint64(15), string("15")},

		{string("15"), int(15)},
		{string("15"), int8(15)},
		{string("15"), int16(15)},
		{string("15"), int32(15)},
		{string("15"), int64(15)},
		{string("15"), uint(15)},
		{string("15"), uint8(15)},
		{string("15"), uint16(15)},
		{string("15"), uint32(15)},
		{string("15"), uint64(15)},
		{string("15"), float32(15)},
		{string("15"), float64(15)},
		{string("15"), string("15")},

		// Arrays
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{[]int{1, 2, 3}, [...]int{1, 2, 3}},
		{[...]int{1, 2, 3}, []int{1, 2, 3}},
		{[...]int{1, 2, 3}, [...]int{1, 2, 3}},

		// Maps
		{map[string]int{"year": 2022}, map[string]uint{"year": 2022}},
		{map[string]string{"name": "John"}, map[string]string{"name": "John"}},
		{map[string]string{"name": "John"}, Person{Name: "John"}},
		{Person{Name: "John"}, map[string]string{"name": "John"}},
		{Person{Name: "John"}, Person{Name: "John"}},
		{Person{Name: "John"}, Person2{Name: "John"}},
		{Person2{Name: "John"}, Person{Name: "John"}},

		// NOTE(rfratto): we don't test capsules or functions here because they're
		// not comparable in the same way as we do the other tests.
		//
		// See TestDecode_Functions and TestDecode_Capsules for specific decoding
		// tests of those types.
	}

	for _, tc := range tt {
		val := value.Encode(tc.input)

		name := fmt.Sprintf(
			"%s (%s) to %s",
			val.Kind(),
			reflect.TypeOf(tc.input),
			reflect.TypeOf(tc.expect),
		)

		t.Run(name, func(t *testing.T) {
			vPtr := reflect.New(reflect.TypeOf(tc.expect)).Interface()
			require.NoError(t, value.Decode(val, vPtr))

			actual := reflect.ValueOf(vPtr).Elem().Interface()

			require.Equal(t, tc.expect, actual)
		})
	}
}

func TestDecode_Functions(t *testing.T) {
	val := value.Encode(func() int { return 15 })

	var f func() int
	require.NoError(t, value.Decode(val, &f))
	require.Equal(t, 15, f())
}

func TestDecode_Capsules(t *testing.T) {
	expect := make(chan int, 5)

	var actual chan int
	require.NoError(t, value.Decode(value.Encode(expect), &actual))
	require.Equal(t, expect, actual)
}

// TestDecode_SliceCopy ensures that copies are made during decoding instead of
// setting values directly.
func TestDecode_SliceCopy(t *testing.T) {
	orig := []int{1, 2, 3}

	var res []int
	require.NoError(t, value.Decode(value.Encode(orig), &res))

	res[0] = 10
	require.Equal(t, []int{1, 2, 3}, orig, "Original slice should not have been modified")
}

// TestDecode_ArrayCopy ensures that copies are made during decoding instead of
// setting values directly.
func TestDecode_ArrayCopy(t *testing.T) {
	orig := [...]int{1, 2, 3}

	var res [3]int
	require.NoError(t, value.Decode(value.Encode(orig), &res))

	res[0] = 10
	require.Equal(t, [3]int{1, 2, 3}, orig, "Original array should not have been modified")
}

type riverEnumType bool

func (et *riverEnumType) UnmarshalRiver(f func(v interface{}) error) error {
	*et = false

	var s string
	if err := f(&s); err != nil {
		return err
	}

	switch s {
	case "accepted_value":
		*et = true
		return nil
	default:
		return fmt.Errorf("unrecognized value %q", s)
	}
}

func TestDecode_Unmarshaler(t *testing.T) {
	t.Run("valid type and value", func(t *testing.T) {
		var et riverEnumType
		require.NoError(t, value.Decode(value.String("accepted_value"), &et))
		require.Equal(t, riverEnumType(true), et)
	})

	t.Run("invalid type", func(t *testing.T) {
		var et riverEnumType
		err := value.Decode(value.Bool(true), &et)
		require.EqualError(t, err, "cannot assign bool to string")
	})

	t.Run("invalid value", func(t *testing.T) {
		var et riverEnumType
		err := value.Decode(value.String("bad_value"), &et)
		require.EqualError(t, err, `unrecognized value "bad_value"`)
	})

	t.Run("unmarshaler nested in other value", func(t *testing.T) {
		input := value.Array(
			value.String("accepted_value"),
			value.String("accepted_value"),
			value.String("accepted_value"),
		)

		var ett []riverEnumType
		require.NoError(t, value.Decode(input, &ett))
		require.Equal(t, []riverEnumType{true, true, true}, ett)
	})
}

type textEnumType bool

func (et *textEnumType) UnmarshalText(text []byte) error {
	*et = false

	switch string(text) {
	case "accepted_value":
		*et = true
		return nil
	default:
		return fmt.Errorf("unrecognized value %q", string(text))
	}
}

func TestDecode_TextUnmarshaler(t *testing.T) {
	t.Run("valid type and value", func(t *testing.T) {
		var et textEnumType
		require.NoError(t, value.Decode(value.String("accepted_value"), &et))
		require.Equal(t, textEnumType(true), et)
	})

	t.Run("invalid type", func(t *testing.T) {
		var et textEnumType
		err := value.Decode(value.Bool(true), &et)
		require.EqualError(t, err, "cannot assign bool to string")
	})

	t.Run("invalid value", func(t *testing.T) {
		var et textEnumType
		err := value.Decode(value.String("bad_value"), &et)
		require.EqualError(t, err, `unrecognized value "bad_value"`)
	})

	t.Run("unmarshaler nested in other value", func(t *testing.T) {
		input := value.Array(
			value.String("accepted_value"),
			value.String("accepted_value"),
			value.String("accepted_value"),
		)

		var ett []textEnumType
		require.NoError(t, value.Decode(input, &ett))
		require.Equal(t, []textEnumType{true, true, true}, ett)
	})
}
