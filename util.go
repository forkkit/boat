package boat

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

func unescape(s string) (string, error) {
	if !strings.ContainsRune(s, '\\') && utf8.ValidString(s) {
		return s, nil
	}
	tmp := make([]byte, utf8.UTFMax)
	buf := make([]byte, 0, 3*len(s)/2)
	for len(s) > 0 {
		c, mb, t, err := strconv.UnquoteChar(s, '"')
		if err != nil {
			return "", err
		}
		s = t
		if c < utf8.RuneSelf || !mb {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(tmp[:], c)
			buf = append(buf, tmp[:n]...)
		}
	}
	return string(buf), nil
}
