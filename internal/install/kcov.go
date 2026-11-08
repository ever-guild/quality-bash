package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

func packageDigest(p Package) string {
	if p.Format != "deb" {
		return p.SHA256
	}
	data, _ := json.Marshal(p)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func packageRoot(cache string, p Package) string {
	version := p.Version + "-" + p.OS + "-" + p.Arch
	if p.Format == "deb" {
		version += "-" + packageDigest(p)[:12]
	}
	return filepath.Join(cache, p.Name, version)
}

// Only data files are unpacked. Debian maintainer scripts are never executed.
func extractDeb(reader io.Reader, destination string) error {
	var magic [8]byte
	if _, err := io.ReadFull(reader, magic[:]); err != nil {
		return err
	}
	if string(magic[:]) != "!<arch>\n" {
		return fmt.Errorf("invalid Debian archive")
	}
	for entries := 0; entries < 64; entries++ {
		var header [60]byte
		if _, err := io.ReadFull(reader, header[:]); err != nil {
			return fmt.Errorf("missing Debian payload: %w", err)
		}
		if string(header[58:]) != "`\n" {
			return fmt.Errorf("invalid ar header")
		}
		name := strings.TrimSuffix(strings.TrimSpace(string(header[:16])), "/")
		size, err := strconv.ParseInt(strings.TrimSpace(string(header[48:58])), 10, 64)
		if err != nil || size < 0 || size > maxArchive {
			return fmt.Errorf("invalid ar entry size")
		}
		member := io.LimitReader(reader, size)
		if strings.HasPrefix(name, "data.tar") {
			var unpacked io.Reader
			switch name {
			case "data.tar.xz":
				unpacked, err = (xz.ReaderConfig{DictCap: 64 << 20}).NewReader(member)
			case "data.tar.gz":
				var gz *gzip.Reader
				gz, err = gzip.NewReader(member)
				if err == nil {
					defer gz.Close()
					unpacked = gz
				}
			case "data.tar":
				unpacked = member
			default:
				return fmt.Errorf("unsupported Debian compression %s", name)
			}
			if err != nil {
				return err
			}
			return extractDebPayload(unpacked, destination)
		}
		if _, err := io.CopyN(io.Discard, member, size); err != nil {
			return err
		}
		if size%2 != 0 {
			if _, err := io.CopyN(io.Discard, reader, 1); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("too many ar entries")
}

func extractDebPayload(reader io.Reader, destination string) error {
	archive := tar.NewReader(&archiveBudgetReader{reader: reader, remaining: maxArchive})
	for count := 0; count < maxArchiveEntries; count++ {
		h, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if !filepath.IsLocal(name) || len(name) > 4096 || strings.Count(name, "/") > 32 {
			return fmt.Errorf("unsafe Debian path %q", h.Name)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		} // Never materialize symlinks, hardlinks or devices.
		if name != "usr/bin/kcov" && !strings.HasPrefix(name, "usr/lib/") && !strings.HasPrefix(name, "lib/") && !strings.HasPrefix(name, "usr/share/doc/") {
			continue
		}
		if h.Size < 0 || h.Size > maxArchive {
			return fmt.Errorf("oversized Debian file")
		}
		target := filepath.Join(destination, name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if h.Mode&0111 != 0 {
			mode = 0755
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(f, archive, h.Size)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return fmt.Errorf("too many Debian files")
}

// Copy each real library under its ELF SONAME, avoiding archive symlinks entirely.
func collectLibraries(root, libraryDir string) error {
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		f, err := elf.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		names, err := f.DynString(elf.DT_SONAME)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return nil
		}
		name := names[0]
		if filepath.Base(name) != name || name == "." || name == ".." {
			return fmt.Errorf("unsafe library SONAME")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(libraryDir, name)
		if previous, err := os.ReadFile(target); err == nil {
			if !bytes.Equal(previous, data) {
				return fmt.Errorf("conflicting library %s", name)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// libc and the ELF loader remain supplied by the host. Everything else in the
// DT_NEEDED closure must be present in the cache, never borrowed from the host.
func verifyKcovLibraries(binary, libraryDir string) error {
	host := map[string]bool{"libc.so.6": true, "libm.so.6": true, "libdl.so.2": true, "libpthread.so.0": true, "librt.so.1": true, "libresolv.so.2": true, "ld-linux-x86-64.so.2": true, "ld-linux-aarch64.so.1": true}
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(path string) error {
		f, err := elf.Open(path)
		if err != nil {
			return err
		}
		names, err := f.ImportedLibraries()
		f.Close()
		if err != nil {
			return err
		}
		for _, name := range names {
			if filepath.Base(name) != name {
				return fmt.Errorf("unsafe ELF dependency %s", name)
			}
			if host[name] || seen[name] {
				continue
			}
			seen[name] = true
			if err := visit(filepath.Join(libraryDir, name)); err != nil {
				return fmt.Errorf("incomplete kcov libraries (%s): %w", name, err)
			}
		}
		return nil
	}
	return visit(binary)
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
func kcovWrapper(root string) []byte {
	return []byte("#!/bin/sh\n# quality-bash managed kcov\nLD_LIBRARY_PATH=" + shellQuote(filepath.Join(root, "lib")) + "${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH} exec " + shellQuote(filepath.Join(root, "root/usr/bin/kcov")) + " \"$@\"\n")
}
func installKcov(ctx context.Context, cache string, p Package) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if p.Name != "kcov" || p.OS != runtime.GOOS || p.Arch != runtime.GOARCH {
		return "", fmt.Errorf("unsupported Debian tool/platform")
	}
	target := packageRoot(cache, p)
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("package already installed")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(parent, ".install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	root := filepath.Join(staging, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		return "", err
	}
	artifacts := append([]Artifact{{Name: p.Name, URL: p.URL, SHA256: p.SHA256}}, p.Dependencies...)
	for i, a := range artifacts {
		archive := filepath.Join(staging, fmt.Sprintf("%d.deb", i))
		if err := Download(ctx, a.URL, a.SHA256, archive); err != nil {
			return "", fmt.Errorf("download %s: %w", a.Name, err)
		}
		file, err := os.Open(archive)
		if err != nil {
			return "", err
		}
		extractErr := extractDeb(file, root)
		file.Close()
		if extractErr != nil {
			return "", fmt.Errorf("extract %s: %w", a.Name, extractErr)
		}
		if err := os.Remove(archive); err != nil {
			return "", err
		}
	}
	libs := filepath.Join(staging, "lib")
	binary := filepath.Join(root, "usr/bin/kcov")
	if err := collectLibraries(root, libs); err != nil {
		return "", err
	}
	if err := verifyKcovLibraries(binary, libs); err != nil {
		return "", err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(checkCtx, binary, "--version")
	command.Env = append(os.Environ(), "LD_LIBRARY_PATH="+libs)
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "kcov 43+dfsg-1~bpo12+1" {
		return "", fmt.Errorf("kcov cannot run on this host (glibc Linux required): %s: %v", output, err)
	}
	if err := os.Mkdir(filepath.Join(staging, "bin"), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, "bin/kcov"), kcovWrapper(target), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, ".archive-sha256"), []byte(packageDigest(p)+"\n"), 0600); err != nil {
		return "", err
	}
	if err := os.Rename(staging, target); err != nil {
		return "", err
	}
	return target, nil
}

func exposeKcov(root string) error {
	bin := os.Getenv("XDG_BIN_HOME")
	if bin == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		bin = filepath.Join(home, ".local/bin")
	}
	if err := os.MkdirAll(bin, 0755); err != nil {
		return err
	}
	path := filepath.Join(bin, "kcov")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace existing %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(data, []byte("#!/bin/sh\n# quality-bash managed kcov\n")) {
			return fmt.Errorf("refusing to replace existing %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(bin, ".kcov-")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	_, writeErr := temp.Write(kcovWrapper(root))
	modeErr := temp.Chmod(0755)
	closeErr := temp.Close()
	if writeErr != nil {
		return writeErr
	}
	if modeErr != nil {
		return modeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temp.Name(), path)
}
