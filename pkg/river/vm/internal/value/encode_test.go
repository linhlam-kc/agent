package value_test

import (
	"reflect"
	"testing"

	"github.com/grafana/agent/pkg/river/vm/internal/value"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

func TestEncode_Decode(t *testing.T) {
	tt := []struct {
		name   string
		in     interface{}
		expect interface{}
	}{
		{
			name:   "array of *int to array of int",
			in:     []*int{pointer.Int(5), pointer.Int(10)},
			expect: []int{5, 10},
		},
		{
			name:   "array of interface *int to array of int",
			in:     []interface{}{pointer.Int(5), pointer.Int(10)},
			expect: []int{5, 10},
		},
		{
			name:   "array of interface int to array of int",
			in:     []interface{}{5, 10},
			expect: []int{5, 10},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			val := value.Encode(tc.in)

			outPtr := reflect.New(reflect.TypeOf(tc.expect)).Interface()
			require.NoError(t, value.Decode(val, outPtr))

			actual := reflect.ValueOf(outPtr).Elem().Interface()
			require.Equal(t, tc.expect, actual)
		})
	}
}
