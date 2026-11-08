package quality

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineUpdatePolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quality-baseline.json")
	m := Measurement{Fingerprint: "current", Qualified: true, Baseline: Baseline{Method: validBaseline(t).Method, Files: []string{"sample.sh"}, Coverage: Ratio{8, 10}, Mutation: Ratio{6, 10}}}
	m.CoverageCounts = m.Baseline.Coverage
	m.MutationCounts = m.Baseline.Mutation
	if UpdateBaseline(path, m, "current", false) == nil {
		t.Fatal("implicit creation accepted")
	}
	if err := UpdateBaseline(path, m, "current", true); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(path)
	for _, kind := range []string{"stale", "coverage", "mutation", "method", "unqualified", "counts"} {
		changed := m
		fingerprint := "current"
		switch kind {
		case "stale":
			fingerprint = "changed"
		case "coverage":
			changed.Baseline.Coverage = Ratio{7, 10}
		case "mutation":
			changed.Baseline.Mutation = Ratio{5, 10}
		case "method":
			changed.Baseline.Method.Coverage.Algorithm = "different"
		case "unqualified":
			changed.Qualified = false
		case "counts":
			changed.MutationCounts = Ratio{1, 10}
		}
		if UpdateBaseline(path, changed, fingerprint, false) == nil {
			t.Fatalf("accepted %s", kind)
		}
		after, _ := os.ReadFile(path)
		if string(after) != string(original) {
			t.Fatalf("changed baseline after %s", kind)
		}
	}
	m.Baseline.Coverage = Ratio{9, 10}
	m.CoverageCounts = m.Baseline.Coverage
	if err := UpdateBaseline(path, m, "current", false); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentBaselineUpdateIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	unlock, err := lockBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	m := Measurement{Fingerprint: "current", Qualified: true, Baseline: Baseline{Method: validBaseline(t).Method, Files: []string{"a.sh"}, Coverage: Ratio{1, 2}, Mutation: Ratio{1, 2}}}
	m.CoverageCounts = m.Baseline.Coverage
	m.MutationCounts = m.Baseline.Mutation
	if UpdateBaseline(path, m, "current", true) == nil {
		t.Fatal("concurrent write accepted")
	}
	unlock()
	if err = UpdateBaseline(path, m, "current", true); err != nil {
		t.Fatal(err)
	}
}
