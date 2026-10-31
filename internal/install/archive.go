package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxArchive = 512 << 20

func Download(ctx context.Context, url, digest, destination string) error {
	expected, err := hex.DecodeString(digest)
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf("invalid pinned SHA-256")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("download requires HTTPS")
	}
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("insecure redirect")
		}
		if len(via) > 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download HTTP status %d", response.StatusCode)
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxArchive+1))
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxArchive {
		return fmt.Errorf("archive too large")
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), digest) {
		return fmt.Errorf("archive SHA-256 mismatch")
	}
	return os.Rename(file.Name(), destination)
}

// ExtractTarGzip accepts only regular files and directories. Links, devices,
// traversal and duplicate files are rejected, including links to other entries.
func ExtractTarGzip(reader io.Reader, destination string) error {
	return ExtractSourceTarGzip(reader, destination, "")
}

// ExtractSourceTarGzip can omit an upstream-only directory during a source
// build. Runtime package installation always uses ExtractTarGzip instead.
func ExtractSourceTarGzip(reader io.Reader, destination, omitDirectory string) error {
	info, err := os.Lstat(destination)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("extraction destination must be a real directory")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("extraction destination must be empty")
	}
	compressed, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	var total int64
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(header.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		if omitDirectory != "" && (name == omitDirectory || strings.HasPrefix(name, omitDirectory+string(filepath.Separator))) {
			continue
		}
		path := filepath.Join(destination, name)
		switch header.Typeflag {
		case tar.TypeXGlobalHeader:
			// Global PAX records carry archive metadata, not filesystem entries.
			continue
		case tar.TypeDir:
			if err = os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			total += header.Size
			if header.Size < 0 || total > maxArchive {
				return fmt.Errorf("expanded archive too large")
			}
			if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			mode := os.FileMode(0644)
			if header.Mode&0111 != 0 {
				mode = 0755
			}
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(file, archive, header.Size)
			modeErr := file.Chmod(mode)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if modeErr != nil {
				return modeErr
			}
		default:
			return fmt.Errorf("unsupported archive entry type for %q", header.Name)
		}
	}
}
