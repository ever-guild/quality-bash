package quality

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOutputSymlinksCannotOverwriteExternalFiles(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	protected := filepath.Join(outside, "measurement.json")
	if err := os.WriteFile(protected, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "reports")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	runner := Runner{Root: root, Reports: link, Timeout: time.Second}
	if _, err := runner.Measure(context.Background(), Selection{}, 1, filepath.Join(root, "baseline.json")); err == nil {
		t.Fatal("reports symlink accepted")
	}
	if err := WriteJSON(filepath.Join(link, "measurement.json"), Measurement{}); err == nil {
		t.Fatal("write followed symlink")
	}
	data, err := os.ReadFile(protected)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("external file changed: %q %v", data, err)
	}
	baseline := filepath.Join(root, "baseline.json")
	if err := os.Symlink(protected, baseline+".lock"); err != nil {
		t.Fatal(err)
	}
	if unlock, err := lockBaseline(baseline); err == nil {
		unlock()
		t.Fatal("lock symlink accepted")
	}
	if err := ValidateOutputPaths(root, filepath.Join(link, "nested"), baseline, Selection{}); err == nil {
		t.Fatal("symlink ancestor accepted")
	}
}

func TestOutputCannotHideSelectedInputs(t *testing.T) {
	s := Selection{Scripts: []string{"src/a.sh"}, Tests: []string{"a.bats"}}
	for _, tc := range []struct{ reports, baseline string }{{"/project", "baseline.json"}, {"src", "baseline.json"}, {"reports", "src/a.sh"}, {"reports", "reports/baseline.json"}} {
		if ValidateOutputPaths("/project", tc.reports, tc.baseline, s) == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	if err := ValidateOutputPaths("/project", "reports", "baseline.json", s); err != nil {
		t.Fatal(err)
	}
}
