package main

const usage = `Usage: quality-bash <command> [options]

Commands:
  version          Print the CLI version
  doctor           Check installed tool versions
  tools install    Install pinned tools into the user cache
  lint             Run Bash syntax checks and ShellCheck
  test             Run Bats tests
  coverage         Measure selected Bash line coverage
  mutation         Run AST mutation tests
  qa               Run all checks and enforce the baseline
  baseline check   Run fresh checks and enforce the baseline
  baseline update  Save fresh qualified results without lowering thresholds

Common options:
  --root PATH      Project directory (default .)
  --script PATH    Select a Bash file (repeatable)
  --test PATH      Select a Bats file (repeatable)
  --reports PATH   Reports directory relative to root
  --baseline PATH Baseline file relative to root
  --jobs N         Parallel mutation workers
  --timeout TIME   Timeout per external process, e.g. 30s

Use quality-bash <command> --help for options and baseline operations.
Help and version work without installed tools or project files.`

var commandHelp = map[string]string{
	"doctor":          "Check the presence, versions and compatibility of installed tools.",
	"tools install":   "Download and verify pinned tools into the user cache; no sudo required.",
	"lint":            "Check Bash syntax and run ShellCheck on selected scripts.",
	"test":            "Run all selected Bats tests and retain process diagnostics.",
	"coverage":        "Measure selected Bash lines with kcov; save HTML, Cobertura and JSON reports.",
	"mutation":        "Run original tests, then isolated AST mutants; timeouts and infrastructure errors fail the check.",
	"qa":              "Run lint, tests, coverage and mutation testing, then enforce the baseline without modifying it.",
	"baseline check":  "Run fresh quality checks and enforce the working and optional target-revision baselines.",
	"baseline update": "Save fresh qualified measurements. Use --create for the first baseline, --upgrade-schema for lossless v1 to v2 conversion, or --migrate REASON for an incompatible methodology. Compatible thresholds cannot decrease.",
}
