package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	output := flag.String("output", "tool-dist", "output directory")
	kcov := flag.String("kcov", "", "built kcov executable")
	flag.Parse()
	if *kcov == "" {
		return fmt.Errorf("--kcov required")
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "quality-bash-bundle-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	command := func(args ...string) error {
		cmd := exec.Command("go", args...)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	launcher := filepath.Join(temp, "launcher")
	if err = command("build", "-buildvcs=false", "-o", launcher, "./cmd/tool-launcher"); err != nil {
		return err
	}
	triplet := "x86_64-linux-gnu"
	if runtime.GOARCH == "arm64" {
		triplet = "aarch64-linux-gnu"
	}
	var sums strings.Builder
	versions := map[string]string{}
	patterns := map[string]string{"bash": `version (5\.2\.21[^\s]*)`, "bats": `Bats (1\.10\.0)(?:\s|$)`, "shellcheck": `version: (0\.9\.0)(?:\s|$)`, "kcov": `kcov (43)(?:\s|$)`}
	for _, name := range []string{"bash", "bats", "shellcheck", "kcov"} {
		binaryPath := filepath.Join("/usr/bin", name)
		if name == "kcov" {
			binaryPath = *kcov
		}
		versionOutput, versionErr := exec.Command(binaryPath, "--version").CombinedOutput()
		if versionErr != nil {
			return fmt.Errorf("version check %s: %w", name, versionErr)
		}
		match := regexp.MustCompile(patterns[name]).FindStringSubmatch(string(versionOutput))
		if len(match) != 2 {
			return fmt.Errorf("unexpected %s version: %s", name, versionOutput)
		}
		versions[name] = match[1]
		root := filepath.Join(temp, name)
		if name == "bats" {
			for _, dir := range []string{"bin", "lib", "libexec"} {
				if err = os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
					return err
				}
			}
			data, err := os.ReadFile("/usr/bin/bats")
			if err != nil {
				return err
			}
			if err = os.WriteFile(filepath.Join(root, "bin/bats"), data, 0755); err != nil {
				return err
			}
			for _, dir := range []string{"lib", "libexec"} {
				if err = install.CopyTree(filepath.Join("/usr", dir, "bats-core"), filepath.Join(root, dir, "bats-core")); err != nil {
					return err
				}
			}
		} else {
			binary := filepath.Join("/usr/bin", name)
			if name == "kcov" {
				binary = *kcov
			}
			if err = command("run", "-buildvcs=false", "./cmd/package-tool", "--binary", binary, "--launcher", launcher, "--output", root, "--name", name, "--libraries", "/lib/"+triplet+":/usr/lib/"+triplet+":/lib64:/lib"); err != nil {
				return err
			}
		}
		// Preserve distribution copyright notices, including bundled library notices.
		licenses := filepath.Join(root, "licenses")
		if err = os.MkdirAll(licenses, 0755); err != nil {
			return err
		}
		if name == "kcov" {
			upstream := filepath.Join(filepath.Dir(filepath.Dir(*kcov)), "share", "licenses", "kcov")
			if err = install.CopyTree(upstream, filepath.Join(licenses, "kcov")); err != nil {
				return err
			}
		}
		entries, err := os.ReadDir("/usr/share/doc")
		if err != nil {
			return err
		}
		for _, entry := range entries {
			source := filepath.Join("/usr/share/doc", entry.Name(), "copyright")
			data, readErr := os.ReadFile(source)
			if readErr != nil {
				continue
			}
			if err = os.WriteFile(filepath.Join(licenses, entry.Name()+".txt"), data, 0644); err != nil {
				return err
			}
		}
		common := filepath.Join(licenses, "common")
		if err = os.MkdirAll(common, 0755); err != nil {
			return err
		}
		commonEntries, readErr := os.ReadDir("/usr/share/common-licenses")
		if readErr != nil {
			return readErr
		}
		for _, entry := range commonEntries {
			data, readErr := os.ReadFile(filepath.Join("/usr/share/common-licenses", entry.Name()))
			if readErr != nil {
				return readErr
			}
			if err = os.WriteFile(filepath.Join(common, entry.Name()), data, 0644); err != nil {
				return err
			}
		}
		archive := name + "-linux-" + runtime.GOARCH + ".tar.gz"
		sum, err := install.PackDirectory(root, filepath.Join(*output, archive))
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%s  %s\n", sum, archive)
	}
	if err = os.WriteFile(filepath.Join(*output, "SHA256SUMS"), []byte(sums.String()), 0644); err != nil {
		return err
	}
	metadata, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*output, "versions-"+runtime.GOARCH+".json"), metadata, 0644)
}
