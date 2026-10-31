# quality-bash

Go CLI for Bash linting, Bats tests, line coverage, mutation testing and quality regression gates. This repository is under development; v1.0.0-alpha.1 has not been released. The Action currently references the planned release and is not ready for consumption.

## Local development

Go 1.27 or newer is required to build. Released binaries will not require Go. Install Bash, Bats, ShellCheck and kcov explicitly; ordinary checks never install dependencies. The automatic tool installer is still being completed.

```console
go build -buildvcs=false -o quality-bash ./cmd/quality-bash
./quality-bash doctor
./quality-bash lint --root testdata/basic
./quality-bash test --root testdata/basic
./quality-bash coverage --root testdata/basic
./quality-bash mutation --root testdata/basic --jobs 2
./quality-bash qa --root testdata/basic --jobs 2
```

The default project has matching `example.sh` and `example.bats` files plus `quality-baseline.json`. Multiple pairs are supported. Unmatched Bats files require explicit selection. Repeat `--script` and `--test` for explicit paths; all selected tests apply to each mutant. Paths are relative to `--root`. Use `--reports`, `--baseline`, `--jobs` and `--timeout` to control execution. Reports may be outside the project.

## Baseline lifecycle

Run `qa` to produce fresh measurements. If the baseline is absent, the command fails after measurement; create it explicitly with `baseline update --create`, using the same root, reports, jobs and timeout arguments. Then run `qa` again. `qa` never modifies the baseline.

`baseline check` and `baseline update` require a qualified report matching current sources, tests, helper files and settings. Updates cannot lower either ratio. Ratios use exact integer comparison, without percentage rounding. Use `--base-revision <commit>` to enforce the target revision's baseline as well as the working copy.

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

Tests execute with the current user's permissions. Temporary copies are not security sandboxes. Linux process groups are terminated on cancellation or timeout; programs deliberately escaping the process group are not contained. Symlinks and nonregular project files are currently unsupported. kcov's comma-separated path interface does not support commas in selected paths. Terminal, subprocess and command-substitution fixtures are tested, but tracing can expose extra collector diagnostics; general output equivalence is still under investigation.

The fixture baseline is tied to its recorded tool versions and settings. A different toolchain requires an explicit methodology migration. arm64 binaries cross-compile, but full arm64 qualification and published Action smoke tests remain outstanding.

## Tests and packaging

```console
go test ./...
go vet ./...
go run -buildvcs=false ./cmd/package-release
```

Integration tests skip when their external tools are absent. To qualify a release, all tools must be present and those tests must execute. The packager currently emits development archives in `dist`, with checksums and the Bash parser's BSD license. It does not publish a release.
