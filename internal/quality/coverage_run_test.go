package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRealCoverage(t *testing.T) {
	if _, err := exec.LookPath("kcov"); err != nil {
		t.Skip("kcov required")
	}
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	root := t.TempDir()
	script := `#!/usr/bin/env bash
if [[ ${1:-} == unused ]]; then
 echo uncovered
fi
printf 'hello\n'
printf 'diagnostic\n' >&2
exit 0
`
	tests := `@test "separate process preserves output" {
 run bash "$BATS_TEST_DIRNAME/example.sh"
 [ "$status" -eq 0 ]
 [ "$output" = $'hello\ndiagnostic' ]
}
`
	for name, data := range map[string]string{"example.sh": script, "example.bats": tests} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	selection, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	runner := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 20 * time.Second, Stderr: os.Stderr}
	if err = runner.Test(context.Background(), selection); err != nil {
		t.Fatal(err)
	}
	report, err := runner.Coverage(context.Background(), selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 1 || report.Score.Numerator == 0 || report.Score.Numerator >= report.Score.Denominator {
		t.Fatalf("unexpected coverage: %+v", report)
	}
	found := false
	for _, line := range report.Files[0].Uncovered {
		if line == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing uncovered line 3: %+v", report)
	}
}
