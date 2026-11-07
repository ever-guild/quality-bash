SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := check

GO ?= go
KCOV_PREFIX ?= $(if $(RUNNER_TEMP),$(RUNNER_TEMP)/kcov,$(CURDIR)/.build/kcov)
TOOL_DIST ?= tool-dist
RELEASE_DIST ?= dist
export KCOV_PREFIX TOOL_DIST RELEASE_DIST

.PHONY: check test integration vet build version ci-deps kcov bundle-tools verify-tools release smoke-metrics smoke-cli smoke-compare

check: test vet build

test:
	$(GO) test -race ./...

integration:
	PATH="$$KCOV_PREFIX/bin:$$PATH" $(GO) test -race -count=1 -v ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build -trimpath -o quality-bash ./cmd/quality-bash

version:
	./quality-bash version

ci-deps:
	sudo apt-get update -qq
	sudo apt-get install -y --no-install-recommends cmake g++ make python3 libcurl4-openssl-dev libdw-dev libelf-dev binutils-dev zlib1g-dev libssl-dev bats shellcheck

kcov:
	$(GO) run ./cmd/build-kcov --prefix "$$KCOV_PREFIX"

bundle-tools:
	$(GO) run ./cmd/bundle-tools --kcov "$$KCOV_PREFIX/bin/kcov" --output "$$TOOL_DIST"

verify-tools:
	$(GO) run ./cmd/check-tool-bundles --directory "$$TOOL_DIST"

release:
	$(GO) run ./cmd/package-release --version v1.0.0-alpha.2 --tools "$$TOOL_DIST" --output "$$RELEASE_DIST"

smoke-metrics:
	test "$$COVERED/$$LINES" = '3/5'
	test "$$DETECTED/$$MUTANTS" = '2/3'
	test "$$PASSED" = 'true'

smoke-cli:
	case "$$RUNNER_ARCH_VALUE" in \
	  X64) arch=amd64; digest=0f6745e4c4163711575a73dd184047f9393c19cc5193f39ec0c48d8aa9f39ee3 ;; \
	  ARM64) arch=arm64; digest=3e9b9fb9d63782d9fee13d186b89a2070a984900634c76ba9d024d410759eec7 ;; \
	  *) exit 1 ;; \
	esac; \
	directory=$$(mktemp -d "$$RUNNER_TEMP/quality-bash-local.XXXXXXXX"); \
	trap 'rm -rf -- "$$directory"' EXIT; \
	export XDG_CACHE_HOME="$$directory/cache"; \
	release=https://github.com/ever-guild/quality-bash/releases/download/v1.0.0-alpha.2; \
	asset="quality-bash-v1.0.0-alpha.2-linux-$$arch.tar.gz"; \
	curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' "$$release/$$asset" -o "$$directory/$$asset"; \
	(cd "$$directory"; printf '%s  %s\n' "$$digest" "$$asset" > selected.sha256; sha256sum --check selected.sha256; tar -xzf "$$asset"); \
	"$$directory/quality-bash" version; \
	"$$directory/quality-bash" tools install; \
	"$$directory/quality-bash" qa --root testdata/basic --jobs 2

smoke-compare:
	test "$$ACTION_COVERAGE" = "$$CLI_COVERAGE"
	test "$$ACTION_MUTATION" = "$$CLI_MUTATION"
