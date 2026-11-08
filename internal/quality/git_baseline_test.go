package quality

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTargetBaseline(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", output, err)
		}
	}
	git("init", "--quiet")
	baseline := Baseline{Method: validBaseline(t).Method, Files: []string{"script.sh"}, Coverage: Ratio{8, 10}, Mutation: Ratio{6, 10}}
	path := filepath.Join(root, "quality-baseline.json")
	if err := WriteJSON(path, baseline); err != nil {
		t.Fatal(err)
	}
	git("add", "quality-baseline.json")
	git("-c", "commit.gpgsign=false", "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "baseline")
	baseline.Coverage = Ratio{1, 10}
	if err := WriteJSON(path, baseline); err != nil {
		t.Fatal(err)
	}
	target, err := ReadGitBaseline(context.Background(), root, path, "HEAD", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if target.Check(baseline) == nil {
		t.Fatal("working tree regression escaped target gate")
	}
	for _, revision := range []string{"--help", "missing-revision"} {
		if _, err = ReadGitBaseline(context.Background(), root, path, revision, time.Second); err == nil {
			t.Fatalf("accepted %s", revision)
		}
	}
}
