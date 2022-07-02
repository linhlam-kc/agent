package value

import (
	"fmt"
	"math"
	"reflect"

	"github.com/grafana/agent/pkg/river/token"
)

// Unary performs a unary operation on the provided value. Unary will panic if
// op is not a valid unary operator or if the value is not the right type.
func Unary(op token.Token, val Value) Value {
	switch op {
	case token.NOT:
		if val.Kind() != KindBool {
			panic("river/vm: binary operation done on non-boolean value")
		}
		return Bool(!val.v.Bool())
	case token.SUB:
		if val.Kind() != KindNumber {
			panic("river/vm: binary operation done on non-number value")
		}
		switch makeNumberKind(val.v.Kind()) {
		case numberKindInt:
			return Int(-val.v.Int())
		case numberKindUint:
			return Int(-int64(val.v.Uint()))
		case numberKindFloat:
			return Float(-val.v.Float())
		}
	}
	panic(fmt.Sprintf("unrecognized binary operator %s", op))
}

// Binop performs a binary operation on the left and right values. Binop will
// panic if op is not a valid binary operator or if left or right are not the
// right values.
//
// Binop will determine what type the result of the operation should be and
// convert left and right appropriately.
func Binop(left Value, op token.Token, right Value) Value {
	switch op {
	case token.AND, token.OR:
		return logicalBinop(left, op, right)
	default:
		return numericalBinop(left, op, right)
	}
}

func logicalBinop(left Value, op token.Token, right Value) Value {
	if left.Kind() != KindBool || right.Kind() != KindBool {
		panic("river/vm: binary operation done on non-boolean value")
	}

	switch op {
	case token.OR:
		return Bool(left.v.Bool() || right.v.Bool())
	case token.AND:
		return Bool(left.v.Bool() && right.v.Bool())
	default:
		panic(fmt.Sprintf("unrecognized binary operator %s", op))
	}
}

func numericalBinop(left Value, op token.Token, right Value) Value {
	needType := fitNumberTypes(left.v.Type(), right.v.Type())
	leftV, err := convertBasicValue(left.v, needType)
	if err != nil {
		panic(err)
	}
	rightV, err := convertBasicValue(right.v, needType)
	if err != nil {
		panic(err)
	}

	nk := makeNumberKind(needType.Kind())

	switch op {
	case token.EQ:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() == rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() == rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() == rightV.Float())
		}
	case token.NEQ:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() != rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() != rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() != rightV.Float())
		}
	case token.LT:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() < rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() < rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() < rightV.Float())
		}
	case token.LTE:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() <= rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() <= rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() <= rightV.Float())
		}
	case token.GT:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() > rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() > rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() > rightV.Float())
		}
	case token.GTE:
		switch nk {
		case numberKindInt:
			return Bool(leftV.Int() >= rightV.Int())
		case numberKindUint:
			return Bool(leftV.Uint() >= rightV.Uint())
		case numberKindFloat:
			return Bool(leftV.Float() >= rightV.Float())
		}
	case token.ADD:
		switch nk {
		case numberKindInt:
			return Int(leftV.Int() + rightV.Int())
		case numberKindUint:
			return Uint(leftV.Uint() + rightV.Uint())
		case numberKindFloat:
			return Float(leftV.Float() + rightV.Float())
		}
	case token.SUB:
		switch nk {
		case numberKindInt:
			return Int(leftV.Int() - rightV.Int())
		case numberKindUint:
			return Uint(leftV.Uint() - rightV.Uint())
		case numberKindFloat:
			return Float(leftV.Float() - rightV.Float())
		}
	case token.MUL:
		switch nk {
		case numberKindInt:
			return Int(leftV.Int() * rightV.Int())
		case numberKindUint:
			return Uint(leftV.Uint() * rightV.Uint())
		case numberKindFloat:
			return Float(leftV.Float() * rightV.Float())
		}
	case token.DIV:
		switch nk {
		case numberKindInt:
			return Int(leftV.Int() / rightV.Int())
		case numberKindUint:
			return Uint(leftV.Uint() / rightV.Uint())
		case numberKindFloat:
			return Float(leftV.Float() / rightV.Float())
		}
	case token.MOD:
		switch nk {
		case numberKindInt:
			return Int(leftV.Int() % rightV.Int())
		case numberKindUint:
			return Uint(leftV.Uint() % rightV.Uint())
		case numberKindFloat:
			return Float(math.Mod(leftV.Float(), rightV.Float()))
		}
	case token.POW:
		switch nk {
		case numberKindInt:
			return Int(intPow(leftV.Int(), rightV.Int()))
		case numberKindUint:
			return Uint(intPow(leftV.Uint(), rightV.Uint()))
		case numberKindFloat:
			return Float(math.Pow(leftV.Float(), rightV.Float()))
		}
	}

	panic(fmt.Sprintf("unrecognized binary operator %s", op))
}

func intPow[Number int64 | uint64](n, m Number) Number {
	if m == 0 {
		return 1
	}
	result := n
	for i := Number(2); i <= m; i++ {
		result *= n
	}
	return result
}

type numberKind uint

const (
	numberKindInt numberKind = iota
	numberKindUint
	numberKindFloat
)

func makeNumberKind(k reflect.Kind) numberKind {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return numberKindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return numberKindUint
	case reflect.Float32, reflect.Float64:
		return numberKindFloat
	default:
		panic("river/vm: makeNumberKind called with unsupported Kind value")
	}
}
