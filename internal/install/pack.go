package install

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

func PackDirectory(source, destination string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(destination), ".pack-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	hash := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(file, hash))
	archive := tar.NewWriter(gz)
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported package file %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		mode := int64(0644)
		if info.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		if err = archive.WriteHeader(&tar.Header{Name: filepath.ToSlash(rel), Mode: mode, Size: info.Size(), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(archive, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	archiveErr := archive.Close()
	gzipErr := gz.Close()
	closeErr := file.Close()
	for _, candidate := range []error{err, archiveErr, gzipErr, closeErr} {
		if candidate != nil {
			return "", candidate
		}
	}
	if err = os.Rename(file.Name(), destination); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
