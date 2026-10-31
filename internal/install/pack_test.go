package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackRoundTrip(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tool"), []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}
	a, err := PackDirectory(source, filepath.Join(root, "a.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := PackDirectory(source, filepath.Join(root, "b.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("non deterministic archive")
	}
	file, err := os.Open(filepath.Join(root, "a.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	dest := t.TempDir()
	if err = ExtractTarGzip(file, dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "tool"))
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("executable mode lost", err)
	}
}
