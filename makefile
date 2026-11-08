SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := check

GO ?= go
RELEASE_DIST ?= dist
export RELEASE_DIST

.PHONY: check test integration vet build version ci-deps check-example release smoke-metrics smoke-cli smoke-compare smoke-example

check: test vet build

test:
	$(GO) test -race ./...

integration:
	$(GO) run ./cmd/integration

vet:
	$(GO) vet ./...

build:
	$(GO) build -trimpath -o quality-bash ./cmd/quality-bash

version:
	./quality-bash version

ci-deps: build
	./quality-bash tools install
	./quality-bash doctor

check-example: build
	$(MAKE) -C example quality-bash-qa QUALITY_BASH="$(CURDIR)/quality-bash" ARGS='--jobs 2'

release:
	$(GO) run ./cmd/package-release --version v1.0.0-alpha.3 --output "$$RELEASE_DIST"

smoke-metrics:
	test "$$COVERED/$$LINES" = '3/5'
	test "$$DETECTED/$$MUTANTS" = '2/3'
	test "$$PASSED" = 'true'

smoke-cli:
	case "$$RUNNER_ARCH_VALUE" in \
	  X64) arch=amd64; digest=38f25ff8d62072cfb381e8cc084b7165fd47b7c8ca0ead5a8f156cd47e0469bf ;; \
	  ARM64) arch=arm64; digest=7cef9a574ae2c119f046c8808446e82a0b60b667fef7997afa172adb46f7acf2 ;; \
	  *) exit 1 ;; \
	esac; \
	directory=$$(mktemp -d "$$RUNNER_TEMP/quality-bash-local.XXXXXXXX"); \
	trap 'rm -rf -- "$$directory"' EXIT; \
	export XDG_CACHE_HOME="$$directory/cache"; \
	release=https://github.com/ever-guild/quality-bash/releases/download/v1.0.0-alpha.3; \
	asset="quality-bash-v1.0.0-alpha.3-linux-$$arch.tar.gz"; \
	curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' "$$release/$$asset" -o "$$directory/$$asset"; \
	(cd "$$directory"; printf '%s  %s\n' "$$digest" "$$asset" > selected.sha256; sha256sum --check selected.sha256; tar -xzf "$$asset"); \
	"$$directory/quality-bash" version; \
	"$$directory/quality-bash" tools install; \
	"$$directory/quality-bash" qa --root testdata/basic --jobs 2

smoke-compare:
	test "$$ACTION_COVERAGE" = "$$CLI_COVERAGE"
	test "$$ACTION_MUTATION" = "$$CLI_MUTATION"

smoke-example:
	directory=$$(mktemp -d "$$RUNNER_TEMP/quality-bash-example.XXXXXXXX"); \
	trap 'rm -rf -- "$$directory"' EXIT; \
	export XDG_BIN_HOME="$$directory/bin" XDG_CACHE_HOME="$$directory/cache"; \
	$(MAKE) -C example quality-bash-install; \
	$(MAKE) -C example quality-bash-qa ARGS='--jobs 2'
