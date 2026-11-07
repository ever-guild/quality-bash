SHELL := /bin/bash

export XDG_BIN_HOME ?= $(HOME)/.local/bin

QUALITY_BASH ?= $(XDG_BIN_HOME)/quality-bash
QUALITY_BASH_INSTALLER := https://raw.githubusercontent.com/ever-guild/quality-bash/refs/heads/main/install.sh
ARGS ?=

.PHONY: quality-bash-install quality-bash-upgrade
quality-bash-upgrade: quality-bash-install:
	curl -fsSL $(QUALITY_BASH_INSTALLER) | bash

$(QUALITY_BASH):
	@$(MAKE) --no-print-directory quality-bash-upgrade

.PHONY: quality-bash-check quality-bash-lint quality-bash-fix
quality-bash-check quality-bash-lint quality-bash-fix: quality-bash-%: $(QUALITY_BASH)
	$(QUALITY_BASH) $* $(ARGS)
