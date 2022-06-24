// Package vm exposes an evaluator to convert River AST nodes into Go values.
package vm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/grafana/agent/pkg/river/ast"
	"github.com/grafana/agent/pkg/river/vm/internal/rivertags"
	"github.com/grafana/agent/pkg/river/vm/internal/value"
)

// TODO(rfratto): unfinished business for an MVP:
//
// 1. Deferred unmarshaling support ([]*ast.Block, []ast.Stmt)
// 2. Support encoding.TextUnmarshaler/encoding.TextMarshaler
// 3. Support custom UnmarshalRiver method on structs
// 4. Automatically determine when something should be a capsule
// 5. Function calls & stdlib
// 6. Allow decoding ast.Body
// 7. Make sure embedded fields work

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

	switch node := vm.node.(type) {
	case *ast.BlockStmt:
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer {
			panic(fmt.Sprintf("river: can only evaluate blocks into pointers, got %s", rv.Kind()))
		}
		rv = rv.Elem()

		return vm.evaluateBlock(scope, node, rv)
	default:
		val, err := vm.evaluateExpr(scope, node)
		if err != nil {
			return err
		}
		return decodeVal(val, v)
	}
}

func (vm *Evaluator) evaluateBlock(scope *Scope, node *ast.BlockStmt, rv reflect.Value) error {
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("river: can only evlauate blocks into struct pointers, got pointer to %s", rv.Kind()))
	}

	tfs := rivertags.Get(rv.Type())

	// Decode the block label first.
	if err := vm.evaluateBlockLabel(node, tfs, rv); err != nil {
		return err
	}

	var (
		foundAttrs  = make(map[string][]*ast.AttributeStmt, len(tfs))
		foundBlocks = make(map[string][]*ast.BlockStmt, len(tfs))
	)
	for _, stmt := range node.Body {
		switch stmt := stmt.(type) {
		case *ast.AttributeStmt:
			name := stmt.Name.Name
			foundAttrs[name] = append(foundAttrs[name], stmt)

		case *ast.BlockStmt:
			name := strings.Join(stmt.Name, ".")
			foundBlocks[name] = append(foundBlocks[name], stmt)

		default:
			panic(fmt.Sprintf("river: unrecognized ast.Stmt type %T", stmt))
		}
	}

	var (
		consumedAttrs  = make(map[string]struct{}, len(foundAttrs))
		consumedBlocks = make(map[string]struct{}, len(foundAttrs))
	)
	for _, tf := range tfs {
		if tf.IsAttr() {
			consumedAttrs[tf.Name] = struct{}{}
		} else if tf.IsBlock() {
			consumedBlocks[tf.Name] = struct{}{}
		}

		// Skip over ignored fields and fields that aren't attributes or blocks.
		if tf.IsIgnored() || (!tf.IsAttr() && !tf.IsBlock()) {
			continue
		}

		var (
			attrs  = foundAttrs[tf.Name]
			blocks = foundBlocks[tf.Name]
		)

		// Validity checks for attributes and blocks
		switch {
		case len(attrs) == 0 && len(blocks) == 0 && tf.IsOptional():
			// Optional field with no set values. Skip.
			continue

		case tf.IsAttr() && len(blocks) > 0:
			return fmt.Errorf("%q must be an attribute, but is used as a block", tf.Name)
		case tf.IsAttr() && len(attrs) == 0 && !tf.IsOptional():
			return fmt.Errorf("missing required attribute %q", tf.Name)
		case tf.IsAttr() && len(attrs) > 1:
			// While blocks may be specified multiple times (when the struct field
			// accepts a slice or an array), attributes may only ever be specified
			// once.
			return fmt.Errorf("attribute %q may only be set once", tf.Name)

		case tf.IsBlock() && len(attrs) > 0:
			return fmt.Errorf("%q must be a block, but is used as an attribute", tf.Name)
		case tf.IsBlock() && len(blocks) == 0 && !tf.IsOptional():
			// TODO(rfratto): does it ever make sense for children blocks to be required?
			return fmt.Errorf("missing required block %q", tf.Name)

		case len(attrs) > 0 && len(blocks) > 0:
			// NOTE(rfratto): it's not possible to reach this condition given the
			// statements above, but this is left in defensively in case there is a
			// bug with the validity checks.
			return fmt.Errorf("%q may only be used as a block or an attribute, but found both", tf.Name)
		}

		field := rv.Field(tf.Index)

		// Decode.
		switch {
		case tf.IsBlock():
			decodeField := prepareDecodeValue(field)

			switch decodeField.Kind() {
			case reflect.Slice:
				// Reset the slice length to zero.
				// TODO(rfratto): document this behavior
				decodeField.Set(reflect.MakeSlice(decodeField.Type(), len(blocks), len(blocks)))

				// Now, iterate over all of the block values and decode them
				// individually into the slice.
				for i, block := range blocks {
					decodeElement := prepareDecodeValue(decodeField.Index(i))
					err := vm.evaluateBlock(scope, block, decodeElement)
					if err != nil {
						return err
					}
				}

			case reflect.Array:
				for i := 0; i < decodeField.Len(); i++ {
					decodeElement := prepareDecodeValue(decodeField.Index(i))

					if i >= len(blocks) {
						// The array is longer than the number of blocks provided. Set the
						// rest to the zero value.
						// TODO(rfratto): document this behavior
						decodeElement.Set(reflect.Zero(decodeElement.Type()))
						continue
					}

					err := vm.evaluateBlock(scope, blocks[i], decodeElement)
					if err != nil {
						return err
					}
				}

			default:
				if len(blocks) > 1 {
					return fmt.Errorf("block %q may only be specified once", tf.Name)
				}
				err := vm.evaluateBlock(scope, blocks[0], decodeField)
				if err != nil {
					return err
				}
			}

		case tf.IsAttr():
			val, err := vm.evaluateExpr(scope, attrs[0].Value)
			if err != nil {
				return err
			}

			// We're reconverting our reflect.Value back into an interface{}, so we
			// need to also turn it back into a pointer for decoding.
			if err := decodeVal(val, field.Addr().Interface()); err != nil {
				return err
			}
		}
	}

	// Make sure that all of the attributes and blocks defined in the AST node
	// matched up with a field from our struct.
	for attr := range foundAttrs {
		if _, consumed := consumedAttrs[attr]; !consumed {
			return fmt.Errorf("unrecognized attribute name %q", attr)
		}
	}
	for block := range foundBlocks {
		if _, consumed := consumedBlocks[block]; !consumed {
			return fmt.Errorf("unrecognized block name %q", block)
		}
	}

	return nil
}

