package boat

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"strconv"
)

var Rules = [...]struct {
	prec int  // precedence
	rtl  bool // right-associative?
}{
	tokNegate: {prec: 4, rtl: true},
	tokBang:   {prec: 4, rtl: true},
	tokGT:     {prec: 4, rtl: true},
	tokGTE:    {prec: 4, rtl: true},
	tokLT:     {prec: 4, rtl: true},
	tokLTE:    {prec: 4, rtl: true},

	tokPlus:  {prec: 3},
	tokMinus: {prec: 3},

	tokAND: {prec: 2},
	tokOR:  {prec: 1},
}

type Rule struct {
	rule string  // rule
	ops  []Token // stack of ops
	vals []Node  // stack of vals
}

func NewRule(rule string) Rule {
	return Rule{rule: rule, ops: make([]Token, 0, 16), vals: make([]Node, 0, 16)}
}

func (e *Rule) Eval(input string) (bool, error) {
	in, err := Decode(input)
	if err != nil {
		return false, err
	}

	e.ops = e.ops[:0]
	e.vals = e.vals[:0]

	m := NewMachine(e.rule)

	c := m.Lex()
	l := c

	for c.Type != tokEOF {
		switch c.Type {
		case tokInt:
			val, err := strconv.ParseInt(c.repr(m.in), 0, 64)
			if err != nil {
				return false, fmt.Errorf("failed to decode int: %w", err)
			}
			e.vals = append(e.vals, Node{Type: nodeInt, Int: val})
		case tokFloat:
			val, err := strconv.ParseFloat(c.repr(m.in), 64)
			if err != nil {
				return false, fmt.Errorf("failed to decode float: %w", err)
			}
			e.vals = append(e.vals, Node{Type: nodeFloat, Float: val})
		case tokText:
			e.vals = append(e.vals, Node{Type: nodeText, Text: c.repr(m.in)})
		case tokBracketStart:
			e.ops = append(e.ops, c) // TODO(kenta): handle bracket start and close
		case tokGT, tokGTE, tokLT, tokLTE, tokBang, tokAND, tokOR, tokPlus, tokMinus:
			if c.Type == tokMinus && l.Type != tokInt && l.Type != tokFloat && l.Type != tokText {
				c.Type = tokNegate
			}

			for len(e.ops) > 0 {
				op := e.ops[len(e.ops)-1]

				o1 := Rules[c.Type]
				o2 := Rules[op.Type]

				if op.Type == tokBracketStart || o1.prec > o2.prec || o1.prec == o2.prec && o1.rtl {
					break
				}

				e.ops = e.ops[:len(e.ops)-1]

				if err := e.EvalOP(in, op); err != nil {
					return false, fmt.Errorf("error while evaluating op: %w", err)
				}
			}
			e.ops = append(e.ops, c)
		}

		l = c
		c = m.Lex()
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

	if len(e.vals) > 1 {
		spew.Dump(e.vals) // TODO(kenta): handle case when multiple values exist
	}

	return e.vals[0].Bool, nil
}

func (e *Rule) EvalNode(in, n Node) bool {
	switch n.Type {
	case nodeInt:
		switch in.Type {
		case nodeInt:
			return in.Int == n.Int
		case nodeFloat:
			return in.Float == float64(n.Int)
		default:
			return false
		}
	case nodeFloat:
		switch in.Type {
		case nodeFloat:
			return in.Float == n.Float
		case nodeInt:
			return float64(in.Int) == n.Float
		default:
			return false
		}
	case nodeText:
		return in.Type == nodeText && in.Text == n.Text
	default:
		return n.Bool
	}
}

func (e *Rule) EvalOP(in Node, op Token) error {
	//fmt.Printf("EVAL %q\n", op.repr(m.in))

	switch op.Type {
	case tokNegate:
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
		l := len(e.vals) - 2
		r := len(e.vals) - 1
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
				e.vals[l] = Node{Type: nodeText, Text: e.vals[l].Text + e.vals[r].Text}
			default:
				return errors.New(`lhs is string, rhs for '+' must be a string`)
			}
		default:
			return errors.New("lhs and rhs for '+' must be int or float")
		}
		e.vals = e.vals[:r]
	case tokMinus:
		l := len(e.vals) - 2
		r := len(e.vals) - 1
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
	case tokBang:
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
		l := len(e.vals) - 2
		r := len(e.vals) - 1

		e.vals[l] = Node{Type: nodeBool, Bool: e.EvalNode(in, e.vals[l]) && e.EvalNode(in, e.vals[r])}
		e.vals = e.vals[:r]
	case tokOR:
		l := len(e.vals) - 2
		r := len(e.vals) - 1

		e.vals[l] = Node{Type: nodeBool, Bool: e.EvalNode(in, e.vals[l]) || e.EvalNode(in, e.vals[r])}
		e.vals = e.vals[:r]
	}

	return nil
}
