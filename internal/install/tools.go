package install

import (
	"context"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func InstallTools(ctx context.Context, cache string, manifest Manifest) ([]string, error) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return nil, fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	for _, p := range manifest.Packages {
		if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
			continue
		}
		root := packageRoot(cache, p)
		marker, err := os.ReadFile(filepath.Join(root, ".archive-sha256"))
		if err == nil {
			if strings.TrimSpace(string(marker)) != packageDigest(p) {
				return nil, fmt.Errorf("cached %s has unexpected checksum; remove the damaged package explicitly", p.Name)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		if _, err = InstallPackage(ctx, cache, p); err != nil {
			return nil, fmt.Errorf("install %s: %w", p.Name, err)
		}
	}
	paths, err := InstalledPaths(cache, manifest)
	if err != nil {
		return nil, err
	}
	for _, p := range manifest.Packages {
		if p.Name == "kcov" && p.Format == "deb" && p.OS == runtime.GOOS && p.Arch == runtime.GOARCH {
			if err := exposeKcov(packageRoot(cache, p)); err != nil {
				return nil, err
			}
		}
	}
	return paths, nil
}

// PreferSystemKcov validates and reuses an existing PATH installation.
func PreferSystemKcov(ctx context.Context, manifest Manifest) (Manifest, error) {
	if _, err := exec.LookPath("kcov"); err != nil {
		return manifest, nil
	}
	if err := quality.CheckKcov(ctx, 15*time.Second); err != nil {
		return manifest, fmt.Errorf("existing kcov is incompatible: %w", err)
	}
	filtered := Manifest{Schema: manifest.Schema}
	for _, p := range manifest.Packages {
		if p.Name != "kcov" {
			filtered.Packages = append(filtered.Packages, p)
		}
	}
	return filtered, nil
}
