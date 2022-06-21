// Package ctyencode deals with converting between cty Values and native go
// values.
//
// It operates under a similar principle to the encoding/json and
// encoding/xml packages in the standard library, using reflection to
// populate native Go data structures from cty values and vice-versa.
//
// ctyencode is a River-focused fork of github.com/zclconf/go-cty/gocty.
package ctyencode

import (
	"math/big"
	"reflect"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/set"
)

var (
	valueType = reflect.TypeOf(cty.Value{})
	typeType  = reflect.TypeOf(cty.Type{})

	setType = reflect.TypeOf(set.Set{})

	bigFloatType = reflect.TypeOf(big.Float{})
	bigIntType   = reflect.TypeOf(big.Int{})

	emptyInterfaceType = reflect.TypeOf(interface{}(nil))

	stringType = reflect.TypeOf("")
)
