package quality

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	Selection         Selection `json:"selection"`
	Method            Method    `json:"method"`
	Timeout           string    `json:"timeout"`
	MutationExecution string    `json:"mutationExecution"`
}

func (r Runner) Settings(ctx context.Context, s Selection) (Settings, error) {
	versions, err := Doctor(ctx, r.Timeout, nil)
	if err != nil {
		return Settings{}, err
	}
	tools := map[string]string{}
	for _, tool := range versions {
		tools[tool.Name] = tool.Version
	}
	return Settings{MutationExecution: "bats-abort-v1", Selection: s, Timeout: r.Timeout.String(), Method: Method{
		Coverage: CoverageMethod{Algorithm: "kcov-ps4-cobertura-lines-v1"}, Tools: tools, Mutation: MutationMethodSettings{Algorithm: MutationMethod,
			Operators: []string{"condition", "arithmetic", "return-code"},
			Parser:    Parser{Name: "mvdan.cc/sh/v3", Version: "v3.12.0"}},
	}}, nil
}
func (r Runner) Measure(ctx context.Context, s Selection, jobs int, baselinePath string) (Measurement, error) {
	var m Measurement
	if err := ValidateOutputPaths(r.Root, r.Reports, baselinePath, s); err != nil {
		return m, err
	}
	if err := os.MkdirAll(r.Reports, 0755); err != nil {
		return m, err
	}
	// A failed new run must not leave an earlier report eligible for updates.
	if err := WriteJSON(filepath.Join(r.Reports, "measurement.json"), map[string]bool{"qualified": false}); err != nil {
		return m, err
	}
	settings, err := r.Settings(ctx, s)
	if err != nil {
		return m, err
	}
	exclusions := []string{r.Reports, baselinePath, baselinePath + ".lock", baselinePath + ".migration.json"}
	before, err := Fingerprint(r.Root, exclusions, settings)
	if err != nil {
		return m, err
	}
	if err = r.Lint(ctx, s); err != nil {
		return m, err
	}
	if err = r.Test(ctx, s); err != nil {
		return m, err
	}
	coverage, err := r.Coverage(ctx, s)
	if err != nil {
		return m, err
	}
	mutation, err := r.Mutate(ctx, s, jobs)
	if err != nil {
		return m, err
	}
	after, err := Fingerprint(r.Root, exclusions, settings)
	if err != nil {
		return m, err
	}
	if before != after {
		return m, fmt.Errorf("project changed during measurement; report cannot qualify")
	}
	m = Measurement{CoverageCounts: coverage.Score, MutationCounts: mutation.Score, Fingerprint: after, Qualified: true, Baseline: Baseline{Files: s.Scripts, Method: settings.Method, Coverage: coverage.Score, Mutation: mutation.Score}}
	if err = WriteJSON(filepath.Join(r.Reports, "measurement.json"), m); err != nil {
		return m, err
	}
	return m, nil
}
func (r Runner) LoadFresh(ctx context.Context, s Selection, baselinePath string) (Measurement, error) {
	var m Measurement
	if err := ValidateOutputPaths(r.Root, r.Reports, baselinePath, s); err != nil {
		return m, err
	}
	if err := ReadJSON(filepath.Join(r.Reports, "measurement.json"), &m); err != nil {
		return m, err
	}
	settings, err := r.Settings(ctx, s)
	if err != nil {
		return m, err
	}
	fingerprint, err := Fingerprint(r.Root, []string{r.Reports, baselinePath, baselinePath + ".lock", baselinePath + ".migration.json"}, settings)
	if err != nil {
		return m, err
	}
	return m, m.ValidateFresh(fingerprint)
}
