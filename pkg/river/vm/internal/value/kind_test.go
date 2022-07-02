package value

import (
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capsuleMarked is a type which would normally be a River number, but should
// instead be a capsule because it implements the capsuleMarker interface.
type capsuleMarked int

func (capsuleMarked) RiverCapsule() {}

var kindTests = []struct {
	input  interface{}
	expect Kind
}{
	{int(0), KindNumber},
	{int8(0), KindNumber},
	{int16(0), KindNumber},
	{int32(0), KindNumber},
	{int64(0), KindNumber},
	{uint(0), KindNumber},
	{uint8(0), KindNumber},
	{uint16(0), KindNumber},
	{uint32(0), KindNumber},
	{uint64(0), KindNumber},
	{float32(0), KindNumber},
	{float64(0), KindNumber},

	{string(""), KindString},

	{bool(false), KindBool},

	{[...]int{0, 1, 2}, KindArray},
	{[]int{0, 1, 2}, KindArray},

	{struct {
		Name string `river:"name,attr"`
	}{}, KindObject},
	{struct{ Untagged string }{}, KindObject},
	{struct{}{}, KindObject},

	{map[string]interface{}{}, KindObject},

	{func() {}, KindFunction},

	{make(chan struct{}), KindCapsule},
	{map[bool]interface{}{}, KindCapsule}, // Maps with non-string types are capsules
	{capsuleMarked(0), KindCapsule},       // Types which implement capsuleMarker should be capsules.
}

func Test_kindFromType(t *testing.T) {
	for _, tc := range kindTests {
		rt := reflect.TypeOf(tc.input)

		t.Run(rt.String(), func(t *testing.T) {
			actual := kindFromType(rt)
			assert.Equal(t, tc.expect, actual, "Unexpected type for %#v", tc.input)
		})
	}
}

func Benchmark_kindFromType(b *testing.B) {
	for _, tc := range kindTests {
		rt := reflect.TypeOf(tc.input)

		b.Run(rt.String(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = kindFromType(rt)
			}
		})
	}
}

func Test_KindAny(t *testing.T) {
	type container struct {
		Value  interface{} `river:"value,attr"`
		Writer io.Writer   `river:"writer,attr"`
	}

	containerTy := reflect.TypeOf(container{})

	kind := kindFromType(containerTy)
	require.Equal(t, KindObject, kind)

	// The first key should be an any kind since it's the empty interface.
	// Everything else should be capsules.
	require.Equal(t, KindAny, kindFromType(containerTy.Field(0).Type))
	require.Equal(t, KindCapsule, kindFromType(containerTy.Field(1).Type))
}
