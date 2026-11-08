package qualitybash

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionReportIsolation(t *testing.T) {
	data, err := os.ReadFile("action.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	step := strings.Split(strings.Split(text, "    - name: Resolve report path\n")[1], "    - name: Download release binary")[0]
	block := strings.SplitN(step, "      run: |\n", 2)[1]
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "        ")
	}
	script := strings.Join(lines, "\n")
	if !strings.Contains(text, "QB_REPORTS: ${{ steps.reports.outputs.path }}") || !strings.Contains(text, "if: always() && steps.reports.outcome == 'success'") {
		t.Fatal("CLI/upload must use only successfully isolated reports")
	}
	for _, tc := range []struct {
		name, reports string
		valid         bool
	}{
		{"normal", "quality-reports", true}, {"nested", "reports/nested", true},
		{"absolute", "/tmp", false}, {"parent", "../outside", false}, {"inner parent", "a/../reports", false},
		{"glob", "reports/*", false}, {"pattern", "[ab]", false}, {"newline", "reports\nother", false}, {"trailing newline", "reports\n", false}, {"carriage return", "reports\r", false}, {"empty", "", false}, {"root", ".", false}, {"project symlink", "escape/reports", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			runnerTemp := t.TempDir()
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(root, "quality-reports")
			if err := os.Mkdir(parent, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(parent, "previous-secret"), []byte("previous"), 0600); err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for i := 0; i < 2; i++ {
				output := filepath.Join(t.TempDir(), "output")
				if err := os.WriteFile(output, nil, 0600); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command("bash", "-eu", "-o", "pipefail", "-c", script)
				cmd.Env = append(os.Environ(), "QB_ROOT="+root, "RUNNER_TEMP="+runnerTemp, "QB_REPORTS="+tc.reports, "GITHUB_OUTPUT="+output)
				diagnostic, err := cmd.CombinedOutput()
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v: %v: %s", tc.valid, err, diagnostic)
				}
				result, _ := os.ReadFile(output)
				if !tc.valid {
					if len(result) != 0 {
						t.Fatal("invalid path exported")
					}
					continue
				}
				path := strings.TrimSpace(strings.TrimPrefix(string(result), "path="))
				if !strings.HasPrefix(path, runnerTemp+"/") || seen[path] {
					t.Fatal("reports not isolated", path)
				}
				seen[path] = true
				entries, err := os.ReadDir(path)
				if err != nil || len(entries) != 0 {
					t.Fatal("uploaded pre-existing files", entries, err)
				}
			}
		})
	}
}
