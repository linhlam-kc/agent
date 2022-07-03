package vm

import (
	"fmt"
	"strings"

	"github.com/grafana/agent/pkg/river/ast"
	"github.com/grafana/agent/pkg/river/printer"
	"github.com/grafana/agent/pkg/river/vm/internal/value"
)

type ValueError struct {
	// Node where the error originated.
	Node ast.Node

	// Error message (i.e., foobar should be number, got string).
	Message string

	// Value is the printed value which caused the error.
	Value string

	// Indirect indicates that the value which caused the error was deeply nested
	// inside the Node.
	//
	// Indirect should be used to determine whether the Value field should be
	// printed. When Indirect is false, it indicates that the value which caused
	// the error is viewable on the line number of Node.
	Indirect bool
}

// Error returns the short-form error message of ve.
func (ve ValueError) Error() string {
	if ve.Node != nil {
		return fmt.Sprintf("%s: %s", ast.StartPos(ve.Node).Position(), ve.Message)
	}
	return ve.Message
}

// convertValueError converts the err into an ValueError. If err is not a
// value-related Error from the value package, err is returned unmodified.
func convertValueError(err error, assoc map[value.Value]ast.Node) error {
	// Build up the expression
	var (
		node      ast.Node
		expr      strings.Builder
		message   string
		valueText string

		// Start off as being indirect until we find a node.
		indirect = true
	)

	isValueError := value.WalkError(err, func(err error) {
		var val value.Value

		switch ne := err.(type) {
		case value.TypeError:
			// TODO(rfratto): print value
			message = fmt.Sprintf("should be %s, got %s", ne.Expected, ne.Value.Kind())
			val = ne.Value
		case value.MissingKeyError:
			// TODO(rfratto): print value
			message = fmt.Sprintf("does not have field named %q", ne.Missing)
			val = ne.Value
		case value.ElementError:
			fmt.Fprintf(&expr, "[%d]", ne.Index)
			val = ne.Value
		case value.FieldError:
			fmt.Fprintf(&expr, ".%s", ne.Field)
			val = ne.Value
		}

		if foundNode, ok := assoc[val]; ok {
			// If we just found a direct node, we can reset the expression buffer.
			if !indirect {
				expr.Reset()
			}

			node = foundNode
			indirect = false
		} else {
			indirect = true
		}
	})
	if !isValueError {
		return err
	}

	if node != nil {
		var nodeText strings.Builder
		if err := printer.Print(&nodeText, node); err != nil {
			// TODO(rfratto): is it OK for this to panic?
			panic(err)
		}

		// Merge the node text with the expression together (which will be relative
		// accesses to the expression).
		message = fmt.Sprintf("%s%s %s", nodeText.String(), expr.String(), message)
	} else {
		message = fmt.Sprintf("%s %s", expr.String(), message)
	}

	return ValueError{
		Node:     node,
		Message:  message,
		Value:    valueText,
		Indirect: indirect,
	}
}
