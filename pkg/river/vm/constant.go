package vm

import (
	"fmt"
	"strconv"

	"github.com/grafana/agent/pkg/river/token"
	"github.com/grafana/agent/pkg/river/vm/internal/value"
)

func valueFromLiteral(lit string, tok token.Token) (value.Value, error) {
	switch tok {
	case token.NUMBER:
		v, err := strconv.ParseInt(lit, 0, 64)
		if err != nil {
			return value.Null, err
		}
		return value.Int(v), nil

	case token.FLOAT:
		v, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return value.Null, err
		}
		return value.Float(v), nil

	case token.STRING:
		v, err := strconv.Unquote(lit)
		if err != nil {
			return value.Null, err
		}
		return value.String(v), nil

	case token.BOOL:
		switch lit {
		case "true":
			return value.Bool(true), nil
		case "false":
			return value.Bool(false), nil
		default:
			return value.Null, fmt.Errorf("invalid boolean literal %q", lit)
		}
	default:
		panic(fmt.Sprintf("%v is not a valid token", tok))
	}
}
