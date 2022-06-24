package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestType_ArrayOfAny(t *testing.T) {
	var v []interface{}

	ty := Encode(v).Type()
	require.Equal(t, KindArray, ty.Kind())
	require.Equal(t, KindAny, ty.Elem().Kind())
}

func TestType_Equals(t *testing.T) {
	tt := []struct {
		a, b   interface{}
		expect bool
	}{
		{int(0), int(0), true},                   // number
		{int(0), int8(0), true},                  // number
		{int(0), float32(0), true},               // number
		{string(""), string(""), true},           // string
		{bool(false), bool(false), true},         // bool
		{[]int{}, []int{}, true},                 // []number
		{[]int{}, []int8{}, true},                // []number
		{[]interface{}{}, []interface{}{}, true}, // []any

		{make(chan int), make(chan int), true},   // capsule(chan int)
		{make(chan int), make(chan int8), false}, // capsule(chan int), capsule(chan int8)
	}

	for _, tc := range tt {
		valA, valB := Encode(tc.a), Encode(tc.b)

		if tc.expect {
			assert.True(t, valA.Type().Equals(valB.Type()),
				"Expected %#v and %#v to be equal river types (%s and %s)",
				tc.a, tc.b, valA.Type(), valB.Type(),
			)
			// The inverse equality should also be true
			assert.True(t, valB.Type().Equals(valA.Type()),
				"Expected %#v and %#v to be equal river types (%s and %s)",
				tc.b, tc.a, valB.Type(), valA.Type(),
			)
		} else {
			assert.False(t, valA.Type().Equals(valB.Type()),
				"Expected %#v and %#v to be not equal river types (%s and %s)",
				tc.a, tc.b, valA.Type(), valB.Type(),
			)
			// Test the inverse equality
			assert.False(t, valB.Type().Equals(valA.Type()),
				"Expected %#v and %#v to be not equal river types (%s and %s)",
				tc.b, tc.a, valB.Type(), valA.Type(),
			)
		}
	}
}
