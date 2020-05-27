package boat

import (
	"unicode/utf8"
)

type Machine struct {
	input string  // input
	err   string  // error
	buf   []Token // token buf
	pos   int     // start pos (byte)
	ptr   int     // end pos (byte)
	cc    int     // end pos (char)
	lcw   int     // last char width
}

func NewMachine(input string) Machine {
	return Machine{input: input, buf: make([]Token, 0, 16), lcw: -1}
}

func (m *Machine) next() rune {
	if m.ptr >= len(m.input) {
		if m.ptr > len(m.input) {
			m.error("went too far ahead")
			return eof
		}
		return eof
	}
	r, cw := utf8.DecodeRuneInString(m.input[m.ptr:])
	m.ptr += cw
	m.lcw = cw
	m.cc++
	return r
}

func (m *Machine) backup() {
	if m.lcw < 0 {
		m.error("went back too far")
	}
	m.ptr -= m.lcw
	m.lcw = -1
	m.cc--
}

func (m *Machine) emit(typ TokenType) {
	m.buf = append(m.buf, Token{Type: typ, Start: m.pos, End: m.ptr})
	m.ignore()
}

func (m *Machine) error(err string) {
	m.buf = append(m.buf, Token{Type: tokError, Start: m.pos, End: m.ptr})
	m.err = err
}

func (m *Machine) ignore() {
	m.pos = m.ptr
}

func (m *Machine) Next() Token {
	for {
		if len(m.buf) > 0 {
			token := m.buf[0]
			m.buf = m.buf[1:]
			return token
		}

		r := m.next()
		for isWhitespace(r) {
			m.ignore()
			r = m.next()
		}

		if r == eof {
			m.emit(tokEOF)
			continue
		}

		if isDecimalRune(r) || r == '.' {
			m.lexNumber(r)
			continue
		}

		switch r {
		case '\'', '"':
			m.lexEscapedText(r)
		case '>':
			r = m.next()
			if r == '=' {
				m.emit(tokGTE)
			} else {
				m.backup()
				m.emit(tokGT)
			}
		case '<':
			r = m.next()
			if r == '=' {
				m.emit(tokLTE)
			} else {
				m.backup()
				m.emit(tokLT)
			}
		case '!':
			m.emit(tokBang)
		case '+':
			m.emit(tokPlus)
		case '-':
			m.emit(tokMinus)
		case '*':
			m.emit(tokMultiply)
		case '/':
			m.emit(tokDivide)
		case '(':
			m.emit(tokBracketStart)
		case ')':
			m.emit(tokBracketEnd)
		case '&':
			m.emit(tokAND)
		case '|':
			m.emit(tokOR)
		default:
			m.error("unexpected rune")
		}
	}
}

func (m *Machine) lexNumber(r rune) {
	var (
		separator bool
		digit     bool
		prefix    rune
	)

	float := r == '.'

	skip := func(pred func(rune) bool) {
		for {
			switch {
			case r == '_':
				separator = true
				r = m.next()
				continue
			case pred(r):
				digit = true
				r = m.next()
				continue
			default:
				m.backup()
			case r == eof:
			}
			break
		}
	}

	if r == '0' {
		prefix = lower(m.next())

		switch prefix {
		case 'x':
			r = m.next()
			skip(isHexRune)
		case 'o':
			r = m.next()
			skip(isOctalRune)
		case 'b':
			r = m.next()
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
			m.error("invalid radix point")
			return
		}

		r = lower(m.next())
		r = lower(m.next())

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
		m.error("number has no digits")
		return
	}

	e := lower(r)

	if e == 'e' || e == 'p' {
		if e == 'e' && prefix != eof && prefix != '0' {
			m.error(`'e' exponent requires decimal mantissa`)
			return
		}
		if e == 'p' && prefix != 'x' {
			m.error(`'p' exponent requires hexadecimal mantissa`)
			return
		}

		r = m.next()
		r = m.next()
		if r == '+' || r == '-' {
			r = m.next()
		}

		float = true

		skip(isDecimalRune)

		if !digit {
			m.error("exponent has no digits")
			return
		}
	} else if float && prefix == 'x' {
		m.error("hexadecimal mantissa requires a 'p' exponent")
		return
	}

	_ = separator

	if float {
		m.emit(tokFloat)
	} else {
		m.emit(tokInt)
	}
}

func (m *Machine) lexEscapedText(quote rune) {
	m.ignore()

	for len(m.buf) == 0 {
		switch m.next() {
		case quote:
			m.backup()
			m.emit(tokText)
			m.next()
			m.ignore()
			return
		case '\\':
			m.lexEscape(quote)
			continue
		case eof, '\n':
			m.error("unterminated string literal")
			return
		default:
			continue
		}
	}
}

func (m *Machine) lexEscape(quote rune) {
	r := m.next()

	skip := func(n int, pred func(rune) bool) {
		for n > 0 {
			r = m.next()
			if !pred(r) || r == eof {
				m.error("got invalid escape sequence literal")
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
		m.error("reached eof while parsing escape sequence literal")
	default:
		if !isOctalRune(r) || r == eof {
			m.error("got invalid escape sequence literal")
		}
		skip(2, isOctalRune)
	}
}
