package install

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallPackageHTTPS(t *testing.T) {
	payload := archiveBytes(t, "bin/tool", '0')
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) }))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	p := Package{Name: "tool", Version: "1.0", OS: runtime.GOOS, Arch: runtime.GOARCH, URL: server.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256(payload)), Executables: []string{"bin/tool"}}
	cache := t.TempDir()
	path, err := InstallPackage(context.Background(), cache, p)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(path, "bin/tool"))
	if err != nil || string(data) != "ok" {
		t.Fatal(err)
	}
	if _, err = InstalledPaths(cache, Manifest{Schema: 1, Packages: []Package{p}}); err != nil {
		t.Fatal(err)
	}
	if _, err = InstallTools(context.Background(), cache, Manifest{Schema: 1, Packages: []Package{p}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(cache, "tool"))
	if err != nil || len(entries) != 1 {
		t.Fatal("staging files leaked", err)
	}
}
