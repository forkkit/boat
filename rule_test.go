package boat

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRule(t *testing.T) {
	cases := []struct {
		in   string
		rule string
		pass bool
	}{
		{in: "hello world", rule: `123 | "hello " + "world"`, pass: true},
		{in: "100", rule: `>=100 & <=100`, pass: true},
		{in: "100", rule: `>100`, pass: false},
	}

	for _, test := range cases {
		px := NewRule(test.rule)

		pass, err := px.Eval(test.in)
		require.NoError(t, err)
		require.EqualValues(t, pass, test.pass)
	}
}

func BenchmarkRule(b *testing.B) {
	px := NewRule(`123 +456 |  "hello " + "world"`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pass, err := px.Eval(`579`)
		if !pass || err != nil {
			b.Fatal(err)
		}
	}
}
