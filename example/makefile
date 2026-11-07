SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := quality-bash-check

export XDG_BIN_HOME ?= $(HOME)/.local/bin
QUALITY_BASH ?= $(XDG_BIN_HOME)/quality-bash
QUALITY_BASH_INSTALLER := https://raw.githubusercontent.com/ever-guild/quality-bash/v1.0.0-alpha.2/install.sh
ARGS ?=

.PHONY: quality-bash-install quality-bash-upgrade quality-bash-ready quality-bash-tools quality-bash-check quality-bash-lint quality-bash-baseline-create quality-bash-baseline-update

quality-bash-upgrade: quality-bash-install

quality-bash-install:
	temporary=$$(mktemp); trap 'rm -f -- "$$temporary"' EXIT; \
	curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' "$(QUALITY_BASH_INSTALLER)" -o "$$temporary"; \
	bash "$$temporary"

quality-bash-ready:
	@test -x "$(QUALITY_BASH)" || { echo 'Run make quality-bash-install first' >&2; exit 1; }

quality-bash-tools: quality-bash-ready
	"$(QUALITY_BASH)" tools install

quality-bash-check: quality-bash-ready
	"$(QUALITY_BASH)" qa $(ARGS)

quality-bash-lint: quality-bash-ready
	"$(QUALITY_BASH)" lint $(ARGS)

quality-bash-baseline-create: quality-bash-ready
	"$(QUALITY_BASH)" baseline update --create $(ARGS)

quality-bash-baseline-update: quality-bash-ready
	"$(QUALITY_BASH)" baseline update $(ARGS)
