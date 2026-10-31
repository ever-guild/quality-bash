package main

import (
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	binary := flag.String("binary", "", "external ELF executable")
	launcher := flag.String("launcher", "", "Go launcher executable")
	output := flag.String("output", "", "new package directory")
	search := flag.String("libraries", "/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:/lib64", "library search paths")
	name := flag.String("name", "", "tool name")
	flag.Parse()
	if *binary == "" || *launcher == "" || *output == "" || *name == "" || filepath.Base(*name) != *name {
		return fmt.Errorf("require --binary --launcher --output --name")
	}
	if err := os.Mkdir(*output, 0755); err != nil {
		return err
	}
	for _, dir := range []string{"bin", "libexec", "lib"} {
		if err := os.Mkdir(filepath.Join(*output, dir), 0755); err != nil {
			return err
		}
	}
	copyFile := func(source, target string, mode os.FileMode) error {
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, mode)
	}
	if err := copyFile(*launcher, filepath.Join(*output, "bin", *name), 0755); err != nil {
		return err
	}
	if err := copyFile(*binary, filepath.Join(*output, "libexec", *name), 0755); err != nil {
		return err
	}
	libs, err := install.SharedLibraries(*binary, strings.Split(*search, ":"))
	if err != nil {
		return err
	}
	for _, lib := range libs {
		if err = copyFile(lib, filepath.Join(*output, "lib", filepath.Base(lib)), 0755); err != nil {
			return err
		}
	}
	return nil
}
