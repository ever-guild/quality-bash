package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRealMutationRun(t *testing.T) {
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	root := t.TempDir()
	script := "#!/usr/bin/env bash\nif [[ ${1:-} == unused ]]; then exit 1; fi\nexit 0\n"
	tests := `@test "successful default" {
 run bash "$BATS_TEST_DIRNAME/example.sh"
 [ "$status" -eq 0 ]
}
`
	for name, data := range map[string]string{"example.sh": script, "example.bats": tests} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 10 * time.Second}
	report, err := r.Mutate(context.Background(), s, 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.Score.Numerator != 2 || report.Score.Denominator != 3 {
		t.Fatalf("unexpected score %+v", report)
	}
	after, err := os.ReadFile(filepath.Join(root, "example.sh"))
	if err != nil || string(after) != script {
		t.Fatal("original modified", err)
	}
}
