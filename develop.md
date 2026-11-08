# Development

Linux, Go 1.27+ and GNU make. CI uses Go 1.27.2 on Ubuntu 24.04 amd64/arm64.

## Build and check

```sh
make check                       # race tests, vet, build
make ci-deps                     # download pinned ready tools; no sudo
make integration                 # real tool tests; inspect skips
make version
make -C example quality-bash-qa QUALITY_BASH="$PWD/quality-bash"
```

`GO` selects the Go executable. `make integration` uses the verified tool cache, requires doctor to pass and fails on any skipped test.

## Where to edit

| Path | Responsibility |
| --- | --- |
| `cmd/quality-bash` | CLI and help |
| `internal/quality` | Checks, measurements, gates, baseline migration |
| `baseline.schema.json`, `schema.go` | Embedded JSON Schema and CLI version; change the schema URL only for format changes |
| `internal/install` | Downloads, archive validation, pinned tool manifest |
| `cmd/package-release` | Reproducible CLI archives |
| `testdata/basic` | Known coverage 3/5 and mutation score 2/3 |

Own logic and tests are Go. Shell fixtures are inputs; `install.sh` selects the newest published CLI, verifies its archive, and delegates tool installation and verification to `tools install` and `doctor`. Workflow `run` steps call only `makefile` targets. Keep semantic execution settings (including timeout, excluding jobs) in report fingerprints; changing a gate's methodology must remain explicit. See [contribution workflow](contributing.md) for human and agent changes.

## Package and release

```sh
make release RELEASE_DIST=/tmp/release
```

Release archives contain only our CLI, guides, baseline schema, licenses and tool manifest. Third-party executables are downloaded by `tools install`, never built or published in our releases. Native CI must pass on amd64 and arm64.

1. Update the CLI/package version, Action URLs, make targets and release smoke reference. Keep dependency versions pinned; update notices for any new dependency.
2. Build archives and update both SHA-256 pins in `action.yml` and `makefile`. Bundled documentation changes archive hashes too.
3. Run Codex Security CLI on the final revision, retain its report, resolve in-scope findings and re-audit subsequent changes. Use the trusted-code boundary in the user guide.
4. Accumulate changes, sign commits, push once and require native CI on both architectures. Create a signed tag and publish the new release as Latest, with Pre-release disabled with archives, `SHA256SUMS`, limitations and changelog. Never replace an existing release's tag/assets.
5. Run `.github/workflows/release-smoke.yml` once. It downloads the published CLI and Action on both architectures and compares metrics. Record successful CI/smoke links in release notes.

Packaging alone is not release qualification.
