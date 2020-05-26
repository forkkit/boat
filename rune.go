package boat

const whitespace = uint64(1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' ')

func isWhitespace(r rune) bool {
	return whitespace&(1<<uint(r)) != 0
}

func isBinRune(r rune) bool {
	return r == '0' || r == '1'
}

func isOctalRune(r rune) bool {
	return r >= '0' && r <= '7'
}

func isDecimalRune(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexRune(r rune) bool {
	if isDecimalRune(r) {
		return true
	} else {
		r = lower(r)
	}
	return r >= 'a' && r <= 'f'
}

func lower(r rune) rune {
	return ('a' - 'A') | r
}
