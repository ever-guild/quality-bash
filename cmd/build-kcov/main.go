package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	prefix := flag.String("prefix", "", "installation prefix")
	flag.Parse()
	if *prefix == "" {
		return fmt.Errorf("--prefix required")
	}
	abs, err := filepath.Abs(*prefix)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "quality-bash-build-kcov-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	archive := filepath.Join(dir, "source.tar.gz")
	if err = install.Download(ctx, "https://codeload.github.com/SimonKagstrom/kcov/tar.gz/refs/tags/v43", "4cbba86af11f72de0c7514e09d59c7927ed25df7cebdad087f6d3623213b95bf", archive); err != nil {
		return err
	}
	source := filepath.Join(dir, "source")
	if err = os.Mkdir(source, 0700); err != nil {
		return err
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	err = install.ExtractTarGzip(file, source)
	file.Close()
	if err != nil {
		return err
	}
	cmake, err := exec.LookPath("cmake")
	if err != nil {
		return err
	}
	commands := [][]string{{"-S", filepath.Join(source, "kcov-43"), "-B", filepath.Join(dir, "build"), "-DCMAKE_BUILD_TYPE=Release", "-DCMAKE_INSTALL_PREFIX=" + abs}, {"--build", filepath.Join(dir, "build"), "-j2"}, {"--install", filepath.Join(dir, "build")}}
	for _, args := range commands {
		result := quality.RunProcess(ctx, 15*time.Minute, dir, nil, os.Stdout, os.Stderr, cmake, args...)
		if result.ExitCode != 0 || result.TimedOut || result.Cancelled || result.InfrastructureError != "" {
			return fmt.Errorf("cmake failed: %+v", result)
		}
	}
	return nil
}
