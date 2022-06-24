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

func TestValue_Call(t *testing.T) {
	addFunc := func(a, b int) int { return a + b }

	fv := Encode(addFunc)
	require.Equal(t, KindFunction, fv.Kind())

	// Test calling with values that need to be converted to an int.
	res, err := fv.Call(Uint(10), Float(23))
	require.NoError(t, err)

	require.Equal(t, int64(33), res.Int())
}
