package install

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadVerifiesBeforeReplacingDestination(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("archive")) }))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if Download(context.Background(), server.URL, strings.Repeat("00", 32), path) == nil {
		t.Fatal("wrong checksum accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "old" {
		t.Fatal("destination changed on failed verification", err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("archive")))
	if err = Download(context.Background(), server.URL, sum, path); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "archive" {
		t.Fatal(err)
	}
	if Download(context.Background(), "http://example.invalid", sum, path) == nil {
		t.Fatal("HTTP accepted")
	}
}
