package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	OS          string   `json:"os"`
	Arch        string   `json:"arch"`
	URL         string   `json:"url"`
	SHA256      string   `json:"sha256"`
	Executables []string `json:"executables"`
}

func (p Package) Validate() error {
	safe := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	if !safe.MatchString(p.Name) || !safe.MatchString(p.Version) {
		return fmt.Errorf("invalid package identity")
	}
	if p.OS != "linux" || (p.Arch != "amd64" && p.Arch != "arm64") {
		return fmt.Errorf("unsupported package platform")
	}
	if len(p.Executables) == 0 {
		return fmt.Errorf("package has no executables")
	}
	for _, path := range p.Executables {
		if !filepath.IsLocal(path) || path == "." {
			return fmt.Errorf("invalid executable path %s", path)
		}
	}
	return nil
}
func InstallPackage(ctx context.Context, cache string, p Package) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
		return "", fmt.Errorf("package platform differs from host")
	}
	parent := filepath.Join(cache, p.Name)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	target := filepath.Join(parent, p.Version+"-"+p.OS+"-"+p.Arch)
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("package already installed: %s", target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	staging, err := os.MkdirTemp(parent, ".install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	archivePath := filepath.Join(staging, "package.tar.gz")
	if err = Download(ctx, p.URL, p.SHA256, archivePath); err != nil {
		return "", err
	}
	content := filepath.Join(staging, "content")
	if err = os.Mkdir(content, 0700); err != nil {
		return "", err
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	extractErr := ExtractTarGzip(archive, content)
	archive.Close()
	if extractErr != nil {
		return "", extractErr
	}
	for _, path := range p.Executables {
		info, err := os.Lstat(filepath.Join(content, path))
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return "", fmt.Errorf("missing executable permission: %s", path)
		}
	}
	if err = os.WriteFile(filepath.Join(content, ".archive-sha256"), []byte(p.SHA256+"\n"), 0600); err != nil {
		return "", err
	}
	if err = os.Rename(content, target); err != nil {
		return "", err
	}
	return target, nil
}
