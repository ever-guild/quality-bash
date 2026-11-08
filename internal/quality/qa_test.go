package quality

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFullQualityCycle(t *testing.T) {
	for _, tool := range []string{"bash", "bats", "shellcheck", "kcov"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " required")
		}
	}
	root := t.TempDir()
	script := `#!/usr/bin/env bash
if [[ ${1:-} == unused ]]; then
 echo uncovered
 exit 1
fi
printf 'hello\n'
exit 0
`
	tests := `@test "default succeeds" {
 run bash "$BATS_TEST_DIRNAME/example.sh"
 [ "$status" -eq 0 ]
 [ "$output" = hello ]
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
	runner := Runner{Root: root, Reports: filepath.Join(root, "quality-reports"), Timeout: 20 * time.Second}
	baselinePath := filepath.Join(root, "quality-baseline.json")
	measurement, err := runner.Measure(context.Background(), selection, 2, baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Baseline.Mutation != (Ratio{2, 3}) {
		t.Fatalf("mutation: %+v", measurement.Baseline.Mutation)
	}
	if err = UpdateBaseline(baselinePath, measurement, measurement.Fingerprint, true); err != nil {
		t.Fatal(err)
	}
	fresh, err := runner.LoadFresh(context.Background(), selection, 2, baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.CoverageCounts != (Ratio{3, 5}) {
		t.Fatalf("lost coverage counts: %+v", fresh.CoverageCounts)
	}
	if fresh.MutationCounts != (Ratio{2, 3}) {
		t.Fatalf("lost mutation counts: %+v", fresh.MutationCounts)
	}
	var baseline Baseline
	if err = ReadJSON(baselinePath, &baseline); err != nil {
		t.Fatal(err)
	}
	if err = baseline.Check(fresh.Baseline); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "example.bats"), []byte(tests+"\n# changed\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = runner.LoadFresh(context.Background(), selection, 2, baselinePath); err == nil {
		t.Fatal("changed tests accepted")
	}
}

func TestFailedMeasurementInvalidatesPreviousReport(t *testing.T) {
	root := t.TempDir()
	reports := filepath.Join(root, "reports")
	if err := os.Mkdir(reports, 0700); err != nil {
		t.Fatal(err)
	}
	previous := Measurement{Qualified: true, Fingerprint: "old", Baseline: validBaseline(t)}
	previous.MutationCounts = previous.Baseline.Mutation
	path := filepath.Join(reports, "measurement.json")
	if err := WriteJSON(path, previous); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	r := Runner{Root: root, Reports: reports, Timeout: time.Second}
	if _, err := r.Measure(context.Background(), Selection{Scripts: []string{"a.sh"}, Tests: []string{"a.bats"}}, 1, filepath.Join(root, "baseline.json")); err == nil {
		t.Fatal("missing tools accepted")
	}
	var after Measurement
	if err := ReadJSON(path, &after); err != nil {
		t.Fatal(err)
	}
	if after.Qualified {
		t.Fatal("previous report still qualified")
	}
}
