# VALID Makefile — run targets inside the Dev Container (or any Go 1.22+ environment).

GO ?= go
BIN_DIR ?= bin
BIN ?= $(BIN_DIR)/valid
MODULE_PKGS ?= ./...
CMD ?= ./cmd/valid

# Release metadata (override: make dist VERSION=v0.2.0)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X main.version=$(VERSION)

# Default Dev Container / WSL dogfood target
GOOS_DEFAULT ?= linux
GOARCH_DEFAULT ?= amd64

.PHONY: build build-linux build-linux-arm64 build-windows build-darwin dist test tidy fmt vet sync-ide run-dashboard run-mcp clean-bin

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

## build: compile for this machine (Dev Container default)
build:
	CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN) $(CMD)

## build-linux: linux/amd64 (same as typical VALID Dev Container / WSL)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/valid-linux-amd64 $(CMD)

## build-linux-arm64: linux/arm64
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/valid-linux-arm64 $(CMD)

## build-windows: windows/amd64 (.exe)
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/valid-windows-amd64.exe $(CMD)

## build-darwin: macOS amd64 + arm64
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/valid-darwin-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/valid-darwin-arm64 $(CMD)

## dist: release binaries for linux/windows/darwin (copy linux-amd64 → bin/valid too)
dist: build-linux build-linux-arm64 build-windows build-darwin
	cp $(BIN_DIR)/valid-linux-amd64 $(BIN)
	@echo "dist ready (VERSION=$(VERSION)):"
	@ls -la $(BIN_DIR)/valid*

## clean-bin: remove built binaries
clean-bin:
	rm -f $(BIN) $(BIN_DIR)/valid-*

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
	$(GO) run $(CMD) dashboard --feature $(FEATURE)

## run-mcp: stdio MCP server
run-mcp:
	$(GO) run $(CMD) mcp
