package quality

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
)

type Settings struct {
	Selection Selection `json:"selection"`
	Method    Method    `json:"method"`
	Jobs      int       `json:"jobs"`
	Timeout   string    `json:"timeout"`
}

func (r Runner) Settings(ctx context.Context, s Selection, jobs int) (Settings, error) {
	versions, err := Doctor(ctx, r.Timeout, nil)
	if err != nil {
		return Settings{}, err
	}
	tools := map[string]string{}
	for _, tool := range versions {
		tools[tool.Name] = tool.Version
	}
	return Settings{Selection: s, Jobs: jobs, Timeout: r.Timeout.String(), Method: Method{
		Coverage: "kcov-ps4-cobertura-lines-v1", Tools: tools, Mutation: MutationMethod,
		Operators:  []string{"condition", "arithmetic", "return-code"},
		Parameters: map[string]string{"jobs": strconv.Itoa(jobs), "timeout": r.Timeout.String(), "parser": "mvdan.cc/sh/v3@v3.12.0"},
	}}, nil
}
func (r Runner) Measure(ctx context.Context, s Selection, jobs int, baselinePath string) (Measurement, error) {
	var m Measurement
	if err := ValidateOutputPaths(r.Root, r.Reports, baselinePath, s); err != nil {
		return m, err
	}
	settings, err := r.Settings(ctx, s, jobs)
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
	m = Measurement{Fingerprint: after, Qualified: true, Baseline: Baseline{Schema: 1, Files: s.Scripts, Method: settings.Method, Coverage: coverage.Score, Mutation: mutation.Score}}
	if err = WriteJSON(filepath.Join(r.Reports, "measurement.json"), m); err != nil {
		return m, err
	}
	return m, nil
}
func (r Runner) LoadFresh(ctx context.Context, s Selection, jobs int, baselinePath string) (Measurement, error) {
	var m Measurement
	if err := ValidateOutputPaths(r.Root, r.Reports, baselinePath, s); err != nil {
		return m, err
	}
	if err := ReadJSON(filepath.Join(r.Reports, "measurement.json"), &m); err != nil {
		return m, err
	}
	settings, err := r.Settings(ctx, s, jobs)
	if err != nil {
		return m, err
	}
	fingerprint, err := Fingerprint(r.Root, []string{r.Reports, baselinePath, baselinePath + ".lock", baselinePath + ".migration.json"}, settings)
	if err != nil {
		return m, err
	}
	return m, m.ValidateFresh(fingerprint)
}
