# Changelog

## v1.0.0-alpha.4

- Default CLI and Action mutation parallelism to `jobs: auto`; use Go runtime CPU/container settings and cap workers at the mutant count. Explicit positive worker counts remain supported.
- Exclude jobs from measurement freshness; `baseline-update` no longer needs the measurement’s worker setting. Rerun `qa` once after upgrading to refresh old reports; alpha.3 baselines remain valid.
- Add a minimal `make`-driven workflow template in `example/.github/workflows/qa.yml`.

## v1.0.0-alpha.3

- Group coverage and mutation methodology settings; validate the complete baseline offline with JSON Schema.
- Download upstream Bats 1.14.0 and ShellCheck 0.11.0; reuse PATH kcov or unpack Debian kcov 43 with shared libraries without sudo. Stop building and publishing third-party tool archives.
- Rename `baseline update` to `baseline-update`; add `quality-bash-fix` to the minimal makefile example.

- Use system Bash >= 5.2; check its version before installing managed tools. Bash is no longer downloaded or added from the managed cache.

- `install.sh` selects the newest published CLI, verifies the release asset SHA-256, installs managed tools and runs doctor. Requires jq; download, installation or verification errors fail the command.

- Baselines use only the required `$schema` URL; the schema lives at `baseline.schema.json` in the repository and CLI archive.
- `mutation.score` replaces killed/survived counts in baselines: a percentage rounded half up to two decimal places for storage and comparison. `coverage.linePercent` uses the same rounding policy. Reports and Action outputs retain exact counts.
- Remove old alpha baseline readers and `--upgrade-schema`. Recreate old baselines from fresh measurements and review their thresholds.

## v1.0.0-alpha.2

- `install.sh` installs the pinned Linux CLI with SHA-256 verification; the runnable make example includes explicit installation, lint, QA and baseline targets.

- Help for the CLI and each command works without installed tools or a project; help exits successfully. Unknown commands are rejected before tool lookup (issue #4).
- Baseline v2 uses named coverage/mutation counts and offline JSON Schema validation; duplicate keys are rejected. Explicit schema conversion preserves thresholds; jobs/timeout no longer change baseline compatibility.
- Documentation split into user operations (`readme.md`), development (`develop.md`) and contribution workflow (`contributing.md`).

## v1.0.0-alpha.1

- First Linux amd64/arm64 CLI and composite Action: lint, Bats, kcov coverage, AST mutation testing and exact baseline gates.
- Explicit checksum-verified tool installation without Go or sudo; signed release and MIT license.
- Limits: trusted project code only; unmodified kcov 43 may mix coverage for byte-identical scripts.

[Releases and validation links](https://github.com/ever-guild/quality-bash/releases)
