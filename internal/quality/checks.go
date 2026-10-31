package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Runner struct {
	Root    string
	Reports string
	Timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
}

func (r Runner) Execute(ctx context.Context, label, name string, args ...string) error {
	if err := os.MkdirAll(r.Reports, 0755); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(r.Reports, label+"-")
	if err != nil {
		return err
	}
	out, err := os.Create(filepath.Join(dir, "stdout.log"))
	if err != nil {
		return err
	}
	defer out.Close()
	stderr, err := os.Create(filepath.Join(dir, "stderr.log"))
	if err != nil {
		return err
	}
	defer stderr.Close()
	stdoutWriter, stderrWriter := io.Writer(out), io.Writer(stderr)
	if r.Stdout != nil {
		stdoutWriter = io.MultiWriter(out, r.Stdout)
	}
	if r.Stderr != nil {
		stderrWriter = io.MultiWriter(stderr, r.Stderr)
	}
	result := RunProcess(ctx, r.Timeout, r.Root, nil, stdoutWriter, stderrWriter, name, args...)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "result.json"), data, 0644); err != nil {
		return err
	}
	if label == "bats" && result.ExitCode == 0 {
		if err = out.Sync(); err != nil {
			return err
		}
		tap, openErr := os.Open(filepath.Join(dir, "stdout.log"))
		if openErr != nil {
			return openErr
		}
		_, tapErr := ValidateTAP(tap, result.ExitCode)
		tap.Close()
		if tapErr != nil {
			return fmt.Errorf("invalid Bats result: %w; diagnostics: %s", tapErr, dir)
		}
	}
	if result.ExitCode != 0 || result.TimedOut || result.Cancelled || result.InfrastructureError != "" {
		return fmt.Errorf("%s failed: exit=%d timeout=%t cancelled=%t infrastructure=%s; diagnostics: %s", label, result.ExitCode, result.TimedOut, result.Cancelled, result.InfrastructureError, dir)
	}
	return nil
}
func (r Runner) Lint(ctx context.Context, s Selection) error {
	for _, script := range s.Scripts {
		if err := r.Execute(ctx, "syntax", "bash", "-n", "--", filepath.Join(r.Root, script)); err != nil {
			return err
		}
	}
	args := []string{"--shell=bash", "--"}
	for _, script := range s.Scripts {
		args = append(args, filepath.Join(r.Root, script))
	}
	return r.Execute(ctx, "shellcheck", "shellcheck", args...)
}
func (r Runner) Test(ctx context.Context, s Selection) error {
	args := []string{"--formatter", "tap"}
	for _, test := range s.Tests {
		args = append(args, filepath.Join(r.Root, test))
	}
	return r.Execute(ctx, "bats", "bats", args...)
}
