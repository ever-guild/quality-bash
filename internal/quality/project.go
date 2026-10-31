package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Selection struct {
	Scripts []string `json:"scripts"`
	Tests   []string `json:"tests"`
}

func Discover(root string, scripts, tests []string) (Selection, error) {
	var result Selection
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return result, err
	}
	automatic := len(scripts) == 0 && len(tests) == 0
	if len(scripts) == 0 {
		entries, err := os.ReadDir(root)
		if err != nil {
			return result, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
				scripts = append(scripts, entry.Name())
			}
		}
	}
	if len(scripts) == 0 {
		return result, fmt.Errorf("no Bash scripts selected in %s", root)
	}
	if len(tests) == 0 {
		for _, script := range scripts {
			tests = append(tests, strings.TrimSuffix(script, ".sh")+".bats")
		}
	}
	normalize := func(paths []string, extension string) ([]string, error) {
		seen := map[string]bool{}
		var normalized []string
		for _, name := range paths {
			if filepath.IsAbs(name) {
				var err error
				name, err = filepath.Rel(root, name)
				if err != nil {
					return nil, err
				}
			}
			name = filepath.Clean(name)
			if name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) || !strings.HasSuffix(name, extension) {
				return nil, fmt.Errorf("unsupported %s path: %s", extension, name)
			}
			if seen[name] {
				return nil, fmt.Errorf("duplicate selection: %s", name)
			}
			seen[name] = true
			info, err := os.Lstat(filepath.Join(root, name))
			if err != nil {
				return nil, fmt.Errorf("selected file %s: %w", name, err)
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("selected path must be a regular file: %s", name)
			}
			normalized = append(normalized, filepath.ToSlash(name))
		}
		sort.Strings(normalized)
		return normalized, nil
	}
	result.Scripts, err = normalize(scripts, ".sh")
	if err != nil {
		return result, err
	}
	result.Tests, err = normalize(tests, ".bats")
	if err == nil && automatic {
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			return result, readErr
		}
		matched := map[string]bool{}
		for _, name := range result.Tests {
			matched[name] = true
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bats") && !matched[entry.Name()] {
				return result, fmt.Errorf("unmatched Bats file %s: select scripts and tests explicitly", entry.Name())
			}
		}
	}
	return result, err
}

// Fingerprint includes all project files, including test helpers. Only explicitly
// designated output locations and Git metadata are excluded. Symlinks are
// rejected because external targets cannot be copied or fingerprinted reliably.
func Fingerprint(root string, excluded []string, settings any) (string, error) {
	h := sha256.New()
	encoded, err := json.Marshal(settings)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(h, "settings:%d:", len(encoded))
	h.Write(encoded)
	skip := map[string]bool{".git": true}
	for _, p := range excluded {
		if filepath.IsAbs(p) {
			p, err = filepath.Rel(root, p)
			if err != nil {
				return "", err
			}
			// An output outside the project is never encountered by WalkDir.
			if p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
				continue
			}
		}
		p = filepath.Clean(p)
		if p == "." || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid fingerprint exclusion: %s", p)
		}
		skip[p] = true
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if skip[rel] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported project file: %s", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		fmt.Fprintf(h, "file:%d:%s:%d:%d:", len(name), name, info.Mode().Perm(), len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
