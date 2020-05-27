package boat

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMachine(t *testing.T) {
	cases := []string{
		`"hello" + "world"`,
		`0xff 0xfd 1234.0e5 .196 123`,
		`!(>=1 & <=400 | >=500 & <=600)`,
	}

	for _, test := range cases {
		m := NewMachine(test)

		tok := m.Next()
		for tok.Type != tokEOF && tok.Type != tokError {
			tok = m.Next()
		}

		require.NotEqual(t, tok.Type, tokError)
	}
}
