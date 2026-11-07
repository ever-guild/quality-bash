# quality-bash

Check Bash syntax, ShellCheck diagnostics, Bats tests, line coverage and mutation score. `qa` fails if either quality metric falls below the committed baseline.

## Install

Download the Linux amd64 or arm64 archive and `SHA256SUMS` from [Releases](https://github.com/ever-guild/quality-bash/releases). Ubuntu 24.04 is tested on both architectures. No Go or sudo is needed.

```sh
sha256sum --ignore-missing --check SHA256SUMS  # require OK for your archive
# Extract the matching quality-bash-<version>-linux-<architecture>.tar.gz
./quality-bash --help
./quality-bash tools install
./quality-bash doctor
```

`--help` works without tools starting with alpha.2. Installation explicitly downloads pinned Bash, Bats, ShellCheck and unmodified kcov 43 into the user cache. Checks never install dependencies automatically.

## Check a project

Put matching `.sh` and `.bats` files in the project root. Multiple pairs and test helpers are supported; no extra configuration or makefile is required.

```sh
./quality-bash qa --root /path/to/project
# First run: a missing baseline fails AFTER measurements. Then:
./quality-bash baseline update --root /path/to/project --create
./quality-bash qa --root /path/to/project
```

Commit `quality-baseline.json`. `qa` and `baseline check` run fresh checks without changing it. Use `baseline update` after an improvement; neither threshold can decrease. Use identical root, reports, jobs and timeout settings when measuring and updating.

Run individual checks with `lint`, `test`, `coverage` or `mutation`. Repeat `--script` and `--test` for explicit paths relative to `--root`; all selected tests run against each mutant. Other options: `--reports`, `--baseline`, `--jobs`, `--timeout`, `--base-revision <commit>`. Diagnostics, HTML, Cobertura and JSON reports default to `quality-reports/`.

## Baseline v2 (alpha.2)

`coverage.coveredLines` / `coverage.totalLines` and `mutation.killed` / (`killed` + `survived`) define the minimum ratios. Comparisons use exact counts, without rounded percentages. The file also records scope and measurement methodology. The embedded [JSON Schema](internal/quality/schema/baseline-v2.schema.json) validates it offline; unknown fields, duplicate keys and invalid counts fail.

```sh
./quality-bash baseline update --root /path/to/project --upgrade-schema
```

This converts v1 to v2 without tools or changing thresholds and saves a migration record. v1 remains readable for checks and target-revision comparison. A methodology/scope change requires fresh measurements and `baseline update --migrate 'reason'`. This is separate from format conversion; compatible thresholds cannot use migration to decrease.

## GitHub Action

```yaml
- uses: actions/checkout@v7
  with:
    fetch-depth: 0
- uses: ever-guild/quality-bash@v1.0.0-alpha.2
```

PRs compare against the target revision by default. The Action supplies summary metrics, outputs and diagnostic artifacts, using a fresh tool cache. Ordinary checks need no secrets or write permissions.

## Measurement and limits

- Coverage counts selected Bash lines; it does not measure statements inside awk/jq. Empty reports or absent tracing fail. kcov 43 can mix coverage for byte-identical files: do not rely on metrics for that case.
- Mutation operators: `[[ ]]` `==`/`!=`, `-eq`/`-ne`, `-lt`/`-ge`, `-gt`/`-le`, `&&`/`||`; binary arithmetic `+`/`-`, `*`→`+`, `<`/`>=`, `>`/`<=`, `==`/`!=`; literal `exit`/`return` `0`/`1`. Strings, comments and embedded languages are not mutated.
- Invalid mutants are reported but excluded from the score. No valid mutants, timeouts or infrastructure errors fail qualification.
- Run only trusted scripts, tests and helpers. They execute as your user; temporary copies are not sandboxes and cannot prevent deliberate falsification. Review fork code; isolate mutually untrusted workloads. Timeout cleanup kills process groups but cannot contain deliberate escapes.
- Symlinks, special project files and commas in selected coverage paths are unsupported. Tool versions affect baseline compatibility; jobs and timeout affect report freshness, not baseline compatibility in v2.

[Changes](changelog.md) · [Development](develop.md) · [Contributing](contributing.md)
