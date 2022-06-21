// Package vm exposes an evaluator to convert River AST nodes into Go values.
package vm

import (
	"fmt"

	"github.com/grafana/agent/pkg/river/ast"
	"github.com/grafana/agent/pkg/river/token"
	"github.com/grafana/agent/pkg/river/vm/internal/ctyencode"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Evaluator converts River AST nodes into Go values. Each Evaluator is bound
// to a single AST node to allow it to precompute omptimizations before
// conversion into a Go value. To evaluate the node, call Evaluate.
type Evaluator struct {
	node ast.Node
}

// New creates a new Evaluator for the given AST node. The given node must be
// either an ast.Body, ast.BlockStmt, or assignable to an ast.Expr.
func New(node ast.Node) *Evaluator {
	return &Evaluator{node: node}
}

// Evaluate converts the Evaluator's AST node into the Go value v.
//
// River values are converted into Go values with the following rules:
//
//   - River integers evaluate to Go integers or floating point
//     values.
//
//   - River floating point numbers evaluate to Go floating point values.
//
//   - River strings evaluate to Go strings. Evaluation to Go integers and
//     floating point values is permitted if the string can be interpreted as
//     a decimal number of one of those types.
//
//   - River arrays evaluate to Go array or slice values.
//
//   - River bodies and objects evaluate to a Go map[string]interface{} or Go
//     structs, where the inner attributes and blocks evaluate to exported
//     struct fields using the rules described in the "Struct encoding"
//     section.
//
//   - River blocks evaluate to a Go struct, where the inner attributes and
//     blocks evaluate to exported struct fields using the rules described in
//     the "Struct encoding" section.
//
// When evaluating any value, Go values may implement the Unmarshaler interface. The
// UnmarshalRiver method will be called whenever the value is being evaluated.
//
// When evaluating a River string, Go values may implement the
// encoding.TextUnmarshaler interface. The UnmarshalText method will be called
// whenever the value is being evaluated.
//
// Finally, River values may be evaluated into an interface{} with one of the
// following:
//
//   - bool for River bools
//   - int64 for River numbers
//   - string for River strings
//   - []interface{} for River arrays
//   - map[string]interface{} for River bodies and objects
//   - Block for River blocks
//
// Struct encoding
//
// Structs can be used as the targets for River bodies, objects, and blocks.
// Tags on exported fields are used to specify behavior for expected attributes
// and blocks during the evaluation:
//
//   1. A field with tag "-" is omitted.
//
//   2. A field with tag "name,attr" becomes a required attribute with the
//      given name.
//
//   3. A field with tag "name,attr,optional" becomes an optional attribute
//      with the given name.
//
//   4. A field with tag "name,block" becomes a required block with the given
//      name. It is invalid to use this tag when decoding objects, as objects
//      can not contain child blocks.
//
//   5. A field with name "name,block,optional" becomes an optional block with
//      the given name. It is invalid to use this tag when decoding objects, as
//      objects can not contain child blocks.
//
//   6. A field with the name ",label" maps to the label of a block (when
//      decoding a block). It is only valid to use this tag when decoding blocks,
//      as that is the only time block labels are available.
//
//   7. Anonymous struct fields are handled as if the fields of its value were
//      part of the outer struct.
//
// Fields with tag "name,block" or "name,block,optional" may be slices to
// support multiple children blocks with the same name.
func (vm *Evaluator) Evaluate(scope *Scope, v interface{}) (err error) {
	defer func() {
		if err != nil {
			// Wrap error with line information if the AST node has a valid position
			pos := ast.StartPos(vm.node).Position()
			if pos.IsValid() {
				err = fmt.Errorf("%s: %w", pos, err)
			}
		}
	}()

	val, err := vm.evaluate(scope, vm.node)
	if err != nil {
		return err
	}
	return decodeVal(val, v)
}

func (vm *Evaluator) evaluate(scope *Scope, node ast.Node) (v cty.Value, err error) {
	// TODO(rfratto): other expr types:
	// - CallExpr
	//
	// And then:
	// - BlockStmt
	// - Body

	switch node := node.(type) {
	case *ast.LiteralExpr:
		return valueFromLiteral(node.Value, node.Kind)

	case *ast.BinaryExpr:
		lhs, err := vm.evaluate(scope, node.Left)
		if err != nil {
			return lhs, err
		}
		rhs, err := vm.evaluate(scope, node.Right)
		if err != nil {
			return rhs, err
		}

		// TODO(rfratto): check types before doing this, otherwise invalid types
		// for operators below will panic, i.e.: `3 || true`

		switch node.Kind {
		case token.OR:
			return lhs.Or(rhs), nil
		case token.AND:
			return lhs.And(rhs), nil
		case token.EQ:
			return lhs.Equals(rhs), nil
		case token.NEQ:
			return lhs.NotEqual(rhs), nil
		case token.LT:
			return lhs.LessThan(rhs), nil
		case token.LTE:
			return lhs.LessThanOrEqualTo(rhs), nil
		case token.GT:
			return lhs.GreaterThan(rhs), nil
		case token.GTE:
			return lhs.GreaterThanOrEqualTo(rhs), nil
		case token.ADD:
			return lhs.Add(rhs), nil
		case token.SUB:
			return lhs.Subtract(rhs), nil
		case token.MUL:
			return lhs.Multiply(rhs), nil
		case token.DIV:
			return lhs.Divide(rhs), nil
		case token.MOD:
			return lhs.Modulo(rhs), nil
		default:
			panic(fmt.Sprintf("unrecognized binary operator %q", node.Kind))
		}

	case *ast.ArrayExpr:
		var vals []cty.Value
		for _, element := range node.Elements {
			val, err := vm.evaluate(scope, element)
			if err != nil {
				return cty.NilVal, err
			}
			vals = append(vals, val)
		}
		if len(vals) == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType), nil
		}
		return cty.ListVal(vals), nil

	case *ast.ObjectExpr:
		attrs := make(map[string]cty.Value, len(node.Fields))
		for _, field := range node.Fields {
			val, err := vm.evaluate(scope, field.Value)
			if err != nil {
				return cty.NilVal, err
			}
			attrs[field.Name] = val
		}
		return cty.ObjectVal(attrs), nil

	case *ast.IdentifierExpr:
		val := findIdentifier(scope, node.Name)
		if val == nil {
			return cty.NilVal, fmt.Errorf("identifier %q does not exist", node.Name)
		}
		valTy, err := ctyencode.ImpliedType(val)
		if err != nil {
			return cty.NilVal, err
		}
		return ctyencode.ToCtyValue(val, valTy)

	case *ast.AccessExpr:
		val, err := vm.evaluate(scope, node.Value)
		if err != nil {
			return val, err
		}

		if !val.Type().IsObjectType() {
			return cty.NilVal, fmt.Errorf("cannot access field %q on non-object type %s", node.Name, val.Type().FriendlyName())
		} else if !val.Type().HasAttribute(node.Name) {
			return cty.NilVal, fmt.Errorf("field %q does not exist", node.Name)
		}
		return val.GetAttr(node.Name), nil

	case *ast.IndexExpr:
		val, err := vm.evaluate(scope, node.Value)
		if err != nil {
			return val, err
		}
		idx, err := vm.evaluate(scope, node.Index)
		if err != nil {
			return val, err
		}

		if !val.Type().IsListType() && !val.Type().IsTupleType() {
			return cty.NilVal, fmt.Errorf("cannot take an index of non-list type %s", val.Type().FriendlyName())
		}
		if !idx.Type().Equals(cty.Number) {
			return cty.NilVal, fmt.Errorf("type %s cannot be used to index objects", idx.Type().FriendlyName())
		}
		return val.Index(idx), nil

	case *ast.ParenExpr:
		return vm.evaluate(scope, node.Inner)

	case *ast.UnaryExpr:
		val, err := vm.evaluate(scope, node.Expression)
		if err != nil {
			return val, err
		}

		// TODO(rfratto): check types before doing this, otherwise invalid types
		// for operators below will panic, i.e.: `!3`

		switch node.Kind {
		case token.NOT:
			return val.Not(), nil
		case token.SUB:
			return val.Negate(), nil
		default:
			panic(fmt.Sprintf("unrecognized unary operator %q", node.Kind))
		}

	default:
		panic(fmt.Sprintf("unexpected ast.Node type %T", node))
	}
}