// prepareDecodeValue prepares v for decoding. Pointers will be fully
// deferenced until finding a non-pointer value. nil pointers will be
// allocated.
func prepareDecodeValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func (vm *Evaluator) evaluateBlockLabel(node *ast.BlockStmt, tfs rivertags.Fields, rv reflect.Value) error {
	var (
		labelField rivertags.Field
		foundField bool
	)
	for _, tf := range tfs {
		if tf.IsLabel() {
			labelField = tf
			foundField = true
			break
		}
	}

	// Check for user errors first:
	switch {
	case node.Label == "" && foundField: // No user label, but struct expects one
		return fmt.Errorf("block %q requires non-empty label", strings.Join(node.Name, "."))
	case node.Label != "" && !foundField: // User label, but struct doesn't expect one
		return fmt.Errorf("block %q does not support specifying labels", strings.Join(node.Name, "."))
	}

	if node.Label == "" {
		// no-op: no labels to set
		return nil
	}

	var (
		field     = rv.Field(labelField.Index)
		fieldType = field.Type()
	)
	if !reflect.TypeOf(node.Label).AssignableTo(fieldType) {
		panic(fmt.Sprintf("river: cannot decode block label into non-string type %s", fieldType))
	}
	field.Set(reflect.ValueOf(node.Label))
	return nil
}

func (vm *Evaluator) evaluateExpr(scope *Scope, node ast.Node) (v value.Value, err error) {
	switch node := node.(type) {
	case *ast.LiteralExpr:
		return valueFromLiteral(node.Value, node.Kind)

	case *ast.BinaryExpr:
		lhs, err := vm.evaluateExpr(scope, node.Left)
		if err != nil {
			return lhs, err
		}
		rhs, err := vm.evaluateExpr(scope, node.Right)
		if err != nil {
			return rhs, err
		}

		// TODO(rfratto): check types before doing this, otherwise invalid types
		// for operators below will panic, i.e.: `3 || true`
		return value.Binop(lhs, node.Kind, rhs), nil

	case *ast.ArrayExpr:
		var vals []value.Value
		for _, element := range node.Elements {
			val, err := vm.evaluateExpr(scope, element)
			if err != nil {
				return value.Null, err
			}
			vals = append(vals, val)
		}
		if len(vals) == 0 {
			return value.Array(), nil
		}
		return value.Array(vals...), nil

	case *ast.ObjectExpr:
		attrs := make(map[string]value.Value, len(node.Fields))
		for _, field := range node.Fields {
			val, err := vm.evaluateExpr(scope, field.Value)
			if err != nil {
				return value.Null, err
			}
			attrs[field.Name] = val
		}
		return value.Map(attrs), nil

	case *ast.IdentifierExpr:
		val := findIdentifier(scope, node.Name)
		if val == nil {
			return value.Null, fmt.Errorf("identifier %q does not exist", node.Name)
		}
		return value.Encode(val), nil

	case *ast.AccessExpr:
		val, err := vm.evaluateExpr(scope, node.Value)
		if err != nil {
			return val, err
		}

		switch val.Kind() {
		case value.KindMap:
			res, ok := val.MapIndex(node.Name)
			if !ok {
				return value.Null, fmt.Errorf("field %q does not exist", node.Name)
			}
			return res, nil
		case value.KindObject:
			// TODO(rfratto): this is really inefficient
			res, ok := val.KeyByName(node.Name)
			if !ok {
				return value.Null, fmt.Errorf("field %q does not exist", node.Name)
			}
			return res, nil
		default:
			return value.Null, fmt.Errorf("cannot access field %q on non-object or map type %s", node.Name, val.Type())
		}

	case *ast.IndexExpr:
		val, err := vm.evaluateExpr(scope, node.Value)
		if err != nil {
			return val, err
		}
		idx, err := vm.evaluateExpr(scope, node.Index)
		if err != nil {
			return val, err
		}

		if val.Kind() != value.KindArray {
			return value.Null, fmt.Errorf("cannot take an index of non-list type %s", val.Type())
		}
		if idx.Type().Kind() != value.KindNumber {
			return value.Null, fmt.Errorf("type %s cannot be used to index objects", idx.Type())
		}
		return val.Index(int(idx.Int())), nil

	case *ast.ParenExpr:
		return vm.evaluateExpr(scope, node.Inner)

	case *ast.UnaryExpr:
		val, err := vm.evaluateExpr(scope, node.Expression)
		if err != nil {
			return val, err
		}

		// TODO(rfratto): check types before doing this, otherwise invalid types
		// for operators below will panic, i.e.: `!3`
		return value.Unary(node.Kind, val), nil

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

func decodeVal(val value.Value, v interface{}) error {
	return value.Decode(val, v)
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
