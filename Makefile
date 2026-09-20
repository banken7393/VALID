# VALID Makefile — run targets inside the Dev Container (or any Go 1.22+ environment).

GO ?= go
BIN ?= bin/valid
MODULE_PKGS ?= ./...

## sync-ide: copy packages/* wiring into internal/ide/assets (install embed source)
sync-ide:
	cp packages/cursor/mcp.json internal/ide/assets/cursor/mcp.json
	cp packages/cursor/rules/valid.mdc internal/ide/assets/cursor/rules/valid.mdc
	cp packages/claude-code/mcp.json internal/ide/assets/claude-code/mcp.json
	cp packages/claude-code/plugin.json internal/ide/assets/claude-code/plugin.json
	cp packages/claude-code/CLAUDE.md internal/ide/assets/claude-code/CLAUDE.md
	cp packages/opencode/opencode.valid.json internal/ide/assets/opencode/opencode.valid.json
	cp packages/codex/config.toml internal/ide/assets/codex/config.toml
	cp packages/codex/AGENTS.md internal/ide/assets/codex/AGENTS.md

.PHONY: build test tidy run-dashboard run-mcp fmt vet sync-ide

## build: compile the VALID CLI into bin/valid
build:
	$(GO) build -o $(BIN) ./cmd/valid

## test: run all unit tests
test:
	$(GO) test $(MODULE_PKGS) -count=1

## tidy: sync go.mod / go.sum
tidy:
	$(GO) mod tidy

## fmt: format packages
fmt:
	$(GO) fmt $(MODULE_PKGS)

## vet: static checks
vet:
	$(GO) vet $(MODULE_PKGS)

## run-dashboard: serve dashboard (requires FEATURE=name)
run-dashboard:
	@test -n "$(FEATURE)" || (echo "FEATURE=name required"; exit 1)
	$(GO) run ./cmd/valid dashboard --feature $(FEATURE)

## run-mcp: stdio MCP server
run-mcp:
	$(GO) run ./cmd/valid mcp
