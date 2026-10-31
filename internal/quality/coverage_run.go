package quality

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (r Runner) Coverage(ctx context.Context, s Selection) (CoverageReport, error) {
	var report CoverageReport
	bats, err := exec.LookPath("bats")
	if err != nil {
		return report, err
	}
	if err = os.MkdirAll(r.Reports, 0755); err != nil {
		return report, err
	}
	output, err := os.MkdirTemp(r.Reports, "coverage-")
	if err != nil {
		return report, err
	}
	var includes []string
	for _, file := range s.Scripts {
		path := filepath.Join(r.Root, file)
		if strings.Contains(path, ",") {
			return report, fmt.Errorf("kcov does not support commas in selected paths: %s", path)
		}
		includes = append(includes, path)
	}
	args := []string{"--bash-method=PS4", "--bash-dont-parse-binary-dir", "--include-path=" + strings.Join(includes, ","), "--bash-parse-files-in-dir=" + r.Root, output, bats, "--formatter", "tap"}
	for _, test := range s.Tests {
		args = append(args, filepath.Join(r.Root, test))
	}
	if err = r.Execute(ctx, "kcov", "kcov", args...); err != nil {
		return report, err
	}
	xmlPath := filepath.Join(output, filepath.Base(bats), "cobertura.xml")
	file, err := os.Open(xmlPath)
	if err != nil {
		return report, fmt.Errorf("collector produced no Cobertura report: %w", err)
	}
	defer file.Close()
	report, err = ReadCoverage(file, r.Root, s.Scripts)
	if err != nil {
		return report, err
	}
	if _, err = os.Stat(filepath.Join(output, filepath.Base(bats), "index.html")); err != nil {
		return report, fmt.Errorf("collector produced no HTML report: %w", err)
	}
	if err = WriteJSON(filepath.Join(r.Reports, "coverage.json"), report); err != nil {
		return report, err
	}
	return report, nil
}
