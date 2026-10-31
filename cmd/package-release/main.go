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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	toolsDirectory := flag.String("tools", "", "optional directory containing the eight pinned tool archives")
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
	projectLicense, err := os.ReadFile("LICENSE")
	if err != nil {
		return err
	}
	license, err := os.ReadFile("licenses/mvdan-sh.txt")
	if err != nil {
		return err
	}
	goLicense, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		return err
	}
	readme, err := os.ReadFile("reamde.md")
	if err != nil {
		return err
	}
	contributing, err := os.ReadFile("contributing.md")
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
		files["licenses/go.txt"] = goLicense
		files["LICENSE"] = projectLicense
		files["reamde.md"] = readme
		files["contributing.md"] = contributing
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
	if *toolsDirectory != "" {
		manifest, err := install.ParseManifest(install.BundledManifest)
		if err != nil {
			return err
		}
		for _, pkg := range manifest.Packages {
			name := pkg.Name + "-" + pkg.OS + "-" + pkg.Arch + ".tar.gz"
			if err := copyVerifiedArchive(filepath.Join(*toolsDirectory, name), filepath.Join(*output, name), pkg.SHA256); err != nil {
				return err
			}
			fmt.Fprintf(&checksums, "%s  %s\n", pkg.SHA256, name)
		}
	}
	return os.WriteFile(filepath.Join(*output, "SHA256SUMS"), checksums.Bytes(), 0644)
}

func copyVerifiedArchive(source, destination, expected string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(destination), ".archive-*")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(output, hash), input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expected {
		return fmt.Errorf("tool archive checksum mismatch: %s", source)
	}
	if err := os.Chmod(output.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(output.Name(), destination)
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
