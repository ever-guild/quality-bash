package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCoverageTerminalAndSubstitution(t *testing.T) {
	for _, tool := range []string{"kcov", "bats", "script"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " required")
		}
	}
	root := t.TempDir()
	source := `#!/usr/bin/env bash
if [[ -t 1 ]]; then
 printf 'terminal\n'
else
 printf 'pipe\n'
fi
printf 'error\n' >&2
exit 7
`
	tests := `@test "substitution and temporary files preserve status and streams" {
 local output status
 output=$(bash "$BATS_TEST_DIRNAME/example.sh" 2>"$BATS_TEST_TMPDIR/stderr") && status=0 || status=$?
 [ "$status" -eq 7 ]
 [ "$output" = pipe ]
 [ "$(cat "$BATS_TEST_TMPDIR/stderr")" = error ]
}
@test "pseudo terminal remains a terminal" {
 run script -q -e -c "bash '$BATS_TEST_DIRNAME/example.sh'" /dev/null
 [ "$status" -eq 7 ]
 [[ "$output" == *terminal* ]]
 [[ "$output" == *error* ]]
 [[ "$output" != *pipe* ]]
}
`
	for name, data := range map[string]string{"example.sh": source, "example.bats": tests} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 30 * time.Second, Stdout: os.Stdout, Stderr: os.Stderr}
	if err = r.Test(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	report, err := r.Coverage(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if report.Score.Numerator != report.Score.Denominator {
		t.Fatalf("both branches should execute: %+v", report)
	}
}
