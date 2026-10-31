package quality

import (
	"fmt"
	"os"
)

// PublishGitHub writes only fixed keys and numeric values to runner-owned files.
func PublishGitHub(m Measurement, passed bool) error {
	values := fmt.Sprintf("coverage_numerator=%d\ncoverage_denominator=%d\nmutation_numerator=%d\nmutation_denominator=%d\npassed=%t\n", m.Baseline.Coverage.Numerator, m.Baseline.Coverage.Denominator, m.Baseline.Mutation.Numerator, m.Baseline.Mutation.Denominator, passed)
	summary := fmt.Sprintf("### quality-bash\n\n| Metric | Count |\n|---|---:|\n| Coverage | %d/%d |\n| Mutations detected | %d/%d |\n\nQuality gate passed: **%t**\n", m.Baseline.Coverage.Numerator, m.Baseline.Coverage.Denominator, m.Baseline.Mutation.Numerator, m.Baseline.Mutation.Denominator, passed)
	if !m.Qualified {
		values = "passed=false\n"
		summary = "### quality-bash\n\nQuality gate passed: **false**\n\nNo qualified measurement is available. See the command diagnostics and uploaded reports.\n"
	}
	for name, content := range map[string]string{"GITHUB_OUTPUT": values, "GITHUB_STEP_SUMMARY": summary} {
		path := os.Getenv(name)
		if path == "" {
			continue
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		_, writeErr := file.WriteString(content)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
