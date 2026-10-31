package quality

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

type ToolVersion struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Compatible bool   `json:"compatible"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

// These are the minimum API versions understood by the process adapters.
// Exact installed versions are retained in measurement metadata.
func Doctor(ctx context.Context, timeout time.Duration, output io.Writer) ([]ToolVersion, error) {
	specs := []struct {
		name, pattern string
		minimum       []int
	}{
		{"bash", `version (\d+\.\d+[^\s]*)`, []int{5, 2}},
		{"bats", `Bats (\d+\.\d+\.\d+[^\s]*)`, []int{1, 10, 0}},
		{"shellcheck", `version: (\d+\.\d+\.\d+[^\s]*)`, []int{0, 9, 0}},
		{"kcov", `(?:kcov |version: )(\d+[^\s]*)`, []int{43}},
	}
	var versions []ToolVersion
	failed := false
	for _, spec := range specs {
		var buf bytes.Buffer
		result := RunProcess(ctx, timeout, "", nil, &buf, &buf, spec.name, "--version")
		tool := ToolVersion{Name: spec.name}
		match := regexp.MustCompile(spec.pattern).FindStringSubmatch(buf.String())
		if result.ExitCode != 0 || result.TimedOut || result.Cancelled || result.InfrastructureError != "" {
			tool.Diagnostic = fmt.Sprintf("cannot execute: %+v", result)
		} else if len(match) != 2 {
			tool.Diagnostic = "unrecognized version output"
		} else {
			tool.Version = match[1]
			numbers := regexp.MustCompile(`\d+`).FindAllString(tool.Version, -1)
			comparison := 0
			for i, min := range spec.minimum {
				value := 0
				if i < len(numbers) {
					value, _ = strconv.Atoi(numbers[i])
				}
				if value > min {
					comparison = 1
					break
				}
				if value < min {
					comparison = -1
					break
				}
			}
			tool.Compatible = comparison >= 0
			if !tool.Compatible {
				tool.Diagnostic = "version older than supported minimum"
			}
		}
		if !tool.Compatible {
			failed = true
		}
		if output != nil {
			fmt.Fprintf(output, "%s: version=%s compatible=%t %s\n", tool.Name, tool.Version, tool.Compatible, tool.Diagnostic)
		}
		versions = append(versions, tool)
	}
	if failed {
		return versions, fmt.Errorf("external tool checks failed; install supported tools explicitly")
	}
	return versions, nil
}
