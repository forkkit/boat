package boat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

var Ops = [...]struct {
	prec int  // precedence
	rtl  bool // right-associative?
}{
	tokNegate: {prec: 6, rtl: true},

	tokMultiply: {prec: 5},
	tokDivide:   {prec: 5},

	tokPlus:  {prec: 4},
	tokMinus: {prec: 4},

	tokBang: {prec: 3, rtl: true},
	tokGT:   {prec: 3, rtl: true},
	tokGTE:  {prec: 3, rtl: true},
	tokLT:   {prec: 3, rtl: true},
	tokLTE:  {prec: 3, rtl: true},

	tokAND: {prec: 2},
	tokOR:  {prec: 1},
}

type Rule struct {
	rule string  // rule
	buf  []Token // tokens
	ops  []Token // stack of ops
	vals []Node  // stack of vals
}

func ParseRuleBytes(buf []byte) (Rule, error) {
	return ParseRule(*(*string)(unsafe.Pointer(&buf)))
}

func ParseRule(rule string) (Rule, error) {
	r := Rule{rule: rule, ops: make([]Token, 0, 16), vals: make([]Node, 0, 16)}

	m := NewMachine(rule)

	tok := m.Next()
	for tok.Type != tokEOF && tok.Type != tokError {
		r.buf = append(r.buf, tok)
		tok = m.Next()
	}

	if tok.Type == tokError {
		return r, fmt.Errorf("%d:%d error parsing rule: %s", tok.Start, tok.End, m.err)
	}

	return r, nil
}

func (e *Rule) Eval(input string) (bool, error) {
	in, err := Decode(input)
	if err != nil {
		return false, err
	}

	e.ops = e.ops[:0]
	e.vals = e.vals[:0]

	for i := 0; i < len(e.buf); i++ {
		c := e.buf[i]
		switch c.Type {
		case tokInt:
			val, err := strconv.ParseInt(c.repr(e.rule), 0, 64)
			if err != nil {
				return false, fmt.Errorf("failed to decode int: %w", err)
			}
			e.vals = append(e.vals, Node{Type: nodeInt, Int: val})
		case tokFloat:
			val, err := strconv.ParseFloat(c.repr(e.rule), 64)
			if err != nil {
				return false, fmt.Errorf("failed to decode float: %w", err)
			}
			e.vals = append(e.vals, Node{Type: nodeFloat, Float: val})
		case tokText:
			val, err := unescape(c.repr(e.rule))
			if err != nil {
				return false, fmt.Errorf("failed to unescape string: %w", err)
			}
			e.vals = append(e.vals, Node{Type: nodeText, Text: val})
		case tokBracketStart:
			e.ops = append(e.ops, c)
		case tokBracketEnd:
			for len(e.ops) > 0 {
				op := e.ops[len(e.ops)-1]
				e.ops = e.ops[:len(e.ops)-1]

				if op.Type == tokBracketStart {
					break
				}

				if err := e.EvalOP(in, op); err != nil {
					return false, fmt.Errorf("error while evaluating op input brackets: %w", err)
				}
			}
		case tokGT, tokGTE, tokLT, tokLTE, tokBang, tokAND, tokOR, tokPlus, tokMinus, tokMultiply, tokDivide:
			if c.Type == tokMinus {
				if i == 0 {
					c.Type = tokNegate
				} else {
					l := e.buf[i-1]
					if l.Type != tokInt && l.Type != tokFloat && l.Type != tokText {
						c.Type = tokNegate
					}
				}
			}

			for len(e.ops) > 0 {
				op := e.ops[len(e.ops)-1]

				if op.Type == tokBracketStart {
					break
				}

				o1 := Ops[c.Type]
				o2 := Ops[op.Type]

				if o1.prec > o2.prec || o1.prec == o2.prec && o1.rtl {
					break
				}

				e.ops = e.ops[:len(e.ops)-1]

				if err := e.EvalOP(in, op); err != nil {
					return false, fmt.Errorf("error while evaluating op: %w", err)
				}
			}
			e.ops = append(e.ops, c)
		}
	}

	for len(e.ops) > 0 {
		op := e.ops[len(e.ops)-1]
		e.ops = e.ops[:len(e.ops)-1]

		if op.Type == tokBracketStart {
			return false, errors.New("mismatched parenthesis")
		}

		if err := e.EvalOP(in, op); err != nil {
			return false, fmt.Errorf("error while evaluating op: %w", err)
		}
	}

	if len(e.vals) != 1 {
		return false, fmt.Errorf("got %d values from evaluating the rule: expected only one", len(e.vals))
	}

	return EvalNode(in, e.vals[0]), nil
}

