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
	m := Measurement{Baseline: Baseline{Coverage: Ratio{2, 3}, Mutation: Ratio{4, 5}}}
	if err := PublishGitHub(m, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "existing\n") || !strings.Contains(string(data), "coverage_numerator=2\n") || !strings.Contains(string(data), "passed=false\n") {
		t.Fatal(string(data))
	}
}
