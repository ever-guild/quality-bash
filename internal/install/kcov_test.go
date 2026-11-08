package install

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func debFixture(t *testing.T, headers []*tar.Header) []byte {
	t.Helper()
	var payload bytes.Buffer
	tw := tar.NewWriter(&payload)
	for _, h := range headers {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write(bytes.Repeat([]byte("x"), int(h.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	var deb bytes.Buffer
	deb.WriteString("!<arch>\n")
	fmt.Fprintf(&deb, "%-16s%-12s%-6s%-6s%-8s%-10d`\n", "data.tar/", "0", "0", "0", "100644", payload.Len())
	deb.Write(payload.Bytes())
	return deb.Bytes()
}
func TestDebExtraction(t *testing.T) {
	root := t.TempDir()
	data := debFixture(t, []*tar.Header{
		{Name: "./usr/bin/kcov", Typeflag: tar.TypeReg, Mode: 0755, Size: 1},
		{Name: "./usr/lib/libreal.so.1", Typeflag: tar.TypeReg, Mode: 0644, Size: 1},
		{Name: "./usr/lib/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
		{Name: "./etc/ignored", Typeflag: tar.TypeReg, Size: 1},
	})
	if err := extractDeb(bytes.NewReader(data), root); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"usr/bin/kcov", "usr/lib/libreal.so.1"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"usr/lib/link", "etc/ignored"} {
		if _, err := os.Lstat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatal("unexpected extracted path", path, err)
		}
	}
	for _, path := range []string{"../escape", "/absolute"} {
		data = debFixture(t, []*tar.Header{{Name: path, Typeflag: tar.TypeReg, Size: 1}})
		if err := extractDeb(bytes.NewReader(data), t.TempDir()); err == nil {
			t.Fatal("unsafe path accepted", path)
		}
	}
	if err := extractDeb(bytes.NewReader([]byte("invalid")), t.TempDir()); err == nil {
		t.Fatal("invalid ar accepted")
	}
}
func TestPreferSystemKcov(t *testing.T) {
	manifest := Manifest{Schema: 1, Packages: []Package{{Name: "kcov"}, {Name: "bats"}}}
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if got, err := PreferSystemKcov(context.Background(), manifest); err != nil || len(got.Packages) != 2 {
		t.Fatal(got, err)
	}
	path := filepath.Join(bin, "kcov")
	for _, version := range []string{"43", "42", "garbage"} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf 'kcov "+version+"\\n'\n"), 0755); err != nil {
			t.Fatal(err)
		}
		got, err := PreferSystemKcov(context.Background(), manifest)
		if version == "43" {
			if err != nil || len(got.Packages) != 1 || got.Packages[0].Name != "bats" {
				t.Fatal(got, err)
			}
		} else if err == nil {
			t.Fatal("incompatible system tool accepted")
		}
	}
}
func TestExposeKcovPreservesUnrelatedFile(t *testing.T) {
	bin := t.TempDir()
	t.Setenv("XDG_BIN_HOME", bin)
	path := filepath.Join(bin, "kcov")
	if err := os.WriteFile(path, []byte("user file"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := exposeKcov(t.TempDir()); err == nil {
		t.Fatal("overwrote unrelated file")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "user file" {
		t.Fatal("changed unrelated file")
	}
}
