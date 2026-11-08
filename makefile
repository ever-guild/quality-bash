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
	$(MAKE) -C example quality-bash QUALITY_BASH="$(CURDIR)/quality-bash"

release:
	$(GO) run ./cmd/package-release --version v1.0.0-alpha.5 --output "$$RELEASE_DIST"

smoke-metrics:
	test "$$COVERED/$$LINES" = '3/5'
	test "$$DETECTED/$$MUTANTS" = '2/3'
	test "$$PASSED" = 'true'

smoke-cli:
	case "$$RUNNER_ARCH_VALUE" in \
	  X64) arch=amd64; digest=66349fa3434ee12ee64afe3fb78a1634ca7332eefa87102c6071fe7536399450 ;; \
	  ARM64) arch=arm64; digest=35609a8b2ba9037908ef8d73dcd47ecffc318eacb73ef949ef821d98ac82b429 ;; \
	  *) exit 1 ;; \
	esac; \
	directory=$$(mktemp -d "$$RUNNER_TEMP/quality-bash-local.XXXXXXXX"); \
	trap 'rm -rf -- "$$directory"' EXIT; \
	export XDG_CACHE_HOME="$$directory/cache"; \
	release=https://github.com/ever-guild/quality-bash/releases/download/v1.0.0-alpha.5; \
	asset="quality-bash-v1.0.0-alpha.5-linux-$$arch.tar.gz"; \
	curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' "$$release/$$asset" -o "$$directory/$$asset"; \
	(cd "$$directory"; printf '%s  %s\n' "$$digest" "$$asset" > selected.sha256; sha256sum --check selected.sha256; tar -xzf "$$asset"); \
	"$$directory/quality-bash" version; \
	"$$directory/quality-bash" tools install; \
	"$$directory/quality-bash" qa --root testdata/basic

smoke-compare:
	test "$$ACTION_COVERAGE" = "$$CLI_COVERAGE"
	test "$$ACTION_MUTATION" = "$$CLI_MUTATION"

smoke-example:
	directory=$$(mktemp -d "$$RUNNER_TEMP/quality-bash-example.XXXXXXXX"); \
	trap 'rm -rf -- "$$directory"' EXIT; \
	export XDG_BIN_HOME="$$directory/bin" XDG_CACHE_HOME="$$directory/cache"; \
	$(MAKE) -C example quality-bash
