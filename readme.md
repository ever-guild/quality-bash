# quality-bash

Check Bash syntax, ShellCheck diagnostics, Bats tests, line coverage and mutation score. `qa` fails if either quality metric falls below the committed baseline.

## Install

Linux amd64/arm64; Ubuntu 24.04 is tested. Install the newest published CLI and its tools without Go or sudo. The installer requires Bash, curl, jq, tar, sha256sum and install:

```sh
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
  https://raw.githubusercontent.com/ever-guild/quality-bash/main/install.sh -o install.sh
bash install.sh
export PATH="${XDG_BIN_HOME:-$HOME/.local/bin}:$PATH"
quality-bash --help
```

The installer selects the newest release by publication date (including published prereleases), verifies the archive against the SHA-256 in GitHub release metadata, and installs into `XDG_BIN_HOME` (default `$HOME/.local/bin`). It then runs `quality-bash tools install` and `quality-bash doctor`; any failed step returns an error. Running it again installs the newest published version. Alternatively download an archive and `SHA256SUMS` from [Releases](https://github.com/ever-guild/quality-bash/releases), verify with `sha256sum --ignore-missing --check SHA256SUMS` and extract it.

`--help` works without tools. Direct CLI checks never install dependencies automatically.

Bash >= 5.2 must already be in PATH. Bats and ShellCheck come from pinned upstream archives. A compatible kcov in PATH is reused; otherwise Debian kcov 43 and its shared libraries are unpacked into `${XDG_CACHE_HOME:-$HOME/.cache}/quality-bash/kcov`, with a wrapper in `${XDG_BIN_HOME:-$HOME/.local/bin}`. This fallback requires glibc >= 2.36; musl systems need their own working kcov. No sudo, system package installation or compilation is performed.

## Check a project

Put matching `.sh` and `.bats` files in the project root. Multiple pairs and test helpers are supported; no extra configuration or makefile is required.

```sh
quality-bash qa --root /path/to/project
# First run: a missing baseline fails AFTER measurements. Then:
quality-bash baseline-update --root /path/to/project --create
quality-bash qa --root /path/to/project
```

Commit `quality-baseline.json`. `qa` and `baseline check` run fresh checks without changing it. Use `baseline-update` after an improvement; neither threshold can decrease. Use identical root, reports and timeout settings when measuring and updating.

Run individual checks with `lint`, `test`, `coverage` or `mutation`. Repeat `--script` and `--test` for explicit paths relative to `--root`; all selected tests run against each mutant. Other options: `--reports`, `--baseline`, `--jobs`, `--timeout`, `--base-revision <commit>`. Diagnostics, HTML, Cobertura and JSON reports default to `quality-reports/`.

`--jobs auto` is the default: mutation workers use Go’s current [`GOMAXPROCS`](https://pkg.go.dev/runtime#GOMAXPROCS), whose defaults account for CPU affinity and container quotas, capped at the number of mutants. Go may use two workers even with a quota below two CPUs; use `--jobs 1` for strict serial execution. `--jobs N` overrides auto. Lint, normal tests and coverage remain sequential; changing jobs does not invalidate measurements.

## Make example

The repository's [example](https://github.com/ever-guild/quality-bash/tree/v1.0.0-alpha.4/example) includes a script, test and baseline:

```sh
make -C example quality-bash-install
make -C example quality-bash
```

Copy `example/makefile` into your project for the same commands. On a new project, run `make quality-bash` to measure, use `make quality-bash-fix` after improvements. `ARGS='--jobs 2'` passes CLI options. The make targets bootstrap a missing CLI, direct CLI checks never install tools.

## Baseline

The baseline format uses only `$schema`, pointing to `https://raw.githubusercontent.com/ever-guild/quality-bash/<version>/baseline.schema.json`. The embedded [JSON Schema](baseline.schema.json) validates offline; missing/unknown fields, duplicate keys and invalid values fail.

`coverage.linePercent` is the minimum percentage of covered selected lines: total covered lines / total executable lines × 100. `mutation.score` is the minimum percentage of valid mutants detected: killed / (killed + survived) × 100. Both are rounded half up to two decimal places before storage and comparison; 2/3 becomes 66.67. Differences hidden by rounding do not fail the gate. Reports and Action outputs retain the exact counts.

`method.coverage.algorithm` identifies the collector. `method.mutation` groups the algorithm, operators and parser. `method.tools` records exact tool versions. These fields are generated automatically; changing them requires an explicit methodology migration.

The format is unchanged from alpha.3, so its schema URL and baselines remain valid in alpha.4. Older formats are not supported. To adopt this format, move the old baseline outside the project, run fresh `qa` measurements, then `baseline-update --create` with identical settings. Review the new thresholds before committing. Target-revision checks also require the current format.

Scope and methodology must match. Changing them requires fresh measurements and `baseline-update --migrate 'reason'`; compatible thresholds cannot use migration to decrease.

## GitHub Action

```yaml
- uses: actions/checkout@v7
  with:
    fetch-depth: 0
- uses: ever-guild/quality-bash@v1.0.0-alpha.4
```

PRs compare against the target revision by default. The Action supplies summary metrics, outputs and diagnostic artifacts, using a fresh tool cache. Each Action invocation stores and uploads diagnostics from a fresh directory under `RUNNER_TEMP`, using `reports` as a relative subdirectory; absolute paths, traversal and upload patterns are rejected. Ordinary checks need no secrets or write permissions.

## Measurement and limits

- Coverage counts selected Bash lines; it does not measure statements inside awk/jq. Empty reports or absent tracing fail. kcov 43 can mix coverage for byte-identical files: do not rely on metrics for that case.
- Mutation operators: `[[ ]]` `==`/`!=`, `-eq`/`-ne`, `-lt`/`-ge`, `-gt`/`-le`, `&&`/`||`; binary arithmetic `+`/`-`, `*`→`+`, `<`/`>=`, `>`/`<=`, `==`/`!=`; literal `exit`/`return` `0`/`1`. Strings, comments and embedded languages are not mutated.
- Invalid mutants are reported but excluded from the score. No valid mutants, timeouts or infrastructure errors fail qualification.
- Run only trusted scripts, tests and helpers. They execute as your user; temporary copies are not sandboxes and cannot prevent deliberate falsification. Review fork code; isolate mutually untrusted workloads. Timeout cleanup kills process groups but cannot contain deliberate escapes.
- Symlinks, special project files and commas in selected coverage paths are unsupported. Tool versions affect baseline compatibility; timeout affects report freshness, not baseline compatibility.

[Changes](changelog.md) · [Development](develop.md) · [Contributing](contributing.md)
