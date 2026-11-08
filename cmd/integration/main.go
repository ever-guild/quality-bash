// integration runs the test suite with verified real tools and rejects skips.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	ctx := context.Background()
	if err := quality.CheckBash(ctx, 15*time.Second); err != nil {
		return err
	}
	manifest, err := install.ParseManifest(install.BundledManifest)
	if err != nil {
		return err
	}
	manifest, err = install.PreferSystemKcov(ctx, manifest)
	if err != nil {
		return err
	}
	cache, err := install.CacheRoot()
	if err != nil {
		return err
	}
	paths, err := install.InstalledPaths(cache, manifest)
	if err != nil {
		return err
	}
	if err := os.Setenv("PATH", strings.Join(paths, string(os.PathListSeparator))+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		return err
	}
	if _, err := quality.Doctor(ctx, 15*time.Second, os.Stdout); err != nil {
		return err
	}
	cmd := exec.Command("go", "test", "-json", "-race", "-count=1", "./...")
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	skipped := false
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var event struct{ Action, Output, Test string }
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}
		fmt.Print(event.Output)
		if event.Action == "skip" && event.Test != "" {
			skipped = true
		}
	}
	scanErr := scanner.Err()
	err = cmd.Wait()
	if err != nil {
		return err
	}
	if scanErr != nil {
		return scanErr
	}
	if skipped {
		return fmt.Errorf("integration suite skipped tests")
	}
	return nil
}
