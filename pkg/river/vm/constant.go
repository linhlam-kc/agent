package vm

import (
	"fmt"
	"strconv"

	"github.com/grafana/agent/pkg/river/token"
	"github.com/zclconf/go-cty/cty"
)

func valueFromLiteral(lit string, tok token.Token) (cty.Value, error) {
	switch tok {
	case token.NUMBER:
		v, err := strconv.ParseInt(lit, 0, 64)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.NumberIntVal(v), nil

	case token.FLOAT:
		v, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.NumberFloatVal(v), nil

	case token.STRING:
		v, err := strconv.Unquote(lit)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(v), nil

	case token.BOOL:
		switch lit {
		case "true":
			return cty.BoolVal(true), nil
		case "false":
			return cty.BoolVal(false), nil
		default:
			return cty.NilVal, fmt.Errorf("invalid boolean literal %q", lit)
		}
	default:
		panic(fmt.Sprintf("%v is not a valid token", tok))
	}
}
