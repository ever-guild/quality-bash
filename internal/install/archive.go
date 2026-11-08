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
const maxArchiveEntries = 10000

// Unlike LimitReader, exhausting this budget is an error, never a clean EOF.
type archiveBudgetReader struct {
	reader    io.Reader
	remaining int64
}

func (r *archiveBudgetReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, fmt.Errorf("archive stream exceeds byte limit")
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	return n, err
}

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
	compressed, err := gzip.NewReader(&archiveBudgetReader{reader: reader, remaining: maxArchive})
	if err != nil {
		return err
	}
	defer compressed.Close()
	return extractTar(compressed, destination, omitDirectory, Package{})
}

func extractPackageTarGzip(reader io.Reader, destination string, p Package) error {
	compressed, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer compressed.Close()
	return extractTar(compressed, destination, "", p)
}

func extractTar(reader io.Reader, destination, omitDirectory string, p Package) error {
	archive := tar.NewReader(&archiveBudgetReader{reader: reader, remaining: maxArchive})
	var total int64
	entryCount, componentCount := 0, 0
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		entryCount++
		if entryCount > maxArchiveEntries {
			return fmt.Errorf("archive contains too many entries")
		}
		name := filepath.Clean(header.Name)
		depth := strings.Count(name, string(filepath.Separator)) + 1
		componentCount += depth
		if len(header.Name) > 4096 || depth > 32 || componentCount > 100000 {
			return fmt.Errorf("archive path complexity exceeds limit")
		}
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		if omitDirectory != "" && (name == omitDirectory || strings.HasPrefix(name, omitDirectory+string(filepath.Separator))) {
			continue
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if p.StripPrefix != "" {
			if name == p.StripPrefix {
				continue
			}
			if !strings.HasPrefix(name, p.StripPrefix+"/") {
				return fmt.Errorf("unexpected archive prefix %s", name)
			}
			name = strings.TrimPrefix(name, p.StripPrefix+"/")
		}
		if len(p.Include) != 0 {
			include := false
			for _, prefix := range p.Include {
				if name == prefix || strings.HasPrefix(name, prefix+"/") {
					include = true
					break
				}
			}
			if !include {
				continue
			}
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
			if header.Size < 0 || header.Size > maxArchive-total {
				return fmt.Errorf("expanded archive too large")
			}
			total += header.Size
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
