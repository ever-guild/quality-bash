package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCacheRequiresExplicitInstallation(t *testing.T) {
	root := t.TempDir()
	p := Package{Name: "bash", Version: "5.2", OS: runtime.GOOS, Arch: runtime.GOARCH, SHA256: strings.Repeat("ab", 32), Executables: []string{"bin/bash"}}
	m := Manifest{Schema: 1, Packages: []Package{p}}
	if _, err := InstalledPaths(root, m); err == nil {
		t.Fatal("missing cache accepted")
	}
	dir := filepath.Join(root, p.Name, p.Version+"-"+p.OS+"-"+p.Arch)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin/bash"), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".archive-sha256")
	if err := os.WriteFile(marker, []byte(p.SHA256), 0600); err != nil {
		t.Fatal(err)
	}
	paths, err := InstalledPaths(root, m)
	if err != nil || len(paths) != 1 {
		t.Fatal(paths, err)
	}
	if err := os.WriteFile(marker, []byte("wrong"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := InstalledPaths(root, m); err == nil {
		t.Fatal("wrong archive accepted")
	}
}

func TestInstallToolsReusesCache(t *testing.T) {
	root := t.TempDir()
	p := Package{Name: "bash", Version: "5.2", OS: runtime.GOOS, Arch: runtime.GOARCH, SHA256: strings.Repeat("ab", 32), URL: "https://invalid.invalid/never-request", Executables: []string{"bin/bash"}}
	dir := filepath.Join(root, p.Name, p.Version+"-"+p.OS+"-"+p.Arch)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin/bash"), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".archive-sha256"), []byte(p.SHA256), 0600); err != nil {
		t.Fatal(err)
	}
	paths, err := InstallTools(context.Background(), root, Manifest{Schema: 1, Packages: []Package{p}})
	if err != nil || len(paths) != 1 {
		t.Fatal(paths, err)
	}
}
