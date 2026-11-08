package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

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
