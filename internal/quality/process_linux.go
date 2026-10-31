package quality

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ProcessResult struct {
	ExitCode            int    `json:"exit_code"`
	TimedOut            bool   `json:"timed_out"`
	Cancelled           bool   `json:"cancelled"`
	InfrastructureError string `json:"infrastructure_error,omitempty"`
}

// RunProcess uses a dedicated process group. Tests execute with the user's
// privileges; process groups are lifecycle management, not a security sandbox.
func RunProcess(parent context.Context, timeout time.Duration, dir string, env []string, stdout, stderr io.Writer, name string, args ...string) ProcessResult {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	result := ProcessResult{ExitCode: -1}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	err := cmd.Start()
	if err != nil {
		result.InfrastructureError = err.Error()
		return result
	}
	err = cmd.Wait()
	// Also reap background work left by an otherwise successful parent.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
	result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) && !result.TimedOut && !result.Cancelled {
		result.InfrastructureError = err.Error()
	}
	return result
}
