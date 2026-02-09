BINARY = sandbox-mcp
CMD_DIR = ./cmd/sandbox-mcp

.PHONY: build test install register unregister clean

build:
	@if [ ! -f $(BINARY) ]; then \
		if echo "$$LANG" | grep -qi "ja"; then \
			echo "⏳ 初回ビルド中 - 少々お待ちください..."; \
		else \
			echo "⏳ First time build - this may take a moment..."; \
		fi; \
		CGO_ENABLED=0 go build -o $(BINARY) $(CMD_DIR); \
	fi

test:
	go test ./...

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
