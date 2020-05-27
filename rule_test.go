package boat

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInvalidRules(t *testing.T) {
	cases := []string{
		`"hello" ++`,
		`123 + "hello world"`,
		`"test" - 3`,
		`"test" / 3`,
		`>="test"`,
		`123 -+ 4`,
		`"hello world`,
		`0xfg`,
	}

	for _, test := range cases {
		px, err := ParseRule(test)
		if err != nil {
			continue
		}
		_, err = px.Eval("invalid")
		require.Error(t, err)
	}
}

func TestRules(t *testing.T) {
	cases := []struct {
		in   string
		rule string
		pass bool
	}{
		{in: "hello world", rule: `123 | "hello " + "world"`, pass: true},
		{in: "100", rule: `>=100 & <=100`, pass: true},
		{in: "100", rule: `>100`, pass: false},
		{in: "50", rule: ">=100/2 & <100", pass: true},
		{in: "49", rule: ">=100/2 & <100", pass: false},
		{in: "7", rule: "<(1+2)*3", pass: true},
		{in: "7", rule: "<1+2*3", pass: false},
		{in: "8", rule: "<(1+2)*3", pass: true},
		{in: "9", rule: "<(1+2)*3", pass: false},
		{in: "1", rule: "!(>=1 & <=400 | >=500 & <=600)", pass: false},
		{in: "0", rule: "!(>=1 & <=400 | >=500 & <=600)", pass: true},
		{in: "hehe", rule: `"he" * 3`, pass: false},
		{in: "hehehe", rule: `"he" * 3`, pass: true},
		{in: "hello\nworld\test", rule: `"hello\nworld\test"`, pass: true},
		{in: "\377 test \u2847 \xff", rule: `"\377 test \u2847 \xff"`, pass: true},
	}

	for _, test := range cases {
		px, err := ParseRule(test.rule)
		require.NoError(t, err)

		pass, err := px.Eval(test.in)
		require.NoError(t, err)
		require.EqualValues(t, pass, test.pass, test)

		px, err = ParseRuleBytes([]byte(test.rule))
		require.NoError(t, err)

		pass, err = px.Eval(test.in)
		require.NoError(t, err)
		require.EqualValues(t, pass, test.pass, test)
	}
}

func BenchmarkRule(b *testing.B) {
	px, err := ParseRule(`123 +456 |  "hello "`)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pass, err := px.Eval(`579`)
		if !pass || err != nil {
			b.Fatal(err)
		}
	}
}
