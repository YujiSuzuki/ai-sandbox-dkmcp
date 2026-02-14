// Package mcp provides host tool and host command MCP handlers.
// These handlers enable AI assistants to discover and execute tools on the host OS,
// and to run whitelisted CLI commands on the host.
//
// mcpパッケージはホストツールおよびホストコマンドのMCPハンドラーを提供します。
// これらのハンドラーにより、AIアシスタントがホストOS上のツールを検出・実行し、
// ホワイトリストに登録されたCLIコマンドをホスト上で実行できるようになります。
package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/hosttools"
)

// GetHostTools returns the MCP tool definitions for host tool operations.
// These are appended to the main tool list when host tools are enabled.
//
// GetHostToolsはホストツール操作のMCPツール定義を返します。
// ホストツールが有効な場合、メインのツールリストに追加されます。
func GetHostTools() []Tool {
	return []Tool{
		{
			Name:        "list_host_tools",
			Description: "List available tools in .sandbox/host-tools/ with descriptions and execution environment info",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]ToolProperty{},
			},
		},
		{
			Name:        "get_host_tool_info",
			Description: "Get detailed information about a specific host tool including usage, options, and execution environment",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"name": {
						Type:        "string",
						Description: "Tool filename (e.g. my-tool.sh)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "run_host_tool",
			Description: "Execute a host tool. Host tools are scripts/programs in configured directories on the host OS.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"name": {
						Type:        "string",
						Description: "Tool filename (e.g. my-tool.sh)",
					},
					"args": {
						Type:        "array",
						Description: "Arguments to pass to the tool",
						Items:       &ToolPropertyItems{Type: "string"},
					},
				},
				Required: []string{"name"},
			},
		},
	}
}

// toolListHostTools implements the list_host_tools MCP tool.
// toolListHostToolsはlist_host_tools MCPツールを実装します。
func (s *Server) toolListHostTools(ctx context.Context, args map[string]any) (any, error) {
	if s.hostToolsManager == nil {
		return nil, fmt.Errorf("host tools are not configured")
	}

	slog.Debug("Listing host tools")
	tools, err := s.hostToolsManager.ListTools()
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"tools": tools,
		"count": len(tools),
	}
	return jsonTextResponse(result)
}

// toolGetHostToolInfo implements the get_host_tool_info MCP tool.
// toolGetHostToolInfoはget_host_tool_info MCPツールを実装します。
func (s *Server) toolGetHostToolInfo(ctx context.Context, args map[string]any) (any, error) {
	if s.hostToolsManager == nil {
		return nil, fmt.Errorf("host tools are not configured")
	}

	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid name parameter")
	}

	slog.Debug("Getting host tool info", "name", name)
	info, err := s.hostToolsManager.GetToolInfo(name)
	if err != nil {
		return nil, err
	}

	return jsonTextResponse(info)
}

// toolRunHostTool implements the run_host_tool MCP tool.
// toolRunHostToolはrun_host_tool MCPツールを実装します。
func (s *Server) toolRunHostTool(ctx context.Context, args map[string]any) (any, error) {
	if s.hostToolsManager == nil {
		return nil, fmt.Errorf("host tools are not configured")
	}

	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid name parameter")
	}

	// Extract optional args parameter
	// オプションのargsパラメータを抽出
	var toolArgs []string
	if argsRaw, ok := args["args"].([]any); ok {
		for _, a := range argsRaw {
			if s, ok := a.(string); ok {
				toolArgs = append(toolArgs, s)
			}
		}
	}

	slog.Info("Running host tool", "name", name, "args", toolArgs)
	result, err := s.hostToolsManager.RunTool(name, toolArgs)
	if err != nil {
		return nil, err
	}

	// Apply output masking and host path masking
	// 出力マスキングとホストパスマスキングを適用
	output := result.String()
	output = s.docker.GetPolicy().MaskExec(output)
	output = s.docker.GetPolicy().MaskHostPaths(output)

	content := fmt.Sprintf("Tool: %s\nExit Code: %d\n\nOutput:\n%s", name, result.ExitCode, output)
	return textResponse(content), nil
}

// GetHostCommandTools returns the MCP tool definitions for host command operations.
// These are appended to the main tool list when host commands are enabled.
//
// GetHostCommandToolsはホストコマンド操作のMCPツール定義を返します。
// ホストコマンドが有効な場合、メインのツールリストに追加されます。
func GetHostCommandTools() []Tool {
	return []Tool{
		{
			Name:        "exec_host_command",
			Description: "Execute a whitelisted CLI command on the host OS. Commands must be configured in the host_commands whitelist. Use dangerously=true for commands in the dangerously list.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"command": {
						Type:        "string",
						Description: "Command to execute (must be whitelisted in config)",
					},
					"dangerously": {
						Type:        "boolean",
						Description: "Enable dangerous mode to execute commands from the dangerously list",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

// toolExecHostCommand implements the exec_host_command MCP tool.
// toolExecHostCommandはexec_host_command MCPツールを実装します。
func (s *Server) toolExecHostCommand(ctx context.Context, args map[string]any) (any, error) {
	if s.hostCommandPolicy == nil {
		return nil, fmt.Errorf("host commands are not configured")
	}

	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid command parameter")
	}

	dangerously := false
	if d, ok := args["dangerously"].(bool); ok {
		dangerously = d
	}

	// Check security policy
	// セキュリティポリシーをチェック
	var allowed bool
	var err error
	if dangerously {
		slog.Warn("Executing host command (DANGEROUS MODE)", "command", command)
		allowed, err = s.hostCommandPolicy.CanExecHostCommandDangerously(command)
	} else {
		slog.Info("Executing host command", "command", command)
		allowed, err = s.hostCommandPolicy.CanExecHostCommand(command)
	}

	if err != nil {
		slog.Warn("Host command blocked", "command", command, "dangerously", dangerously, "error", err.Error())
		return nil, err
	}
	if !allowed {
		slog.Warn("Host command not allowed", "command", command, "dangerously", dangerously)
		return nil, fmt.Errorf("command not allowed: %s", command)
	}

	// Defensive check: workspaceRoot must be set (should be caught by config.Validate)
	// 防御的チェック: workspaceRootが設定されている必要がある（config.Validateで検出されるはず）
	if s.workspaceRoot == "" {
		return nil, fmt.Errorf("workspace root is not configured")
	}

	// Execute the command
	// コマンドを実行
	result, err := hosttools.ExecHostCommand(command, s.workspaceRoot, s.hostCommandTimeout)
	if err != nil {
		return nil, err
	}

	// Apply output masking
	// 出力マスキングを適用
	output := result.String()
	output = s.docker.GetPolicy().MaskExec(output)
	output = s.docker.GetPolicy().MaskHostPaths(output)

	// Add warning for dangerous mode
	// 危険モードの場合は警告を追加
	prefix := ""
	if dangerously {
		prefix = "⚠️ [DANGEROUS MODE] "
	}

	content := fmt.Sprintf("%sCommand: %s\nExit Code: %d\n\nOutput:\n%s",
		prefix, command, result.ExitCode, output)
	return textResponse(content), nil
}
