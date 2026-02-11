BINARY = sandbox-mcp
CMD_DIR = ./cmd/sandbox-mcp

# Versioning: get the version from the latest git tag
# バージョン管理: 最新のgitタグからバージョンを取得
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X 'main.version=$(VERSION)'"

.PHONY: build test test-version install register unregister clean

build:
	@if [ ! -f $(BINARY) ]; then \
		if echo "$$LANG" | grep -qi "ja"; then \
			echo "⏳ 初回ビルド中 - 少々お待ちください..."; \
		else \
			echo "⏳ First time build - this may take a moment..."; \
		fi; \
		CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) $(CMD_DIR); \
	fi

test:
	go test ./...

# Build integration test: verify ldflags version injection
# ビルド統合テスト: ldflags によるバージョン注入の検証
test-version:
	@TEST_BIN=$$(mktemp) && \
	TEST_VER="0.0.0-test" && \
	CGO_ENABLED=0 go build -ldflags="-X 'main.version=$$TEST_VER'" -o "$$TEST_BIN" $(CMD_DIR) && \
	GOT=$$($$TEST_BIN version) && \
	rm -f "$$TEST_BIN" && \
	if [ "$$GOT" = "sandbox-mcp $$TEST_VER" ]; then \
		echo "PASS: $$GOT"; \
	else \
		echo "FAIL: got='$$GOT', want='sandbox-mcp $$TEST_VER'"; exit 1; \
	fi

install: build
	@if [ -d "$(HOME)/.local/bin" ]; then \
		cp $(BINARY) $(HOME)/.local/bin/$(BINARY); \
		echo "Installed to $(HOME)/.local/bin/$(BINARY)"; \
	elif [ -n "$(GOPATH)" ]; then \
		cp $(BINARY) $(GOPATH)/bin/$(BINARY); \
		echo "Installed to $(GOPATH)/bin/$(BINARY)"; \
	else \
		cp $(BINARY) $(HOME)/go/bin/$(BINARY); \
		echo "Installed to $(HOME)/go/bin/$(BINARY)"; \
	fi

WORKSPACE ?= /workspace

# Register to all available CLIs (Claude, Gemini)
register: build register-claude register-gemini

register-claude:
	@if ! command -v claude >/dev/null 2>&1; then \
		echo "[Claude] CLI not installed, skipping"; \
	elif [ -f "$(HOME)/.claude.json" ] && jq -e '.projects["$(WORKSPACE)"].mcpServers["sandbox-mcp"]' "$(HOME)/.claude.json" >/dev/null 2>&1; then \
		echo "[Claude] sandbox-mcp already registered"; \
	else \
		cd $(WORKSPACE) && claude mcp add sandbox-mcp $(CURDIR)/$(BINARY) && \
		echo "[Claude] sandbox-mcp registered"; \
	fi

register-gemini:
	@if ! command -v gemini >/dev/null 2>&1; then \
		echo "[Gemini] CLI not installed, skipping"; \
	elif [ -f "$(HOME)/.gemini/settings.json" ] && jq -e '.mcpServers["sandbox-mcp"]' "$(HOME)/.gemini/settings.json" >/dev/null 2>&1; then \
		echo "[Gemini] sandbox-mcp already registered"; \
	else \
		cd $(WORKSPACE) && gemini mcp add sandbox-mcp $(CURDIR)/$(BINARY) && \
		echo "[Gemini] sandbox-mcp registered" || \
		echo "[Gemini] skipped (not configured)"; \
	fi

# Unregister from all available CLIs
unregister: unregister-claude unregister-gemini

unregister-claude:
	@if ! command -v claude >/dev/null 2>&1; then \
		echo "[Claude] CLI not installed, skipping"; \
	else \
		cd $(WORKSPACE) && claude mcp remove sandbox-mcp 2>/dev/null && \
		echo "[Claude] sandbox-mcp removed" || \
		echo "[Claude] sandbox-mcp was not registered"; \
	fi

unregister-gemini:
	@if ! command -v gemini >/dev/null 2>&1; then \
		echo "[Gemini] CLI not installed, skipping"; \
	else \
		cd $(WORKSPACE) && gemini mcp remove sandbox-mcp 2>/dev/null && \
		echo "[Gemini] sandbox-mcp removed" || \
		echo "[Gemini] sandbox-mcp was not registered"; \
	fi

clean:
	rm -f $(BINARY)
