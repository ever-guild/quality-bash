package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func ReadGitBaseline(ctx context.Context, root, path, revision string, timeout time.Duration) (Baseline, error) {
	var baseline Baseline
	if revision == "" || strings.HasPrefix(revision, "-") || strings.ContainsAny(revision, "\x00\n\r") {
		return baseline, fmt.Errorf("invalid base revision")
	}
	var resolved, diagnostics bytes.Buffer
	result := RunProcess(ctx, timeout, root, nil, &resolved, &diagnostics, "git", "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if result.ExitCode != 0 || result.InfrastructureError != "" || result.TimedOut || result.Cancelled {
		return baseline, fmt.Errorf("cannot resolve base revision: %s (%+v)", diagnostics.String(), result)
	}
	commit := strings.TrimSpace(resolved.String())
	if !regexp.MustCompile(`^[a-f0-9]{40,64}$`).MatchString(commit) {
		return baseline, fmt.Errorf("invalid resolved commit")
	}
	var top bytes.Buffer
	result = RunProcess(ctx, timeout, root, nil, &top, &diagnostics, "git", "rev-parse", "--show-toplevel")
	if result.ExitCode != 0 || result.InfrastructureError != "" || result.TimedOut || result.Cancelled {
		return baseline, fmt.Errorf("cannot determine repository root")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, err := filepath.Rel(strings.TrimSpace(top.String()), path)
	if err != nil {
		return baseline, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return baseline, fmt.Errorf("baseline is outside repository")
	}
	var data bytes.Buffer
	result = RunProcess(ctx, timeout, root, nil, &data, &diagnostics, "git", "show", commit+":"+filepath.ToSlash(rel))
	if result.ExitCode != 0 || result.InfrastructureError != "" || result.TimedOut || result.Cancelled {
		return baseline, fmt.Errorf("cannot read target baseline: %s", diagnostics.String())
	}
	decoder := json.NewDecoder(&data)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&baseline); err != nil {
		return baseline, err
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return baseline, fmt.Errorf("trailing data in target baseline")
	}
	return baseline, nil
}
