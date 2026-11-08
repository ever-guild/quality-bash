package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func CacheRoot() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "quality-bash"), nil
}

// InstalledPaths verifies the archive identity and expected executable layout.
// It never downloads or repairs a package implicitly.
func InstalledPaths(cache string, manifest Manifest) ([]string, error) {
	var paths []string
	seen := map[string]bool{}
	for _, p := range manifest.Packages {
		if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
			continue
		}
		root := packageRoot(cache, p)
		marker, err := os.ReadFile(filepath.Join(root, ".archive-sha256"))
		if err != nil {
			return nil, fmt.Errorf("%s not installed: run tools install: %w", p.Name, err)
		}
		if strings.TrimSpace(string(marker)) != packageDigest(p) {
			return nil, fmt.Errorf("cached %s differs from pinned archive", p.Name)
		}
		for _, exe := range p.Executables {
			path := filepath.Join(root, exe)
			info, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
				return nil, fmt.Errorf("invalid cached executable %s", path)
			}
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				paths = append(paths, dir)
			}
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no supported tools for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return paths, nil
}
