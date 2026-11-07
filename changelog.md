# Changelog

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
