package quality

import (
	"math"
	"testing"
)

func TestRatios(t *testing.T) {
	for _, tc := range []struct {
		a, b Ratio
		want bool
	}{
		{Ratio{1, 3}, Ratio{33, 100}, true}, {Ratio{33, 100}, Ratio{1, 3}, false},
		{Ratio{math.MaxUint64 - 1, math.MaxUint64}, Ratio{math.MaxUint64 - 2, math.MaxUint64 - 1}, true},
		{Ratio{0, 0}, Ratio{0, 1}, false},
	} {
		if got := tc.a.AtLeast(tc.b); got != tc.want {
			t.Fatalf("%v >= %v: %v", tc.a, tc.b, got)
		}
	}
}
func TestIndependentGates(t *testing.T) {
	base := Baseline{Method: validBaseline(t).Method, Files: []string{"sample.sh"}, Coverage: Ratio{8, 10}, Mutation: Ratio{6, 10}}
	for _, coverage := range []Ratio{{7, 10}, {9, 10}} {
		for _, mutation := range []Ratio{{5, 10}, {7, 10}} {
			current := base
			current.Coverage = coverage
			current.Mutation = mutation
			want := coverage.Numerator >= 8 && mutation.Numerator >= 6
			if (base.Check(current) == nil) != want {
				t.Fatalf("incorrect gate: %+v", current)
			}
		}
	}
	current := base
	current.Files = []string{"other.sh"}
	if base.Check(current) == nil {
		t.Fatal("scope change accepted")
	}
	current = base
	current.Method.Mutation.Algorithm = "new"
	if base.Check(current) == nil {
		t.Fatal("method change accepted")
	}
}
