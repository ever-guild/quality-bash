SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

export XDG_BIN_HOME ?= $(HOME)/.local/bin

QUALITY_BASH ?= $(XDG_BIN_HOME)/quality-bash
QUALITY_BASH_INSTALLER := https://raw.githubusercontent.com/ever-guild/quality-bash/68df4ac53a13a8ec44d2cd20ba9b830751aad615/install.sh
ARGS ?=

.PHONY: quality-bash-install quality-bash-upgrade
quality-bash-upgrade: quality-bash-install
quality-bash-install:
	curl -fsSL --proto '=https' --proto-redir '=https' "$(QUALITY_BASH_INSTALLER)" | bash

$(QUALITY_BASH):
	@$(MAKE) --no-print-directory quality-bash-upgrade

.PHONY: quality-bash-qa quality-bash-lint quality-bash-baseline-update
quality-bash-qa quality-bash-lint quality-bash-baseline-update: quality-bash-%: $(QUALITY_BASH)
	"$(QUALITY_BASH)" $* $(ARGS)
