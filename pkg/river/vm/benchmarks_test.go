package vm_test

import (
	"reflect"
	"testing"

	"github.com/grafana/agent/pkg/river/parser"
	"github.com/grafana/agent/pkg/river/vm"
	"github.com/stretchr/testify/require"
)

func BenchmarkExprs(b *testing.B) {
	// Shared scope across all tests below
	scope := &vm.Scope{
		Variables: map[string]interface{}{
			"foobar": int(42),
		},
	}

	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		// Binops
		{"or", `false || true`, bool(true)},
		{"and", `true && false`, bool(false)},
		{"eq", `3 == 5`, bool(false)},
		{"neq", `3 != 5`, bool(true)},
		{"lt", `3 < 5`, bool(true)},
		{"lte", `3 <= 5`, bool(true)},
		{"gt", `3 > 5`, bool(false)},
		{"gte", `3 >= 5`, bool(false)},
		{"add", `3 + 5`, int(8)},
		{"sub", `3 - 5`, int(-2)},
		{"mul", `3 * 5`, int(15)},
		{"div", `3.0 / 5.0`, float64(0.6)},
		{"mod", `5 % 3`, int(2)},
		// {"pow", `3 ^ 5`, int(243)}, // TODO(rfratto): implement pow
		{"binop chain", `3 + 5 * 2`, int(13)}, // Chain multiple binops

		// Identifier
		{"ident lookup", `foobar`, int(42)},

		// Arrays
		{"array", `[0, 1, 2]`, []int{0, 1, 2}},

		// Objects
		{"object to map", `{ a = 5, b = 10 }`, map[string]int{"a": 5, "b": 10}},
		{
			name: "object to struct",
			input: `{
					name = "John Doe", 
					age = 42,
			}`,
			expect: struct {
				Name    string `river:"name,key"`
				Age     int    `river:"age,key"`
				Country string `river:"country,key,optional"`
			}{
				Name: "John Doe",
				Age:  42,
			},
		},

		// Access
		{"access", `{ a = 15 }.a`, int(15)},
		{"nested access", `{ a = { b = 12 } }.a.b`, int(12)},

		// Indexing
		{"index", `[0, 1, 2][1]`, int(1)},
		{"nested index", `[[1,2,3]][0][2]`, int(3)},

		// Paren
		{"paren", `(15)`, int(15)},

		// Unary
		{"unary not", `!true`, bool(false)},
		{"unary neg", `-15`, int(-15)},
	}

	for _, tc := range tt {
		b.Run(tc.name, func(b *testing.B) {
			b.StopTimer()
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(b, err)

			eval := vm.New(expr)

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				vPtr := reflect.New(reflect.TypeOf(tc.expect)).Interface()
				require.NoError(b, eval.Evaluate(scope, vPtr))

				actual := reflect.ValueOf(vPtr).Elem().Interface()
				require.Equal(b, tc.expect, actual)
			}
		})
	}
}
