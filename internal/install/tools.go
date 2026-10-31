package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func InstallTools(ctx context.Context, cache string, manifest Manifest) ([]string, error) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return nil, fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	for _, p := range manifest.Packages {
		if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
			continue
		}
		root := filepath.Join(cache, p.Name, p.Version+"-"+p.OS+"-"+p.Arch)
		marker, err := os.ReadFile(filepath.Join(root, ".archive-sha256"))
		if err == nil {
			if strings.TrimSpace(string(marker)) != p.SHA256 {
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
	return InstalledPaths(cache, manifest)
}
