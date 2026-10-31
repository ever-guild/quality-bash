package quality

import (
	"strings"
	"testing"
)

func TestTAPClassification(t *testing.T) {
	cases := []struct {
		text            string
		exit            int
		failed, invalid bool
	}{
		{"1..2\nok 1 works\nnot ok 2 broken\n", 1, true, false},
		{"1..1\nok 1 works\n", 0, false, false},
		{"cannot start Bats", 1, false, true},
		{"1..2\nnot ok 1 broken\n", 1, false, true},
		{"1..0\n", 0, false, true},
		{"1..1\nok 1 works\n", 1, false, true},
		{"1..1\nBail out!\n", 1, false, true},
	}
	for _, tc := range cases {
		failed, err := ValidateTAP(strings.NewReader(tc.text), tc.exit)
		if (err != nil) != tc.invalid || (!tc.invalid && failed != tc.failed) {
			t.Fatalf("%q: failed=%t err=%v", tc.text, failed, err)
		}
	}
}
