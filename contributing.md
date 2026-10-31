# Contributing to quality-bash

Use Go 1.27 or newer, GNU make and Linux. CI uses Go 1.27.2 on Ubuntu 24.04 amd64 and arm64. End users install prebuilt binaries and do not need Go or make. User instructions are in [reamde.md](reamde.md).

## Project layout

- `cmd/quality-bash`: CLI commands and argument handling.
- `internal/quality`: process adapters, coverage, mutations, baselines and reports.
- `internal/install`: pinned downloads, archive validation and the tool cache.
- Other `cmd` packages: native tool building, packaging and release verification.
- `testdata/basic`: integration fixture with coverage 3/5 and mutation score 2/3.

Own application logic and infrastructure tests are Go. Bash and Bats files are test inputs; `makefile` and workflow YAML orchestrate the Go commands and external tools.

## Local checks

```console
make check
make version
make kcov
make integration
```

`make check` runs race-enabled tests, vet and the CLI build. Integration tests skip when tools are unavailable, so release qualification also requires the full integration target with Bash, Bats, ShellCheck and kcov present. `make ci-deps` installs Ubuntu build dependencies with sudo; use it only on a development or CI host. This is unrelated to the sudo-free end-user installer.

`KCOV_PREFIX` defaults to `$RUNNER_TEMP/kcov` in CI and `.build/kcov` locally. Set it to an existing kcov installation to avoid rebuilding. `GO` selects the Go executable. All workflow `run` steps call goals in `makefile`; keep executable command sequences out of workflow YAML. Third-party Actions use full commit SHAs, with monthly Dependabot proposals requiring review.

## Tool and release packaging

```console
make bundle-tools
make verify-tools
make release TOOL_DIST=/path/to/both-architectures RELEASE_DIST=/path/to/release
```

Build and verify native tool bundles on both architectures. The release target requires all eight tool archives in `TOOL_DIST` and checks them against the embedded manifest. It produces both CLI archives, copies the tool archives and writes combined `SHA256SUMS`. CLI archives include user documentation, Go and parser licenses, and the pinned tool manifest. Tool archives contain dependency license notices; kcov includes its pinned upstream source archive.

The underlying `cmd/package-release` command defaults to development archives when used without a release version. Before assets are public, `cmd/install-local-tools` can install the verified local archives for testing. Packaging alone does not qualify or publish a release.

## Audit and release gates

Run Codex Security CLI against the final revision before publishing. Retain the report and scope, resolve in-scope findings, validate fixes, and audit subsequent code or configuration changes. The accepted product boundary is trusted scripts, Bats tests and helpers, as documented in the user guide. Do not describe deliberately malicious test behavior as technically contained.

Verify native CI, signatures, archive digests and every specification requirement before publication. Commit changes with a GPG signature and Conventional Commits message. Accumulate changes before pushing. Publish the signed `v1.0.0-alpha.1` tag and prerelease assets only after the audit gate passes.

The CLI archive digests are pinned in `action.yml` and the `smoke-cli` target. Rebuilding an archive, including changes to its bundled user documentation, requires updating both pins before publication. Do not silently replace published assets.

After publication, run the separate release smoke workflow. It uses the published Action, runs `make smoke-metrics`, downloads and executes the standalone CLI through `make smoke-cli`, then compares both results using `make smoke-compare`. Both architectures must report coverage 3/5 and mutation score 2/3. Record the release and successful workflow links; do not claim completion while the published smoke test is outstanding.
