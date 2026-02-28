# Teamwork — Central Command Interface
# ======================================
# This Makefile is the single entry point for all project operations.
# AI agents and human developers use the same targets.
#
# Usage: make <target>
#   Run `make help` (or just `make`) to see available targets.

.DEFAULT_GOAL := help

.PHONY: help setup lint test build check plan review clean

help: ## Show this help message
	@echo "Teamwork — available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
	@echo ""

setup: ## One-time dev environment setup
	@bash scripts/setup.sh

lint: ## Run linters
	@bash scripts/lint.sh

test: ## Run tests
	@bash scripts/test.sh

build: ## Build the project
	@bash scripts/build.sh

check: lint test build ## Run lint + test + build in sequence

plan: ## Invoke planning agent (usage: make plan GOAL="description")
	@bash scripts/plan.sh "$(GOAL)"

review: ## Invoke review agent (usage: make review REF="pr-number-or-branch")
	@bash scripts/review.sh "$(REF)"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@# TODO: Add project-specific clean commands, e.g.:
	@#   rm -rf dist/ build/ node_modules/.cache coverage/
	@echo "Nothing to clean yet — add project-specific paths to the clean target."
