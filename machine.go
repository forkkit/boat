package boat

import (
	"fmt"
	"unicode/utf8"
)

type Machine struct {
	in  string // input
	bc  int    // byte count
	cc  int    // char count
	lcw int    // last char width
}

func NewMachine(in string) Machine {
	return Machine{in: in, lcw: -1}
}

func (m *Machine) Advance() rune {
	if m.bc >= len(m.in) {
		return eof
	}
	r, cw := utf8.DecodeRuneInString(m.in[m.bc:])
	m.bc += cw
	m.lcw = cw
	m.cc++
	return r
}

func (m *Machine) Backup() {
	if m.lcw < 0 {
		panic("went back too far")
	}
	m.bc -= m.lcw
	m.lcw = -1
	m.cc--
}

func (m *Machine) Lex() Token {
	r := m.Advance()
	for isWhitespace(r) {
		r = m.Advance()
	}
	if r == eof {
		return Token{Type: tokEOF, Start: m.bc - 1, End: m.bc}
	}

	if isDecimalRune(r) || r == '.' {
		s := m.bc - 1

		if m.LexNumber(r) {
			return Token{Type: tokFloat, Start: s, End: m.bc}
		}
		return Token{Type: tokInt, Start: s, End: m.bc}
	}

	switch r {
	case '\'', '"':
		return m.LexText(r)
	case '>':
		r = m.Advance()
		if r == '=' {
			return Token{Type: tokGTE, Start: m.bc - 2, End: m.bc}
		} else {
			m.Backup()
			return Token{Type: tokGT, Start: m.bc - 1, End: m.bc}
		}
	case '<':
		r = m.Advance()
		if r == '=' {
			return Token{Type: tokLTE, Start: m.bc - 2, End: m.bc}
		} else {
			m.Backup()
			return Token{Type: tokLT, Start: m.bc - 1, End: m.bc}
		}
	case '!':
		return Token{Type: tokBang, Start: m.bc - 1, End: m.bc}
	case '+':
		return Token{Type: tokPlus, Start: m.bc - 1, End: m.bc}
	case '-':
		return Token{Type: tokMinus, Start: m.bc - 1, End: m.bc}
	case '*':
		return Token{Type: tokMultiply, Start: m.bc - 1, End: m.bc}
	case '/':
		return Token{Type: tokDivide, Start: m.bc - 1, End: m.bc}
	case '(':
		return Token{Type: tokBracketStart, Start: m.bc - 1, End: m.bc}
	case ')':
		return Token{Type: tokBracketEnd, Start: m.bc - 1, End: m.bc}
	case '&':
		return Token{Type: tokAND, Start: m.bc - 1, End: m.bc}
	case '|':
		return Token{Type: tokOR, Start: m.bc - 1, End: m.bc}
	}
	panic(fmt.Sprintf("unknown rune %c", r))
}

func (m *Machine) LexNumber(r rune) (float bool) {
	var (
		separator bool
		digit     bool
		prefix    rune
	)

	float = r == '.'

	skip := func(pred func(rune) bool) {
		for {
			switch {
			case r == '_':
				separator = true
				r = m.Advance()
				continue
			case pred(r):
				digit = true
				r = m.Advance()
				continue
			default:
				m.Backup()
			case r == eof:
			}
			break
		}
	}

	if r == '0' {
		prefix = lower(m.Advance())

		switch prefix {
		case 'x':
			r = m.Advance()
			skip(isHexRune)
		case 'o':
			r = m.Advance()
			skip(isOctalRune)
		case 'b':
			r = m.Advance()
			skip(isBinRune)
		default:
			prefix, digit = '0', true
			skip(isOctalRune)
		}
	} else {
		skip(isDecimalRune)
	}

	if !float {
		float = r == '.'
	}

	if float {
		if prefix == 'o' || prefix == 'b' {
			panic("invalid radix point")
		}

		r = lower(m.Advance())
		r = lower(m.Advance())

		switch prefix {
		case 'x':
			skip(isHexRune)
		case '0':
			skip(isOctalRune)
		default:
			skip(isDecimalRune)
		}
	}

	if !digit {
		panic("number has no digits")
	}

	e := lower(r)

	if e == 'e' || e == 'p' {
		if e == 'e' && prefix != eof && prefix != '0' {
			panic(`'e' exponent requires decimal mantissa`)
		}
		if e == 'p' && prefix != 'x' {
			panic(`'p' exponent requires hexadecimal mantissa`)
		}

		r = m.Advance()
		r = m.Advance()
		if r == '+' || r == '-' {
			r = m.Advance()
		}

		float = true

		skip(isDecimalRune)

		if !digit {
			panic("exponent has no digits")
		}
	} else if float && prefix == 'x' {
		panic("hexadecimal mantissa requires a 'p' exponent")
	}

	_ = separator

	return float
}

func (m *Machine) LexText(quote rune) Token {
	start := m.bc

	for {
		switch m.Advance() {
		case quote:
			return Token{Type: tokText, Start: start, End: m.bc - 1}
		case '\\':
			m.LexEscape(quote)
			continue
		case eof, '\n':
			panic("unterminated")
		default:
			continue
		}
	}
}

func (m *Machine) LexEscape(quote rune) {
	r := m.Advance()

	skip := func(n int, pred func(rune) bool) {
		for n > 0 {
			r = m.Advance()
			if !pred(r) || r == eof {
				panic("bad")
			}
			n--
		}
	}

	switch r {
	case quote, 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
		// ignore
	case 'x':
		skip(2, isHexRune)
	case 'u':
		skip(4, isHexRune)
	case 'U':
		skip(8, isHexRune)
	case eof:
		panic("got eof while parsing escape literal")
	default:
		if !isOctalRune(r) || r == eof {
			panic("bad 8")
		}
		skip(2, isOctalRune)
	}
}
