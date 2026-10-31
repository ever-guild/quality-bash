package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRejectsOverlappingOutputBeforeExecution(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.sh", "a.bats"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# fixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err := run(context.Background(), []string{"lint", "--root", root, "--reports", "."})
	if err == nil || !strings.Contains(err.Error(), "reports directory") {
		t.Fatalf("unexpected error %v", err)
	}
}
func TestCLIRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"baseline"}, {"tools", "unknown"}, {"lint", "--timeout", "0s"}, {"mutation", "--jobs", "0"}, {"version", "extra"}} {
		if err := run(context.Background(), args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
