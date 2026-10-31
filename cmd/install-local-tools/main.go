package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	dir := flag.String("directory", "", "verified archives")
	cache := flag.String("cache", "", "tool cache destination")
	flag.Parse()
	if err := run(*dir, *cache); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(dir, cache string) error {
	if dir == "" || cache == "" {
		return fmt.Errorf("--directory and --cache required")
	}
	manifest, err := install.ParseManifest(install.BundledManifest)
	if err != nil {
		return err
	}
	for _, p := range manifest.Packages {
		if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
			continue
		}
		if _, err = install.InstallLocalPackage(context.Background(), cache, p, filepath.Join(dir, p.Name+"-linux-"+p.Arch+".tar.gz")); err != nil {
			return err
		}
	}
	_, err = install.InstalledPaths(cache, manifest)
	return err
}
