package boat

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type NodeType int

const (
	nodeBool NodeType = iota
	nodeInt
	nodeFloat
	nodeText
)

var nodeStr = [...]string{
	nodeBool:  "bool",
	nodeInt:   "int",
	nodeFloat: "float",
	nodeText:  "text",
}

func (t NodeType) String() string {
	return nodeStr[t]
}

type Node struct {
	Type  NodeType
	Bool  bool
	Int   int64
	Float float64
	Text  string
}

func Decode(val string) (Node, error) {
	var n Node

	r, _ := utf8.DecodeRuneInString(val)

	switch {
	case r == '.' || r == '-' || isDecimalRune(r):
		if strings.ContainsRune(val, '.') {
			n.Type = nodeFloat
			val, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return n, fmt.Errorf("failed to decode float: %w", err)
			}
			n.Float = val
		} else {
			n.Type = nodeInt
			val, err := strconv.ParseInt(val, 0, 64)
			if err != nil {
				return n, fmt.Errorf("failed to decode int: %w", err)
			}
			n.Int = val
		}
	default:
		n.Type = nodeText
		n.Text = val
	}
	return n, nil
}

func EvalNode(a, b Node) bool {
	switch b.Type {
	case nodeInt:
		switch a.Type {
		case nodeInt:
			return a.Int == b.Int
		case nodeFloat:
			return a.Float == float64(b.Int)
		default:
			return false
		}
	case nodeFloat:
		switch a.Type {
		case nodeFloat:
			return a.Float == b.Float
		case nodeInt:
			return float64(a.Int) == b.Float
		default:
			return false
		}
	case nodeText:
		return a.Type == nodeText && a.Text == b.Text
	default:
		return b.Bool
	}
}
