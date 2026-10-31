package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type MutationReport struct {
	Mutations []Mutation `json:"mutations"`
	Score     Ratio      `json:"score"`
	Qualified bool       `json:"qualified"`
}

func copyProject(root, dest, reports string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" || path == reports {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported copy source: %s", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
func (r Runner) Mutate(ctx context.Context, s Selection, jobs int) (MutationReport, error) {
	report := MutationReport{}
	if jobs < 1 {
		return report, fmt.Errorf("jobs must be positive")
	}
	if err := os.MkdirAll(r.Reports, 0755); err != nil {
		return report, err
	}
	runReports, err := os.MkdirTemp(r.Reports, "mutation-")
	if err != nil {
		return report, err
	}
	workerRunner := r
	workerRunner.Reports = runReports
	if err := r.Test(ctx, s); err != nil {
		return report, fmt.Errorf("original tests failed: %w", err)
	}
	// Verify the exact copied layout before attributing failures to mutations.
	// Excluded reports or Git metadata must not supply required test helpers.
	control, err := os.MkdirTemp("", "quality-bash-control-")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(control)
	if err := copyProject(r.Root, control, r.Reports); err != nil {
		return report, err
	}
	controlRunner := workerRunner
	controlRunner.Root = control
	if err := controlRunner.Test(ctx, s); err != nil {
		return report, fmt.Errorf("unmodified tests fail in the mutation copy; qualification blocked: %w", err)
	}
	for _, file := range s.Scripts {
		data, err := os.ReadFile(filepath.Join(r.Root, file))
		if err != nil {
			return report, err
		}
		mutations, err := GenerateMutations(file, string(data))
		if err != nil {
			return report, err
		}
		report.Mutations = append(report.Mutations, mutations...)
	}
	if len(report.Mutations) == 0 {
		return report, fmt.Errorf("no applicable mutations; mutation score is undefined")
	}
	queue := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < jobs; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range queue {
				m := &report.Mutations[index]
				m.Result = workerRunner.runMutant(ctx, s, *m, r.Reports)
			}
		}()
	}
	for index := range report.Mutations {
		queue <- index
	}
	close(queue)
	wg.Wait()
	report.Qualified = true
	for _, m := range report.Mutations {
		switch m.Result {
		case "detected":
			report.Score.Numerator++
			report.Score.Denominator++
		case "survived":
			report.Score.Denominator++
		case "invalid":
		default:
			report.Qualified = false
		}
	}
	if report.Score.Denominator == 0 {
		report.Qualified = false
	}
	if err = WriteJSON(filepath.Join(runReports, "mutation.json"), report); err != nil {
		return report, err
	}
	if err = WriteJSON(filepath.Join(r.Reports, "mutation.json"), report); err != nil {
		return report, err
	}
	if !report.Qualified {
		return report, fmt.Errorf("mutation qualification blocked by timeout, infrastructure failure, or absence of valid mutants")
	}
	return report, nil
}
func (r Runner) runMutant(ctx context.Context, s Selection, m Mutation, excludedReports string) string {
	dir, err := os.MkdirTemp("", "quality-bash-mutant-")
	if err != nil {
		return "infrastructure-error"
	}
	defer os.RemoveAll(dir)
	if err = copyProject(r.Root, dir, excludedReports); err != nil {
		return "infrastructure-error"
	}
	path := filepath.Join(dir, m.File)
	source, err := os.ReadFile(path)
	if err != nil {
		return "infrastructure-error"
	}
	changed, err := m.Apply(string(source))
	if err != nil {
		return "infrastructure-error"
	}
	if err = os.WriteFile(path, []byte(changed), 0600); err != nil {
		return "infrastructure-error"
	}
	logs := filepath.Join(r.Reports, "mutants", m.ID)
	if err = os.MkdirAll(logs, 0755); err != nil {
		return "infrastructure-error"
	}
	out, err := os.Create(filepath.Join(logs, "stdout.log"))
	if err != nil {
		return "infrastructure-error"
	}
	defer out.Close()
	stderr, err := os.Create(filepath.Join(logs, "stderr.log"))
	if err != nil {
		return "infrastructure-error"
	}
	defer stderr.Close()
	classify := func(p ProcessResult) string {
		if p.TimedOut {
			return "timeout"
		}
		if p.Cancelled || p.InfrastructureError != "" || p.ExitCode < 0 {
			return "infrastructure-error"
		}
		return ""
	}
	syntax := RunProcess(ctx, r.Timeout, dir, nil, io.Discard, stderr, "bash", "-n", "--", path)
	if status := classify(syntax); status != "" {
		return status
	}
	if syntax.ExitCode != 0 {
		return "invalid"
	}
	args := []string{"--formatter", "tap"}
	for _, test := range s.Tests {
		args = append(args, filepath.Join(dir, test))
	}
	result := RunProcess(ctx, r.Timeout, dir, nil, out, stderr, "bats", args...)
	data, _ := json.MarshalIndent(result, "", "  ")
	if os.WriteFile(filepath.Join(logs, "result.json"), data, 0644) != nil {
		return "infrastructure-error"
	}
	if status := classify(result); status != "" {
		return status
	}
	if err := out.Sync(); err != nil {
		return "infrastructure-error"
	}
	tap, err := os.Open(filepath.Join(logs, "stdout.log"))
	if err != nil {
		return "infrastructure-error"
	}
	_, tapErr := ValidateTAP(tap, result.ExitCode)
	tap.Close()
	if tapErr != nil {
		return "infrastructure-error"
	}
	if result.ExitCode == 0 {
		return "survived"
	}
	if result.ExitCode == 1 {
		return "detected"
	}
	return "infrastructure-error"
}
