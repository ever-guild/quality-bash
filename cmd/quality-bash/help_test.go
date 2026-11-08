package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestHelpWithoutToolsOrProject(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for _, command := range []string{"", "version", "doctor", "tools", "tools install", "lint", "test", "coverage", "mutation", "qa", "baseline", "baseline check", "baseline-update"} {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(command+flag, func(t *testing.T) {
				out, err := os.CreateTemp(t.TempDir(), "help")
				if err != nil {
					t.Fatal(err)
				}
				previous := os.Stdout
				os.Stdout = out
				err = run(context.Background(), append(strings.Fields(command), flag))
				os.Stdout = previous
				out.Close()
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(out.Name())
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "Usage:") {
					t.Fatalf("missing help: %s", data)
				}
			})
		}
	}
	for _, args := range [][]string{{"unknown"}, {"qa", "--not-an-option"}, {"baseline-update", "--upgrade-schema"}, {"baseline", "update", "--help"}} {
		err := run(context.Background(), args)
		if err == nil || strings.Contains(err.Error(), "not installed") {
			t.Fatalf("wrong argument error: %v", err)
		}
	}
	err := run(context.Background(), nil)
	for _, command := range []string{"coverage", "qa", "baseline check", "baseline-update", "tools install"} {
		if err == nil || !strings.Contains(err.Error(), command) {
			t.Fatalf("incomplete usage: %v", err)
		}
	}
}
