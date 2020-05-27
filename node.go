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
