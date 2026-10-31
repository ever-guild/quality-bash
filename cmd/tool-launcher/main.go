// tool-launcher is copied under each external tool's name. Package layout:
// bin/<tool> (this launcher), libexec/<tool> (real executable), lib/ (ELF libs).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func main() {
	if err := launch(); err != nil {
		fmt.Fprintln(os.Stderr, "quality-bash tool launcher:", err)
		os.Exit(126)
	}
}
func launch() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	name := filepath.Base(executable)
	root := filepath.Dir(filepath.Dir(executable))
	target := filepath.Join(root, "libexec", name)
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("invalid packaged executable %s", target)
	}
	// Bats is a Bash program, and its own helpers remain in libexec/libexec.
	if name == "bats" {
		bash, err := exec.LookPath("bash")
		if err != nil {
			return err
		}
		return syscall.Exec(bash, append([]string{bash, target}, os.Args[1:]...), os.Environ())
	}
	loaderName := ""
	switch runtime.GOARCH {
	case "amd64":
		loaderName = "ld-linux-x86-64.so.2"
	case "arm64":
		loaderName = "ld-linux-aarch64.so.1"
	default:
		return fmt.Errorf("unsupported architecture %s", runtime.GOARCH)
	}
	loader := filepath.Join(root, "lib", loaderName)
	args := []string{loader, "--library-path", filepath.Join(root, "lib"), target}
	args = append(args, os.Args[1:]...)
	return syscall.Exec(loader, args, os.Environ())
}
