package rivertags

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getTaggedFields_Block(t *testing.T) {
	type testBlockStruct struct {
		IgnoreMe bool `river:"-"`

		ReqAttr  string   `river:"req_attr,attr"`
		OptAttr  string   `river:"opt_attr,attr,optional"`
		ReqBlock struct{} `river:"req_block,block"`
		OptBlock struct{} `river:"opt_block,block,optional"`
		Label    string   `river:",label"`
	}

	tfs := Get(reflect.TypeOf(testBlockStruct{}))
	require.True(t, tfs.BlockKind(), "expected testBlockStruct to be a BlockKind")
	require.False(t, tfs.ObjectKind(), "expected testBlockStruct to not be an ObjectKind")

	expect := []Field{
		{Name: "-", Index: 0},
		{Name: "req_attr", Index: 1, flags: flagAttr},
		{Name: "opt_attr", Index: 2, flags: flagAttr | flagOptional},
		{Name: "req_block", Index: 3, flags: flagBlock},
		{Name: "opt_block", Index: 4, flags: flagBlock | flagOptional},
		{Name: "", Index: 5, flags: flagLabel},
	}

	require.Equal(t, expect, []Field(tfs))
}

func Test_getTaggedFields_Object(t *testing.T) {
	type testObjectStruct struct {
		IgnoreMe bool `river:"-"`

		ReqKey string `river:"req_key,key"`
		OptKey string `river:"opt_key,key,optional"`
	}

	tfs := Get(reflect.TypeOf(testObjectStruct{}))
	require.True(t, tfs.ObjectKind(), "expected testBlockStruct to be an ObjectKind")
	require.False(t, tfs.BlockKind(), "expected testBlockStruct to not be a BlockKind")

	expect := []Field{
		{Name: "-", Index: 0},
		{Name: "req_key", Index: 1, flags: flagKey},
		{Name: "opt_key", Index: 2, flags: flagKey | flagOptional},
	}

	require.Equal(t, expect, []Field(tfs))
}

func Test_getTaggedFields_Invalid(t *testing.T) {
	type testMixedStruct struct {
		BlockField  string `river:"block_field,attr"`
		ObjectField string `river:"object_field,key,optional"`
	}

	expect := "river: struct rivertags.testMixedStruct has tags for both objects and blocks at the same time"
	require.PanicsWithValue(t, expect, func() {
		_ = Get(reflect.TypeOf(testMixedStruct{}))
	})
}

func Test_getTaggedFields_ReusedField(t *testing.T) {
	type testBlockStruct struct {
		First  string `river:"field1,attr"`
		Reused string `river:"field1,attr,optional"`
	}

	expect := "river: field1 already used in struct rivertags.testBlockStruct by field First"
	require.PanicsWithValue(t, expect, func() {
		_ = Get(reflect.TypeOf(testBlockStruct{}))
	})
}
