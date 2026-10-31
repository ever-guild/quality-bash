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
