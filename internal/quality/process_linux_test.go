package quality

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessDiagnostics(t *testing.T) {
	var out, errout bytes.Buffer
	r := RunProcess(context.Background(), time.Second, t.TempDir(), nil, &out, &errout, "bash", "-c", "printf output; printf diagnostic >&2; exit 7")
	if r.ExitCode != 7 || out.String() != "output" || errout.String() != "diagnostic" || r.InfrastructureError != "" {
		t.Fatalf("%+v %q %q", r, out.String(), errout.String())
	}
	r = RunProcess(context.Background(), time.Second, "", nil, io.Discard, io.Discard, "/nonexistent/quality-bash-tool")
	if r.InfrastructureError == "" {
		t.Fatal("missing executable was not infrastructure failure")
	}
}
func TestTimeoutKillsChildren(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "child survived")
	r := RunProcess(context.Background(), 100*time.Millisecond, dir, nil, io.Discard, io.Discard, "bash", "-c", `(sleep .4; touch "$1") & wait`, "test", marker)
	if !r.TimedOut {
		t.Fatalf("%+v", r)
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("child survived: %v", err)
	}
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	r := RunProcess(ctx, time.Second, "", nil, io.Discard, io.Discard, "bash", "-c", "sleep 10 & wait")
	if !r.Cancelled || r.TimedOut {
		t.Fatalf("%+v", r)
	}
}
