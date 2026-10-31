package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"path/filepath"
	"runtime"
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
	directory := flag.String("directory", "tool-dist", "tool archives")
	flag.Parse()
	temp, err := os.MkdirTemp("", "quality-bash-bundle-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	var paths []string
	for _, name := range []string{"bash", "bats", "shellcheck", "kcov"} {
		root := filepath.Join(temp, name)
		if err = os.Mkdir(root, 0700); err != nil {
			return err
		}
		file, err := os.Open(filepath.Join(*directory, name+"-linux-"+runtime.GOARCH+".tar.gz"))
		if err != nil {
			return err
		}
		extractErr := install.ExtractTarGzip(file, root)
		file.Close()
		if extractErr != nil {
			return extractErr
		}
		paths = append(paths, filepath.Join(root, "bin"))
	}
	env := []string{"PATH=" + strings.Join(paths, string(os.PathListSeparator)) + string(os.PathListSeparator) + os.Getenv("PATH")}
	result := quality.RunProcess(context.Background(), 10*time.Minute, "", env, os.Stdout, os.Stderr, "go", "test", "-count=1", "-v", "./internal/quality")
	if result.ExitCode != 0 || result.TimedOut || result.Cancelled || result.InfrastructureError != "" {
		return fmt.Errorf("packaged tool checks failed: %+v", result)
	}
	return nil
}
