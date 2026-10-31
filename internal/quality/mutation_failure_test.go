package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExcludedHelperCannotInflateMutationScore(t *testing.T) {
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	root := t.TempDir()
	reports := filepath.Join(root, "reports")
	if err := os.Mkdir(reports, 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"example.sh":     "#!/usr/bin/env bash\nexit 0\n",
		"reports/helper": "true\n",
		"example.bats":   "@test \"helper required\" {\n source \"$BATS_TEST_DIRNAME/reports/helper\"\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	runner := Runner{Root: root, Reports: reports, Timeout: 5 * time.Second}
	if err := runner.Test(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	report, err := runner.Mutate(context.Background(), s, 1)
	if err == nil || !strings.Contains(err.Error(), "mutation copy") || report.Qualified || len(report.Mutations) != 0 {
		t.Fatalf("excluded helper inflated score: %+v, %v", report, err)
	}
}

func TestMutantTimeoutBlocksQualification(t *testing.T) {
	if _, err := exec.LookPath("bats"); err != nil {
		t.Skip("bats required")
	}
	root := t.TempDir()
	source := `#!/usr/bin/env bash
if [[ ${1:-} == slow ]]; then
 sleep 10
fi
exit 0
`
	tests := `@test "default" {
 run bash "$BATS_TEST_DIRNAME/example.sh"
 [ "$status" -eq 0 ]
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
	runner := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: 500 * time.Millisecond}
	report, err := runner.Mutate(context.Background(), s, 2)
	if err == nil || report.Qualified {
		t.Fatal("timeout qualified")
	}
	timeouts := 0
	for _, mutation := range report.Mutations {
		if mutation.Result == "timeout" {
			timeouts++
		}
	}
	if timeouts != 1 {
		t.Fatalf("expected one timeout: %+v", report)
	}
	if report.Score.Numerator != 1 || report.Score.Denominator != 1 {
		t.Fatalf("timeout counted as detected: %+v", report.Score)
	}
}
func TestMissingBatsBlocksOriginalTests(t *testing.T) {
	root := t.TempDir()
	for name, data := range map[string]string{"example.sh": "exit 0\n", "example.bats": "@test x { true; }\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	report, err := (Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: time.Second}).Mutate(context.Background(), s, 1)
	if err == nil || report.Qualified || len(report.Mutations) != 0 {
		t.Fatal("missing tool did not stop before mutations")
	}
}
