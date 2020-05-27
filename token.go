package boat

const eof rune = 0

type TokenType int

const (
	tokError TokenType = iota
	tokEOF
	tokGT
	tokGTE
	tokLT
	tokLTE
	tokBang
	tokAND
	tokOR
	tokPlus
	tokMinus
	tokMultiply
	tokDivide
	tokNegate
	tokText
	tokInt
	tokFloat
	tokBracketStart
	tokBracketEnd
)

var tokStr = [...]string{
	tokEOF:          "eof",
	tokGT:           ">",
	tokGTE:          ">=",
	tokLT:           "<",
	tokLTE:          "<=",
	tokBang:         "!",
	tokAND:          "&",
	tokOR:           "|",
	tokPlus:         "+",
	tokMinus:        "-",
	tokMultiply:     "*",
	tokDivide:       "/",
	tokNegate:       "-",
	tokText:         "text",
	tokInt:          "int",
	tokFloat:        "float",
	tokBracketStart: "(",
	tokBracketEnd:   ")",
}

func (t TokenType) String() string {
	return tokStr[t]
}

type Token struct {
	Type  TokenType // token type
	Start int       // token start index
	End   int       // token end index
}

func (t Token) repr(input string) string {
	return input[t.Start:t.End]
}
