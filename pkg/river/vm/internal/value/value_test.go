package value

import (
	"testing"
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
