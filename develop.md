# Development

Linux, Go 1.27+ and GNU make. CI uses Go 1.27.2 on Ubuntu 24.04 amd64/arm64.

## Build and check

```sh
make check                       # race tests, vet, build
make ci-deps                     # Ubuntu development dependencies; uses sudo
make kcov                        # pinned, unmodified upstream kcov 43
make integration                 # real tool tests; inspect skips
make version
```

Use `KCOV_PREFIX=/path/to/kcov` to reuse an installation. Default: `.build/kcov` locally, `$RUNNER_TEMP/kcov` in CI. `GO` selects the Go executable. Missing external tools cause integration tests to skip; skipped coverage/mutation tests do not qualify a release.

## Where to edit

| Path | Responsibility |
| --- | --- |
| `cmd/quality-bash` | CLI and help |
| `internal/quality` | Checks, measurements, gates, baseline migration |
| `internal/quality/schema` | Embedded JSON Schema; keep fixtures valid |
| `internal/install` | Downloads, archive validation, pinned tool manifest |
| `cmd/package-release` | Reproducible CLI archives |
| `testdata/basic` | Known coverage 3/5 and mutation score 2/3 |

Own logic and tests are Go. Shell fixtures are inputs; `install.sh` only downloads and installs the pinned CLI. Workflow `run` steps call only `makefile` targets. Keep execution settings in report fingerprints; changing a gate's methodology must remain explicit. See [contribution workflow](contributing.md) for human and agent changes.

## Package and release

```sh
make bundle-tools
make verify-tools
make release TOOL_DIST=/path/to/both-architectures RELEASE_DIST=/tmp/release
```

Build/verify native tools on both architectures. Packaging requires all eight archives matching the embedded manifest. CLI archives include all four guides, baseline schema, project/dependency licenses and tool manifest.

1. Update the CLI/package version, Action URLs, make targets and release smoke reference. Keep dependency versions pinned; update notices for any new dependency.
2. Build archives and update both SHA-256 pins in `action.yml`, `makefile` and `install.sh`. Bundled documentation changes archive hashes too.
3. Run Codex Security CLI on the final revision, retain its report, resolve in-scope findings and re-audit subsequent changes. Use the trusted-code boundary in the user guide.
4. Accumulate changes, sign commits, push once and require native CI on both architectures. Create a signed tag and publish the new prerelease with archives, `SHA256SUMS`, limitations and changelog. Never replace an existing release's tag/assets.
5. Run `.github/workflows/release-smoke.yml` once. It downloads the published CLI and Action on both architectures and compares metrics. Record successful CI/smoke links in release notes.

Current packaging targets alpha.2. Local prepublication installation is available through `cmd/install-local-tools`; packaging alone is not release qualification.
