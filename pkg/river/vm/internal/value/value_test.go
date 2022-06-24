package value

import (
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

	b.Run("Type", func(b *testing.B) {
		v := Encode(p)
		for i := 0; i < b.N; i++ {
			_ = v.Type()
		}
	})
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
