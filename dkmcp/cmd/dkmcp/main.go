// Package main is the entry point for the DockMCP application.
// DockMCPアプリケーションのエントリーポイントとなるパッケージです。
//
// DockMCP is an MCP (Model Context Protocol) server that provides
// controlled access to Docker containers for AI assistants.
// DockMCPは、AIアシスタントがDockerコンテナに制御されたアクセスを
// 提供するMCP（Model Context Protocol）サーバーです。
package main

import (
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/cli"
)

// BuildTime is set during build using ldflags.
// This variable is populated at compile time with the build timestamp.
// Example: go build -ldflags "-X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// BuildTimeはビルド時にldflagsを使用して設定されます。
// この変数はコンパイル時にビルドタイムスタンプで埋められます。
// 例: go build -ldflags "-X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var BuildTime string = "development"

// main initializes and executes the CLI application.
// The actual command parsing and execution is delegated to the cli package.
//
// mainはCLIアプリケーションを初期化して実行します。
// 実際のコマンド解析と実行はcliパッケージに委譲されます。
func main() {
	cli.Execute()
}
