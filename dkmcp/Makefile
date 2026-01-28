.PHONY: build build-all clean test install

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
	go build ${LDFLAGS} -o dkmcp ./cmd/dkmcp

# Build for all platforms
# 全プラットフォーム向けにビルド（クロスコンパイル）
build-all:
	@echo "Building dkmcp version ${VERSION} for all platforms..."
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_windows_amd64.exe ./cmd/dkmcp
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/dkmcp_darwin_arm64 ./cmd/dkmcp
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_darwin_amd64 ./cmd/dkmcp
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/dkmcp_linux_amd64 ./cmd/dkmcp
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/dkmcp_linux_arm64 ./cmd/dkmcp

# Build for MacOS ARM64
# MacOS ARM64向けにビルド
build-mac-arm64:
	@echo "Building dkmcp version ${VERSION} for MacOS ARM64..."
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/dkmcp_darwin_arm64 ./cmd/dkmcp

# Install binary built in DevContainer (run on MacOS host)
# DevContainerでビルドしたバイナリをインストール（MacOSホスト上で実行）
install-mac-arm64:
	@mv dist/dkmcp_darwin_arm64 $$GOPATH/bin/dkmcp



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
	@go build ${LDFLAGS} -o /home/node/.local/bin/dkmcp ./cmd/dkmcp
	@echo "✓ Installed to /home/node/.local/bin/dkmcp"
	@echo "  (persisted in DevContainer volume)"
else
	go install ${LDFLAGS} ./cmd/dkmcp
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
