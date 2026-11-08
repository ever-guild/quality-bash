package qualitybash

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExampleWorkflowUsesMake(t *testing.T) {
	workflow, err := os.ReadFile("example/.github/workflows/qa.yml")
	if err != nil {
		t.Fatal(err)
	}
	var runs []string
	for line := range strings.SplitSeq(string(workflow), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "run:") {
			command := strings.TrimSpace(strings.TrimPrefix(line, "run:"))
			if !strings.HasPrefix(command, "make ") {
				t.Fatal("workflow must use make:", command)
			}
			runs = append(runs, command)
		}
	}
	if len(runs) != 2 {
		t.Fatalf("expected installation and QA steps, got %v", runs)
	}
	makefile, err := os.ReadFile("example/makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, revision := range []string{"", "0123456789012345678901234567890123456789"} {
		t.Run("revision="+revision, func(t *testing.T) {
			root := t.TempDir()
			binary := filepath.Join(root, "quality-bash")
			record := filepath.Join(root, "arguments")
			if err := os.WriteFile(filepath.Join(root, "makefile"), makefile, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$RECORD\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-eu", "-o", "pipefail", "-c", runs[1])
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "QUALITY_BASH="+binary, "RECORD="+record, "BASE_REVISION="+revision)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v: %s", err, output)
			}
			got, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			want := "qa\n"
			if revision != "" {
				want += "--base-revision\n" + revision + "\n"
			}
			if string(got) != want {
				t.Fatalf("CLI arguments %q, want %q", got, want)
			}
		})
	}
}
