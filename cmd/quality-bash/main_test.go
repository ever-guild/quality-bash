package main

import (
	"context"
	"github.com/ever-guild/quality-bash/internal/install"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBaselineCheckRejectsForgedMeasurement(t *testing.T) {
	for _, tool := range []string{"bash", "bats", "shellcheck", "kcov"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " required")
		}
	}
	previous := install.BundledManifest
	install.BundledManifest = nil
	t.Cleanup(func() { install.BundledManifest = previous })
	root := t.TempDir()
	for name, content := range map[string]string{
		"a.sh":   "#!/usr/bin/env bash\nexit 0\n",
		"a.bats": "@test \"always fails\" {\n false\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	selection, err := quality.Discover(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(root, "quality-reports")
	baseline := filepath.Join(root, "quality-baseline.json")
	runner := quality.Runner{Root: root, Reports: reports, Timeout: 5 * time.Minute}
	settings, err := runner.Settings(context.Background(), selection, 1)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := quality.Fingerprint(root, []string{reports, baseline, baseline + ".lock", baseline + ".migration.json"}, settings)
	if err != nil {
		t.Fatal(err)
	}
	forged := quality.Measurement{Qualified: true, Fingerprint: fingerprint, Baseline: quality.Baseline{Schema: 1, Files: selection.Scripts, Method: settings.Method, Coverage: quality.Ratio{Numerator: 1, Denominator: 1}, Mutation: quality.Ratio{Numerator: 1, Denominator: 1}}}
	if err := os.Mkdir(reports, 0700); err != nil {
		t.Fatal(err)
	}
	if err := quality.WriteJSON(filepath.Join(reports, "measurement.json"), forged); err != nil {
		t.Fatal(err)
	}
	if err := quality.WriteJSON(baseline, forged.Baseline); err != nil {
		t.Fatal(err)
	}
	// The forged report satisfies the old cache checks, proving the attack setup.
	if _, err := runner.LoadFresh(context.Background(), selection, 1, baseline); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"baseline", "check", "--root", root}); err == nil {
		t.Fatal("forged report bypassed failing tests")
	}
}

func TestCLIRejectsOverlappingOutputBeforeExecution(t *testing.T) {
	previous := install.BundledManifest
	install.BundledManifest = nil
	t.Cleanup(func() { install.BundledManifest = previous })
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
