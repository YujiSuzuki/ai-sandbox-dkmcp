// sandbox-mcp is a lightweight MCP server (stdio) for AI Sandbox scripts and tools.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/jsonrpc"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/sandbox-mcp/internal/server"
)

// version is set at build time via ldflags.
// ビルド時に ldflags で設定されます。
//
// go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("sandbox-mcp " + version)
		return
	}

	// Log to stderr only (stdout is reserved for JSON-RPC)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	srv := server.New(
		"/workspace/.sandbox/scripts",
		"/workspace/.sandbox/tools",
		version,
	)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpc.Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonrpc.NewErrorResponse(nil, jsonrpc.CodeParseError, "Parse error")
			writeResponse(resp)
			continue
		}

		resp := srv.HandleRequest(&req)
		if resp != nil {
			writeResponse(resp)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("stdin scanner error", "error", err)
		os.Exit(1)
	}
}

func writeResponse(resp *jsonrpc.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		return
	}
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}
