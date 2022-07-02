package value

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkValue(b *testing.B) {
	type Person struct {
		Name     string `river:"name,attr"`
		Location string `river:"location,attr,optional"`
	}

	p := Person{Name: "John Doe"}

	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Encode(p)
		}
	})
}

func Benchmark_numberConversion(b *testing.B) {
	tt := []struct {
		input, expect interface{}
	}{
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
	}

	for _, tc := range tt {
		name := fmt.Sprintf(
			"%s to %s",
			reflect.TypeOf(tc.input),
			reflect.TypeOf(tc.expect),
		)

		b.Run("reflect.Convert/"+name, func(b *testing.B) {
			inVal := reflect.ValueOf(tc.input)
			expectType := reflect.TypeOf(tc.expect)

			for i := 0; i < b.N; i++ {
				_ = inVal.Convert(expectType)
			}
		})

		b.Run("convertNumber/"+name, func(b *testing.B) {
			inVal := reflect.ValueOf(tc.input)
			expectType := reflect.TypeOf(tc.expect)

			for i := 0; i < b.N; i++ {
				_ = convertNumber(inVal, expectType)
			}
		})
	}
}

func TestStringByteSliceConversion(t *testing.T) {
	t.Run("[]byte to string", func(t *testing.T) {
		input := []byte("Hello, world!")

		var actual string
		require.NoError(t, Decode(Encode(input), &actual))
		require.Equal(t, "Hello, world!", actual)
	})

	t.Run("string to []byte", func(t *testing.T) {
		input := "Hello, world!"

		var actual []byte
		require.NoError(t, Decode(Encode(input), &actual))
		require.Equal(t, []byte("Hello, world!"), actual)
	})
}

func TestValue_Call(t *testing.T) {
	addFunc := func(a, b int) int { return a + b }

	fv := Encode(addFunc)
	require.Equal(t, KindFunction, fv.Kind())

	// Test calling with values that need to be converted to an int.
	res, err := fv.Call(Uint(10), Float(23))
	require.NoError(t, err)

	require.Equal(t, int64(33), res.Int())
}