func findIdentifier(scope *Scope, name string) interface{} {
	for scope != nil {
		if val, ok := scope.Variables[name]; ok {
			return val
		}
		scope = scope.Parent
	}
	return nil
}

func decodeVal(val cty.Value, v interface{}) error {
	valTy := val.Type()

	targetType, err := ctyencode.ImpliedType(v)
	if err != nil {
		return err
	}

	if !valTy.Equals(targetType) {
		conv := convert.GetConversionUnsafe(valTy, targetType)
		if conv == nil {
			return fmt.Errorf("cannot convert from %s to %s", valTy.FriendlyName(), targetType.FriendlyName())
		}
		val, err = conv(val)
		if err != nil {
			return err
		}
	}

	return ctyencode.FromCtyValue(val, v)
}

// A Scope exposes a set of identifiers available to use when evaluating a
// Node.
type Scope struct {
	// Parent optionally points to a Parent scope containing more variables.
	// Identifier lookups are done first by searching the child scope, and
	// continually searching up more scopes until an indentifier is found or all
	// scopes are exhausted.
	Parent *Scope

	// Variables holds the list of available identifiers that can be used when
	// evaluating a Node.
	//
	// If a Go value cannot be interpreted as a River number, string, bool,
	// array, object, or block, it is carried around verbatim as an encapsulated
	// Go value. value." This allows for expressions that deal with arbitrary Go
	// values. However, such values can only be evaluated to Go values of the
	// same type.
	Variables map[string]interface{}
}
