package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func archiveBytes(t *testing.T, name string, kind byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	header := &tar.Header{Name: name, Mode: 0755, Typeflag: kind}
	if kind == tar.TypeReg {
		header.Size = 2
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if kind == tar.TypeReg {
		tw.Write([]byte("ok"))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}
func TestExtract(t *testing.T) {
	root := t.TempDir()
	if err := ExtractTarGzip(bytes.NewReader(archiveBytes(t, "bin/tool", tar.TypeReg)), root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "bin/tool"))
	if err != nil || string(data) != "ok" {
		t.Fatal(err)
	}
	for _, name := range []string{"../escape", "/absolute"} {
		if ExtractTarGzip(bytes.NewReader(archiveBytes(t, name, tar.TypeReg)), t.TempDir()) == nil {
			t.Fatal("unsafe path accepted")
		}
	}
	if ExtractTarGzip(bytes.NewReader(archiveBytes(t, "link", tar.TypeSymlink)), t.TempDir()) == nil {
		t.Fatal("symlink accepted")
	}
}

func TestExtractGlobalPAXMetadata(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"comment": "upstream commit"}}); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "source/file", Typeflag: tar.TypeReg, Mode: 0644, Size: 2}); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte("ok"))
	tw.Close()
	gz.Close()
	root := t.TempDir()
	if err := ExtractTarGzip(bytes.NewReader(buf.Bytes()), root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "pax_global_header")); !os.IsNotExist(err) {
		t.Fatal("metadata written as file")
	}
	data, err := os.ReadFile(filepath.Join(root, "source/file"))
	if err != nil || string(data) != "ok" {
		t.Fatal(err)
	}
}

func TestSourceExtractionOmitsOnlyRequestedDirectory(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, header := range []*tar.Header{
		{Name: "upstream/tests/link", Typeflag: tar.TypeSymlink, Linkname: "fixture"},
		{Name: "upstream/src/main", Typeflag: tar.TypeReg, Mode: 0644, Size: 2},
	} {
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeReg {
			tw.Write([]byte("ok"))
		}
	}
	tw.Close()
	gz.Close()
	root := t.TempDir()
	if err := ExtractSourceTarGzip(bytes.NewReader(buf.Bytes()), root, "upstream/tests"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "upstream/src/main")); err != nil {
		t.Fatal(err)
	}
	if err := ExtractTarGzip(bytes.NewReader(buf.Bytes()), t.TempDir()); err == nil {
		t.Fatal("runtime archive accepted symlink")
	}
}
