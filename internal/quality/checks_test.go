package quality

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestParallelDiagnosticsRemainSeparate(t *testing.T) {
	root := t.TempDir()
	r := Runner{Root: root, Reports: filepath.Join(root, "reports"), Timeout: time.Second}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.Execute(context.Background(), "failure", "bash", "-c", "printf diagnostic >&2; exit 7") == nil {
				t.Error("failure accepted")
			}
		}()
	}
	wg.Wait()
	entries, err := os.ReadDir(r.Reports)
	if err != nil || len(entries) != 4 {
		t.Fatal("diagnostics overwritten", err)
	}
	for _, entry := range entries {
		dir := filepath.Join(r.Reports, entry.Name())
		var result ProcessResult
		if err := ReadJSON(filepath.Join(dir, "result.json"), &result); err != nil {
			t.Fatal(err)
		}
		if result.ExitCode != 7 {
			t.Fatal(result)
		}
		data, err := os.ReadFile(filepath.Join(dir, "stderr.log"))
		if err != nil || string(data) != "diagnostic" {
			t.Fatal("diagnostic lost", err)
		}
	}
}
