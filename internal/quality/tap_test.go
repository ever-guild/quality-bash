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

func TestAbortTAP(t *testing.T) {
	for _, tc := range []struct {
		text  string
		exit  int
		valid bool
	}{
		{"1..2\nnot ok 1 failed\n# bats warning: Executed 1 instead of expected 2 tests\n", 1, true},
		{"1..2\nok 1 works\nok 2 works\n", 0, true},
		{"1..2\nnot ok 1 failed\n", 1, false},
		{"1..2\nok 1 works\n# bats warning: Executed 1 instead of expected 2 tests\n", 1, false},
		{"1..2\nnot ok 1 failed\n# bats warning: Executed 2 instead of expected 2 tests\n", 1, false},
		{"1..2\nnot ok 1 failed\n# bats warning: Executed 1 instead of expected 2 tests\nnot ok 2 teardown\n", 1, false},
		{"1..1\nnot ok 1 failed\n", 130, false},
	} {
		_, err := validateTAP(strings.NewReader(tc.text), tc.exit, true)
		if (err == nil) != tc.valid {
			t.Fatalf("%q: %v", tc.text, err)
		}
	}
}

func TestAbortRejectsMalformedResults(t *testing.T) {
	for _, text := range []string{
		"1..1\nnot ok 1 failed\ngarbage\n",
		"1..3\nnot ok 1 failed\nok 2 passed\n# bats warning: Executed 2 instead of expected 3 tests\n",
		"1..1\nnot ok 1 failed # timeout after 1s\n",
	} {
		if _, err := validateTAP(strings.NewReader(text), 1, true); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}
