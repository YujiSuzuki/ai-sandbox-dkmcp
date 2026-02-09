.PHONY: build build-all build-host clean test install install-host

# Versioning
# バージョン管理
# Get the version from the latest git tag
# 最新のgitタグからバージョンを取得
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags="-X 'github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/cli.Version=${VERSION}' -X 'github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/client.clientVersion=${VERSION}'"

# Build for current platform
# 現在のプラットフォーム向けにビルド
build:
	@echo "Building dkmcp version ${VERSION}..."
	CGO_ENABLED=0 go build ${LDFLAGS} -o dkmcp ./cmd/dkmcp

# Build for all platforms
# 全プラットフォーム向けにビルド（クロスコンパイル）
build-all:
	@echo "Building dkmcp version ${VERSION} for all platforms..."
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_windows_amd64.exe ./cmd/dkmcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/dkmcp_darwin_arm64 ./cmd/dkmcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_darwin_amd64 ./cmd/dkmcp
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_linux_amd64 ./cmd/dkmcp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/dkmcp_linux_arm64 ./cmd/dkmcp

# Cross-build for the host OS
# ホストOS向けクロスビルド
#
# Read host OS/arch from .sandbox/.host-os (written at container startup)
# Falls back to current platform if file not found
# .sandbox/.host-os からホストOS/アーキテクチャを読み取り（コンテナ起動時に書き出される）
# ファイルがない場合は現在のプラットフォームにフォールバック
HOST_OS_FILE := $(wildcard $(realpath ../.sandbox/.host-os))
ifdef HOST_OS_FILE
  HOST_OS ?= $(shell sed -n '1p' $(HOST_OS_FILE))
  HOST_ARCH ?= $(shell sed -n '2p' $(HOST_OS_FILE))
else
  HOST_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
  HOST_ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
endif
DEST ?= dist
build-host:
	@echo "Building dkmcp version ${VERSION} for ${HOST_OS}/${HOST_ARCH}..."
	mkdir -p dist
	CGO_ENABLED=0 GOOS=${HOST_OS} GOARCH=${HOST_ARCH} go build ${LDFLAGS} -o dist/dkmcp_${HOST_OS}_${HOST_ARCH} ./cmd/dkmcp
	@echo "✓ Built: dist/dkmcp_${HOST_OS}_${HOST_ARCH}"

# Install cross-built binary to host OS (run on host OS)
# クロスビルドしたバイナリをホストOSにインストール（ホストOS上で実行）
# Usage: make install-host [DEST=/usr/local/bin]
install-host:
	@set -e; \
	BINARY="dist/dkmcp_${HOST_OS}_${HOST_ARCH}"; \
	if [ ! -f "$$BINARY" ]; then \
		echo "Error: $$BINARY not found. Run 'make build-host' first."; \
		exit 1; \
	fi; \
	mkdir -p ${DEST}; \
	cp $$BINARY ${DEST}/dkmcp; \
	chmod +x ${DEST}/dkmcp; \
	echo "✓ Installed to ${DEST}/dkmcp"


# Clean build artifacts
# ビルド成果物をクリーンアップ
clean:
	rm -f dkmcp
	rm -rf dist/

# Run tests
# テストを実行
test:
	go test -v ./...

# Run tests with coverage
# カバレッジレポート付きでテストを実行
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests for a specific package
# 特定パッケージのテストを実行
test-security:
	go test -v ./internal/security

test-config:
	go test -v ./internal/config

test-docker:
	go test -v ./internal/docker

test-mcp:
	go test -v ./internal/mcp

# Install locally (DevContainer: /home/node/.local/bin, otherwise: $GOPATH/bin)
# ローカルにインストール（DevContainerの場合は/home/node/.local/binに永続化、それ以外は$GOPATH/binに配置）
install:
ifdef DEVCONTAINER
	@mkdir -p /home/node/.local/bin
	@CGO_ENABLED=0 go build ${LDFLAGS} -o /home/node/.local/bin/dkmcp ./cmd/dkmcp
	@echo "✓ Installed to /home/node/.local/bin/dkmcp"
	@echo "  (persisted in DevContainer volume)"
else
	CGO_ENABLED=0 go install ${LDFLAGS} ./cmd/dkmcp
	@echo "✓ Installed to $$GOPATH/bin/dkmcp"
endif



# Run the server (development)
# サーバーを起動（開発モード）
run:
	go run ./cmd/dkmcp serve

# Format code
# コードをフォーマット
fmt:
	go fmt ./...

# Lint code
# コードをリント
lint:
	golangci-lint run
