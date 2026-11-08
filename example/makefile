SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

export XDG_BIN_HOME ?= $(HOME)/.local/bin

QUALITY_BASH ?= $(XDG_BIN_HOME)/quality-bash
QUALITY_BASH_INSTALLER := https://raw.githubusercontent.com/ever-guild/quality-bash/main/install.sh
ARGS ?=

.PHONY: quality-bash-install
quality-bash-install:
	curl -fsSL --proto '=https' --proto-redir '=https' "$(QUALITY_BASH_INSTALLER)" | bash

$(QUALITY_BASH):
	@$(MAKE) --no-print-directory quality-bash-install

.PHONY: quality-bash
quality-bash: $(QUALITY_BASH)
	"$(QUALITY_BASH)" qa $(ARGS)

.PHONY: quality-bash-fix
quality-bash-fix: $(QUALITY_BASH)
	"$(QUALITY_BASH)" baseline-update $(ARGS)
