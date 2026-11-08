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

func TestDoctorRetainsBuildSuffix(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash required")
	}
	output, err := exec.Command(bash, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	versions, _ := Doctor(context.Background(), time.Second, nil)
	for _, tool := range versions {
		if tool.Name == "bash" {
			if tool.Version == "" || !strings.Contains(string(output), "version "+tool.Version+" ") {
				t.Fatalf("version token truncated: %q", tool.Version)
			}
			return
		}
	}
	t.Fatal("bash absent from doctor")
}

func TestSystemBashCheck(t *testing.T) {
	for _, tc := range []struct {
		version    string
		compatible bool
	}{
		{"5.2.21(1)-release", true}, {"5.3.0(1)-release", true}, {"5.1.16(1)-release", false}, {"unknown", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nprintf '%s\\n' 'GNU bash, version " + tc.version + " (test)'\n"
			if err := os.WriteFile(filepath.Join(dir, "bash"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			if err := CheckBash(context.Background(), time.Second); (err == nil) != tc.compatible {
				t.Fatalf("compatible=%v: %v", tc.compatible, err)
			}
		})
	}
	t.Setenv("PATH", t.TempDir())
	if CheckBash(context.Background(), time.Second) == nil {
		t.Fatal("missing system Bash accepted")
	}
}
