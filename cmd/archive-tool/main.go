package main

import (
	"flag"
	"fmt"
	"github.com/ever-guild/quality-bash/internal/install"
	"os"
)

func main() {
	source := flag.String("source", "", "package directory")
	output := flag.String("output", "", "archive path")
	flag.Parse()
	if *source == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "require --source and --output")
		os.Exit(2)
	}
	sum, err := install.PackDirectory(*source, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s  %s\n", sum, *output)
}
