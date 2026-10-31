package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateOutputPaths(root, reports, baseline string, s Selection) error {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	absolute := func(path string) string {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		return filepath.Clean(path)
	}
	root = absolute(root)
	reports = absolute(reports)
	baseline = absolute(baseline)
	for _, path := range []string{reports, baseline, baseline + ".lock", baseline + ".migration.json"} {
		if err := rejectSymlinkComponents(path); err != nil {
			return err
		}
	}
	inside := func(parent, path string) bool {
		rel, err := filepath.Rel(parent, path)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if inside(reports, root) {
		return fmt.Errorf("reports directory cannot contain project root")
	}
	if inside(reports, baseline) {
		return fmt.Errorf("baseline must be outside reports directory")
	}
	selected := append(append([]string{}, s.Scripts...), s.Tests...)
	for _, name := range selected {
		path := absolute(name)
		if inside(reports, path) || path == baseline || path == baseline+".lock" || path == baseline+".migration.json" {
			return fmt.Errorf("output location overlaps selected input %s", name)
		}
	}
	return nil
}

func rejectSymlinkComponents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(absolute, current), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path contains unsupported symlink: %s", current)
		}
	}
	return nil
}
