# quality-bash

Bash quality checks in one command: syntax, ShellCheck, Bats tests, coverage and mutation testing, fail when coverage or mutation score drops below the committed baseline

## Install

Linux amd64/arm64, Bash ≥ 5.2 and glibc ≥ 2.36; Ubuntu 24.04 is tested, requires curl, jq, tar

```bash
curl -fsSL --proto '=https' \
  https://raw.githubusercontent.com/ever-guild/quality-bash/main/install.sh \
  | bash
```

Installs the newest published release, downloads its tools and runs `doctor`, run again to upgrade, bash must already be installed

## Check your project

Run from your project root with `.sh` scripts and `.bats` tests:

```bash
quality-bash qa
# First run reports a missing baseline after measuring, create it:
quality-bash baseline-update --create
```

Commit `quality-baseline.json`. After improving tests, run `qa`, then `baseline-update` to raise the thresholds, updating never lowers them, keep the same root, reports and timeout settings for both commands

Reports: `quality-reports/`, mutation parallelism is automatic, use `--jobs 1` for serial execution or `--jobs N` to choose, for individual checks and other options, run `quality-bash --help`

## Make and CI

Copy [example/makefile](example/makefile) into your project:

```sh
make quality-bash                         # install if missing, then check
make quality-bash-fix ARGS=--create       # first baseline, after measuring
make quality-bash-fix                     # update baseline after improvements
make quality-bash ARGS='--timeout 3m'
```

Copy [example/.github/workflows/qa-via-make.yml](example/.github/workflows/qa-via-make.yml) to `.github/workflows/qa.yml` to run the same make command in CI, it checks the checked-out baseline

Alternatively, copy [qa-via-action.yml](example/.github/workflows/qa-via-action.yml) to `.github/workflows/qa.yml` to use our Action, which also checks the target branch’s baseline on PRs. Choose one workflow to avoid duplicate checks.

## Limits

- Run only trusted scripts, tests and helpers: they execute as your user, without a sandbox
- Coverage measures Bash lines, not code inside awk/jq, kcov may mix coverage for byte-identical files
- After successful control runs, mutants stop at the first Bats failure, including setup/teardown; timeouts, runner errors and malformed reports still fail the check
- Symlinks, special files and commas in coverage paths are unsupported
