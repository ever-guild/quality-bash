// package-release builds deterministic development archives. Publication and
// release qualification are separate steps.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	output := flag.String("output", "dist", "artifact directory")
	version := flag.String("version", "dev", "dev or v1.0.0-alpha.1")
	flag.Parse()
	if *version != "dev" && *version != "v1.0.0-alpha.1" {
		return fmt.Errorf("unsupported release version")
	}
	if *version != "dev" {
		if _, err := install.ParseManifest(install.BundledManifest); err != nil {
			return fmt.Errorf("release requires a complete tool manifest: %w", err)
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "quality-bash-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	license, err := os.ReadFile("licenses/mvdan-sh.txt")
	if err != nil {
		return err
	}
	var checksums bytes.Buffer
	for _, arch := range []string{"amd64", "arm64"} {
		binary := filepath.Join(temp, "quality-bash-"+arch)
		buildVersion := *version
		if buildVersion == "dev" {
			buildVersion = "v1.0.0-alpha.1-dev"
		}
		cmd := exec.Command("go", "build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w -X main.version="+buildVersion, "-o", binary, "./cmd/quality-bash")
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return err
		}
		data, err := os.ReadFile(binary)
		if err != nil {
			return err
		}
		files := map[string][]byte{"quality-bash": data, "licenses/mvdan-sh.txt": license, "BUILD-STATUS.txt": []byte("Development build. Not a qualified v1.0.0-alpha.1 release.\n")}
		if *version != "dev" {
			files["BUILD-STATUS.txt"] = []byte("Version: " + *version + "\n")
			files["tools-manifest.json"] = install.BundledManifest
		}
		archive, err := pack(files)
		if err != nil {
			return err
		}
		name := "quality-bash-" + *version + "-linux-" + arch + ".tar.gz"
		if err = os.WriteFile(filepath.Join(*output, name), archive, 0644); err != nil {
			return err
		}
		fmt.Fprintf(&checksums, "%x  %s\n", sha256.Sum256(archive), name)
	}
	return os.WriteFile(filepath.Join(*output, "SHA256SUMS"), checksums.Bytes(), 0644)
}
func pack(files map[string][]byte) ([]byte, error) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mode := int64(0644)
		if name == "quality-bash" {
			mode = 0755
		}
		header := &tar.Header{Name: name, Mode: mode, Size: int64(len(files[name])), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tw.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
