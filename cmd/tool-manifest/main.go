package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	dir := flag.String("directory", "tool-dist", "directory containing both architectures")
	release := flag.String("release", "v1.0.0-alpha.1", "release tag")
	output := flag.String("output", "tools-manifest.json", "manifest destination")
	flag.Parse()
	manifest := install.Manifest{Schema: 1}
	versions := map[string]string{"bash": "5.2.21", "bats": "1.10.0", "shellcheck": "0.9.0", "kcov": "43"}
	for _, arch := range []string{"amd64", "arm64"} {
		for _, name := range []string{"bash", "bats", "shellcheck", "kcov"} {
			filename := name + "-linux-" + arch + ".tar.gz"
			file, err := os.Open(filepath.Join(*dir, filename))
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, err = io.Copy(hash, file)
			file.Close()
			if err != nil {
				return err
			}
			manifest.Packages = append(manifest.Packages, install.Package{Name: name, Version: versions[name], OS: "linux", Arch: arch, URL: "https://github.com/ever-guild/quality-bash/releases/download/" + *release + "/" + filename, SHA256: fmt.Sprintf("%x", hash.Sum(nil)), Executables: []string{"bin/" + name}})
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if _, err = install.ParseManifest(data); err != nil {
		return err
	}
	return os.WriteFile(*output, append(data, '\n'), 0644)
}
