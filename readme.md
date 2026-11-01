# quality-bash

Go CLI for Bash linting, Bats tests, line coverage, mutation testing and quality regression gates.

## Release installation

The `v1.0.0-alpha.1` release targets Linux amd64 and arm64. Download the matching `quality-bash-v1.0.0-alpha.1-linux-<architecture>.tar.gz` and `SHA256SUMS` from the [release page](https://github.com/ever-guild/quality-bash/releases/tag/v1.0.0-alpha.1). The commands below use amd64; substitute arm64 where appropriate. They become available when the release assets are published.

```console
sha256sum --ignore-missing --check SHA256SUMS
tar -xzf quality-bash-v1.0.0-alpha.1-linux-amd64.tar.gz
./quality-bash version
./quality-bash tools install
./quality-bash doctor
./quality-bash qa --root /path/to/project
```

Confirm that checksum verification reports `OK` for the chosen archive before extracting it. Go and sudo are not required. Keep the binary in a directory on your PATH, or invoke its absolute path. Tool downloads need HTTPS access to GitHub; subsequent checks use the user cache without network installation. Bundled native tools are qualified on Ubuntu 24.04 for both architectures. Other Linux distributions are not yet qualified.

## Local usage

Run `quality-bash tools install` explicitly to install the pinned Bash, Bats, ShellCheck and kcov packages into the user cache without sudo. Ordinary checks only use the installed packages and never download or update dependencies.

```console
./quality-bash tools install
./quality-bash doctor
./quality-bash lint --root /path/to/project
./quality-bash test --root /path/to/project
./quality-bash coverage --root /path/to/project
./quality-bash mutation --root /path/to/project --jobs 2
./quality-bash qa --root /path/to/project --jobs 2
```

## GitHub Action

After release publication, connect the Action as follows. Checkout history is needed for target-revision baseline comparisons; pull requests use their base commit by default.

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
- uses: ever-guild/quality-bash@v1.0.0-alpha.1
```

The default project has matching `example.sh` and `example.bats` files plus `quality-baseline.json`. Multiple pairs are supported. Unmatched Bats files require explicit selection. Repeat `--script` and `--test` for explicit paths; all selected tests apply to each mutant. Paths are relative to `--root`. Use `--reports`, `--baseline`, `--jobs` and `--timeout` to control execution. Reports may be outside the project.

Each Action invocation installs tools into its own fresh cache under the runner's temporary directory. It does not reuse the OS user's persistent tool cache across jobs. Local CLI caches belong to the current user; do not share an OS account or runner between mutually untrusted workloads.

## Baseline lifecycle

Run `qa` to produce fresh measurements. If the baseline is absent, the command fails after measurement; create it explicitly with `baseline update --create`, using the same root, reports, jobs and timeout arguments. Then run `qa` again. `qa` never modifies the baseline.

`baseline check` runs fresh checks and compares their in-memory results with the baseline, just like `qa`; it never trusts a saved measurement as evidence that tests passed. `baseline update` requires a qualified report matching current sources, tests, helper files and settings and is a local maintenance operation, not a CI gate. Updates cannot lower either ratio. Ratios use exact integer comparison, without percentage rounding. Use `--base-revision <commit>` to enforce the target revision's baseline as well as the working copy.

Changes to scope or methodology require `baseline update --migrate 'explanation'`. Migration writes the previous and replacement baselines to `quality-baseline.json.migration.json`. Compatible baselines cannot use migration to lower thresholds. Target-revision checks continue to reject incompatible methodology until the migration is accepted in the target branch.

## Measurement semantics

Coverage uses kcov's PS4 Bash tracing and Cobertura line data. Only selected scripts count; duplicate line records are merged by line number. Empty reports, missing selected scripts and absence of any hits fail. HTML and raw Cobertura remain in the unique coverage report directory. Bash-level coverage cannot measure branches or statements inside embedded awk or jq programs; their invocation is the relevant Bash statement.

The initial AST mutation set changes `[[ ... ]]` comparisons and boolean operators, selected binary arithmetic operators, and literal `exit` or `return` codes 0 and 1. It does not mutate comments, arbitrary strings, URLs or embedded awk/jq source. IDs depend on method, file, byte offset, operator and replacement. Every mutant runs in a temporary project copy after the original tests pass. A complete Bats TAP result is required. Invalid syntax is recorded separately; timeouts and infrastructure errors block qualification and never count as detected mutants.

The `bash-ast-v1` operators are fixed:

| AST context | Replacements |
|---|---|
| Binary `[[ ... ]]` tests | `==` ↔ `!=`, `-eq` ↔ `-ne`, `-lt` ↔ `-ge`, `-gt` ↔ `-le`, `&&` ↔ `\|\|` |
| Binary arithmetic | `+` ↔ `-`, `*` → `+`, `<` ↔ `>=`, `>` ↔ `<=`, `==` ↔ `!=` |
| Literal `exit` and `return` arguments | `0` ↔ `1` |

There are no operators for single-bracket external `test` commands, regex bodies, unary arithmetic or arbitrary numeric literals. Each replacement affects one AST operator or argument. The mutation ratio is detected / (detected + survived); invalid mutants are listed but excluded from that denominator. Zero valid mutants is an error. Any timeout or infrastructure failure invalidates the entire measurement, regardless of the ratio.

## Execution limitations

### Trusted project code

Only run scripts, Bats tests and helpers that you trust. The quality gate measures this trusted code and detects quality regressions; it does not protect measurement integrity against code deliberately trying to falsify results. A malicious test running as the same user can replace tools, temporarily rewrite source, or fabricate reports. Fingerprints, checksum verification and temporary copies do not prevent such same-user attacks, and mandatory test sandboxing is outside this release's scope.

Fork PR support means the Action requires no secrets or write permissions. It does not make arbitrary fork code safe to execute: review code before execution and apply your runner's isolation and trust policy. Keep mutually untrusted workloads on separate runners or OS accounts.

Tests execute with the current user's permissions. Temporary copies are not security sandboxes. Linux process groups are terminated on cancellation or timeout; programs deliberately escaping the process group are not contained. Symlinks and nonregular project files are currently unsupported. kcov's comma-separated path interface does not support commas in selected paths. Integration tests assert exact script stdout, stderr, exit status and pseudo-terminal output both with and without coverage. Collector diagnostics are retained separately in the command logs.

Baselines are tied to their recorded tool versions and settings. A different toolchain requires an explicit methodology migration.

For development, testing and release procedures, see [contributing.md](contributing.md).
