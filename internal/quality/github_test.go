package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	summary := filepath.Join(root, "summary")
	for _, path := range []string{output, summary} {
		if err := os.WriteFile(path, []byte("existing\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GITHUB_OUTPUT", output)
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	m := Measurement{Qualified: true, Baseline: Baseline{Coverage: Ratio{2, 3}, Mutation: Ratio{6667, 10000}}}
	m.CoverageCounts = Ratio{2, 3}
	m.MutationCounts = Ratio{2, 3}
	if err := PublishGitHub(m, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "existing\n") || !strings.Contains(string(data), "coverage_numerator=2\n") || !strings.Contains(string(data), "passed=false\n") || !strings.Contains(string(data), "mutation_numerator=2\nmutation_denominator=3\n") {
		t.Fatal(string(data))
	}
}

func TestGitHubUnqualifiedMeasurement(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	summary := filepath.Join(t.TempDir(), "summary")
	for _, path := range []string{output, summary} {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GITHUB_OUTPUT", output)
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	if err := PublishGitHub(Measurement{}, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "passed=false\n" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	data, err = os.ReadFile(summary)
	if err != nil || !strings.Contains(string(data), "No qualified measurement") {
		t.Fatalf("summary=%q error=%v", data, err)
	}
}
