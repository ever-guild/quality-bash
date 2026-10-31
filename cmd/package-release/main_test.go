package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUnverifiedToolCannotReplaceArchive(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "input"), filepath.Join(dir, "output")
	for path, content := range map[string]string{source: "corrupt", destination: "previous"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyVerifiedArchive(source, destination, "incorrect"); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "previous" {
		t.Fatalf("previous archive changed: %q %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("temporary archive leaked: %v %v", entries, err)
	}
}

func TestDeterministicArchive(t *testing.T) {
	files := map[string][]byte{"quality-bash": []byte("binary"), "licenses/dependency.txt": []byte("license")}
	first, err := pack(files)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pack(files)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("non deterministic archive")
	}
	gz, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	count := 0
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, files[header.Name]) {
			t.Fatal("content mismatch")
		}
		if header.Name == "quality-bash" && header.Mode != 0755 {
			t.Fatal("binary not executable")
		}
	}
	if count != len(files) {
		t.Fatal("missing entries")
	}
}