func (e *Rule) EvalOP(in Node, op Token) error {
	//fmt.Printf("EVAL %q\n", op.repr(e.rule))

	switch op.Type {
	case tokNegate:
		if len(e.vals) < 1 {
			return errors.New(`unary '-' must have a rhs that is an int or float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeInt:
			e.vals[i].Int = -e.vals[i].Int
		case nodeFloat:
			e.vals[i].Float = -e.vals[i].Float
		default:
			return errors.New(`unary '-' not paired with int or float`)
		}
	case tokGT:
		if len(e.vals) < 1 {
			return errors.New(`'>' must have a rhs that is an int or float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeInt:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Int > e.vals[i].Int}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float > float64(e.vals[i].Int)}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		case nodeFloat:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: float64(in.Int) > e.vals[i].Float}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float > e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		default:
			return errors.New(`'>' not paired with int or float`)
		}
	case tokLT:
		if len(e.vals) < 1 {
			return errors.New(`'<' must have a rhs that is an int or float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeInt:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Int < e.vals[i].Int}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float < float64(e.vals[i].Int)}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		case nodeFloat:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: float64(in.Int) < e.vals[i].Float}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float < e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		default:
			return errors.New(`'<' not paired with int or float`)
		}
	case tokGTE:
		if len(e.vals) < 1 {
			return errors.New(`'>=' must have a rhs that is an int or float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeInt:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Int >= e.vals[i].Int}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float >= float64(e.vals[i].Int)}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		case nodeFloat:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: float64(in.Int) >= e.vals[i].Float}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float >= e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		default:
			return errors.New(`'>=' not paired with int or float`)
		}
	case tokLTE:
		if len(e.vals) < 1 {
			return errors.New(`'<=' must have a rhs that is an int or float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeInt:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Int <= e.vals[i].Int}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: float64(in.Int) <= e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		case nodeFloat:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float <= float64(e.vals[i].Int)}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float <= e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: false}
			}
		default:
			return errors.New(`'<=' not paired with int or float`)
		}
	case tokPlus:
		if len(e.vals) < 2 {
			return errors.New(`'+' requires a lhs and rhs that is an string/int/float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		switch e.vals[l].Type {
		case nodeInt:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeInt, Int: e.vals[l].Int + e.vals[r].Int}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: float64(e.vals[l].Int) + e.vals[r].Float}
			default:
				return errors.New(`lhs is int, rhs for '+' must be an int or float`)
			}
		case nodeFloat:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float + float64(e.vals[r].Int)}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float + e.vals[r].Float}
			default:
				return errors.New(`lhs is float, rhs for '+' must be an int or float`)
			}
		case nodeText:
			switch e.vals[r].Type {
			case nodeText:
				var b strings.Builder
				b.Grow(len(e.vals[l].Text) + len(e.vals[r].Text))
				b.WriteString(e.vals[l].Text)
				b.WriteString(e.vals[r].Text)
				e.vals[l] = Node{Type: nodeText, Text: b.String()}
			default:
				return errors.New(`lhs is string, rhs for '+' must be a string`)
			}
		default:
			return errors.New("lhs and rhs for '+' must be int or float")
		}
		e.vals = e.vals[:r]
	case tokMinus:
		if len(e.vals) < 2 {
			return errors.New(`'-' requires a lhs and rhs that is an int or float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		switch e.vals[l].Type {
		case nodeInt:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeInt, Int: e.vals[l].Int - e.vals[r].Int}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: float64(e.vals[l].Int) - e.vals[r].Float}
			default:
				return errors.New(`lhs is int, rhs for '-' must be an int or float`)
			}
		case nodeFloat:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float - float64(e.vals[r].Int)}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float - e.vals[r].Float}
			default:
				return errors.New(`lhs is float, rhs for '-' must be an int or float`)
			}
		default:
			return errors.New(`lhs and rhs for '-' must be int or float`)
		}
		e.vals = e.vals[:r]
	case tokMultiply:
		if len(e.vals) < 2 {
			return errors.New(`'*' requires a lhs that is an string/int/float, and a rhs that is an int/float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		switch e.vals[l].Type {
		case nodeInt:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeInt, Int: e.vals[l].Int * e.vals[r].Int}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: float64(e.vals[l].Int) * e.vals[r].Float}
			default:
				return errors.New(`lhs is int, rhs for '*' must be an int or float`)
			}
		case nodeFloat:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float * float64(e.vals[r].Int)}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float * e.vals[r].Float}
			default:
				return errors.New(`lhs is float, rhs for '*' must be an int or float`)
			}
		case nodeText:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeText, Text: strings.Repeat(e.vals[l].Text, int(e.vals[r].Int))}
			default:
				return errors.New(`lhs is string, rhs for '*' must be an int`)
			}
		default:
			return errors.New(`lhs and rhs for '*' must be int or float or string`)
		}
		e.vals = e.vals[:r]
	case tokDivide:
		if len(e.vals) < 2 {
			return errors.New(`'/' requires a lhs and rhs that is an int or float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		switch e.vals[l].Type {
		case nodeInt:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeInt, Int: e.vals[l].Int / e.vals[r].Int}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: float64(e.vals[l].Int) / e.vals[r].Float}
			default:
				return errors.New(`lhs is int, rhs for '/' must be an int or float`)
			}
		case nodeFloat:
			switch e.vals[r].Type {
			case nodeInt:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float / float64(e.vals[r].Int)}
			case nodeFloat:
				e.vals[l] = Node{Type: nodeFloat, Float: e.vals[l].Float / e.vals[r].Float}
			default:
				return errors.New(`lhs is float, rhs for '/' must be an int or float`)
			}
		default:
			return errors.New(`lhs and rhs for '/' must be int or float`)
		}
		e.vals = e.vals[:r]
	case tokBang:
		if len(e.vals) < 1 {
			return errors.New(`'!' requires a rhs that is a string/bool/int/float`)
		}
		i := len(e.vals) - 1
		switch e.vals[i].Type {
		case nodeText:
			switch in.Type {
			case nodeText:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Text != e.vals[i].Text}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: true}
			}
		case nodeBool:
			e.vals[i] = Node{Type: nodeBool, Bool: !e.vals[i].Bool}
		case nodeInt:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Int != e.vals[i].Int}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float != float64(e.vals[i].Int)}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: true}
			}
		case nodeFloat:
			switch in.Type {
			case nodeInt:
				e.vals[i] = Node{Type: nodeBool, Bool: float64(in.Int) != e.vals[i].Float}
			case nodeFloat:
				e.vals[i] = Node{Type: nodeBool, Bool: in.Float != e.vals[i].Float}
			default:
				e.vals[i] = Node{Type: nodeBool, Bool: true}
			}
		}
	case tokAND:
		if len(e.vals) < 2 {
			return errors.New(`'&' requires a lhs and rhs that is a string/bool/int/float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		e.vals[l] = Node{Type: nodeBool, Bool: EvalNode(in, e.vals[l]) && EvalNode(in, e.vals[r])}
		e.vals = e.vals[:r]
	case tokOR:
		if len(e.vals) < 2 {
			return errors.New(`'|' requires a lhs and rhs that is a string/bool/int/float`)
		}
		l := len(e.vals) - 2
		r := l + 1
		e.vals[l] = Node{Type: nodeBool, Bool: EvalNode(in, e.vals[l]) || EvalNode(in, e.vals[r])}
		e.vals = e.vals[:r]
	}

	return nil
}
