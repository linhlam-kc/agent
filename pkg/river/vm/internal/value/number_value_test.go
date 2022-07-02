package value

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_numberValue(t *testing.T) {
	tt := []struct {
		input  interface{}
		expect numberValue
	}{
		{uint(10), numberValue{10, nativeUintBits, numberKindUint}},
		{uint8(20), numberValue{20, 8, numberKindUint}},
		{uint16(30), numberValue{30, 16, numberKindUint}},
		{uint32(40), numberValue{40, 32, numberKindUint}},
		{uint64(50), numberValue{50, 64, numberKindUint}},
		{int(60), numberValue{60, nativeIntBits, numberKindInt}},
		{int8(70), numberValue{70, 8, numberKindInt}},
		{int16(80), numberValue{80, 16, numberKindInt}},
		{int32(90), numberValue{90, 32, numberKindInt}},
		{int64(100), numberValue{100, 64, numberKindInt}},
		{float32(55), numberValue{math.Float64bits(55), 32, numberKindFloat}},
		{float64(105), numberValue{math.Float64bits(105), 64, numberKindFloat}},
	}

	for _, tc := range tt {
		val := reflect.ValueOf(tc.input)
		t.Run(val.Type().String(), func(t *testing.T) {
			require.Equal(t, tc.expect, newNumberValue(val))
		})
	}
}

func Benchmark_numberValue(b *testing.B) {
	tt := []interface{}{
		uint(0),
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		int(0),
		int8(0),
		int16(0),
		int32(0),
		int64(0),
		float32(0),
		float64(0),
	}

	for _, tc := range tt {
		val := reflect.ValueOf(tc)
		b.Run(val.Type().String(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = newNumberValue(val)
			}
		})
	}
}
