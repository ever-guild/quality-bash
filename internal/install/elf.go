package install

import (
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SharedLibraries resolves the complete DT_NEEDED closure without executing
// the inspected binary. The caller supplies the build sysroot search paths.
func SharedLibraries(binary string, search []string) ([]string, error) {
	seen := map[string]bool{}
	var result []string
	var visit func(string) error
	visit = func(path string) error {
		file, err := elf.Open(path)
		if err != nil {
			return err
		}
		names, err := file.ImportedLibraries()
		file.Close()
		if err != nil {
			return err
		}
		for _, name := range names {
			if filepath.Base(name) != name {
				return fmt.Errorf("nonportable ELF dependency %s", name)
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			found := ""
			for _, dir := range search {
				candidate := filepath.Join(dir, name)
				info, err := os.Stat(candidate)
				if err == nil && info.Mode().IsRegular() {
					found = candidate
					break
				}
			}
			if found == "" {
				return fmt.Errorf("unresolved ELF dependency %s from %s", name, path)
			}
			result = append(result, found)
			if err = visit(found); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(binary); err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}
