// Package mcp provides the MCP tool definitions and implementations.
// This file defines all available tools that AI assistants can use to interact
// with Docker containers, including listing containers, getting logs, executing
// commands, and reading files.
//
// mcpパッケージはMCPツールの定義と実装を提供します。
// このファイルは、コンテナの一覧表示、ログの取得、コマンドの実行、
// ファイルの読み取りなど、AIアシスタントがDockerコンテナと対話するために
// 使用できるすべてのツールを定義しています。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// ServerVersion is the version string returned in MCP initialize response.
// This can be set by the CLI package to match the application version.
// Default value "dev" indicates a development build.
//
// ServerVersionはMCP initializeレスポンスで返されるバージョン文字列です。
// CLIパッケージからアプリケーションバージョンに合わせて設定できます。
// デフォルト値"dev"は開発ビルドを示します。
var ServerVersion = "dev"

// ToolInputSchema represents the JSON schema for tool input parameters.
// It follows the JSON Schema specification to define the structure of
// arguments that a tool accepts.
//
// ToolInputSchemaはツール入力パラメータのJSONスキーマを表します。
// ツールが受け入れる引数の構造を定義するために
// JSON Schema仕様に従っています。
type ToolInputSchema struct {
	// Type is the JSON type of the input (usually "object")
	// Typeは入力のJSON型（通常は"object"）
	Type string `json:"type"`

	// Properties defines the individual parameters and their schemas
	// Propertiesは個々のパラメータとそのスキーマを定義します
	Properties map[string]ToolProperty `json:"properties"`

	// Required lists the parameter names that must be provided
	// Requiredは必ず提供しなければならないパラメータ名のリスト
	Required []string `json:"required,omitempty"`
}

// ToolProperty represents a property in the tool input schema.
// It describes a single parameter including its type, description, and constraints.
//
// ToolPropertyはツール入力スキーマ内のプロパティを表します。
// 型、説明、制約を含む単一のパラメータを記述します。
type ToolProperty struct {
	// Type is the JSON type of this parameter (string, integer, boolean, etc.)
	// TypeはこのパラメータのJSON型（string、integer、booleanなど）
	Type string `json:"type"`

	// Description explains what this parameter is used for
	// Descriptionはこのパラメータの用途を説明します
	Description string `json:"description"`

	// Default is the default value if not provided
	// Defaultは提供されなかった場合のデフォルト値
	Default any `json:"default,omitempty"`

	// Minimum sets the minimum value for numeric parameters
	// Minimumは数値パラメータの最小値を設定します
	Minimum *int `json:"minimum,omitempty"`
}

// Tool represents an MCP tool that can be invoked by AI assistants.
// Each tool has a name, description, and input schema that defines
// what parameters it accepts.
//
// ToolはAIアシスタントが呼び出せるMCPツールを表します。
// 各ツールは名前、説明、および受け入れるパラメータを定義する
// 入力スキーマを持っています。
type Tool struct {
	// Name is the unique identifier for this tool
	// Nameはこのツールの一意の識別子
	Name string `json:"name"`

	// Description explains what this tool does
	// Descriptionはこのツールが何をするかを説明します
	Description string `json:"description"`

	// InputSchema defines the parameters this tool accepts
	// InputSchemaはこのツールが受け入れるパラメータを定義します
	InputSchema ToolInputSchema `json:"inputSchema"`
}

// ClientInfo represents the client information sent during MCP initialization.
// This identifies the AI assistant or application connecting to the server.
//
// ClientInfoはMCP初期化中に送信されるクライアント情報を表します。
// これはサーバーに接続するAIアシスタントまたはアプリケーションを識別します。
type ClientInfo struct {
	// Name is the name of the client application
	// Nameはクライアントアプリケーションの名前
	Name string `json:"name"`

	// Version is the version of the client application
	// Versionはクライアントアプリケーションのバージョン
	Version string `json:"version"`
}

// InitializeParams represents the parameters for the MCP initialize method.
// The initialize method must be called before any other methods can be used.
//
// InitializeParamsはMCPのinitializeメソッドのパラメータを表します。
// initializeメソッドは他のメソッドを使用する前に呼び出す必要があります。
type InitializeParams struct {
	// ClientInfo contains information about the connecting client
	// ClientInfoは接続しているクライアントに関する情報を含みます
	ClientInfo ClientInfo `json:"clientInfo"`
}

// initialize handles the MCP initialization request.
// It returns the server capabilities and extracts the client name and version for logging.
// This must be the first method called by a client before any other operations.
//
// initializeはMCP初期化リクエストを処理します。
// サーバーの機能を返し、ログ記録用にクライアント名とバージョンを抽出します。
// これは他の操作の前にクライアントが最初に呼び出すメソッドである必要があります。
func (s *Server) initialize(params any) (any, string, string, error) {
	slog.Debug("Handling initialize request")
	var clientName, clientVersion string

	// To robustly handle the params, marshal it back to JSON and then unmarshal into a struct
	// This avoids complex and unsafe type assertions
	// paramsを堅牢に処理するため、JSONに再マーシャルしてから構造体にアンマーシャル
	// これにより複雑で安全でない型アサーションを回避
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		slog.Warn("Could not marshal initialize params", "error", err)
	} else {
		var initParams InitializeParams
		if err := json.Unmarshal(paramsBytes, &initParams); err != nil {
			// This is not a critical error, client might not send params
			// これは重大なエラーではなく、クライアントがparamsを送信しない場合がある
			slog.Debug("Could not unmarshal initialize params into expected structure", "error", err)
		} else {
			clientName = initParams.ClientInfo.Name
			clientVersion = initParams.ClientInfo.Version
			// Logging is done in processRequest with clientID
			// ログ出力はclientIDを含むprocessRequestで行う
		}
	}

	// Build the response with server information and capabilities
	// サーバー情報と機能を含むレスポンスを構築
	response := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "dkmcp",
			"version": ServerVersion,
		},
		"capabilities": map[string]any{
			"tools": map[string]bool{},
		},
	}
	return response, clientName, clientVersion, nil
}

// GetTools returns the list of all available MCP tools.
// This function defines the complete tool catalog that AI assistants
// can use to interact with Docker containers.
//
// GetToolsは利用可能なすべてのMCPツールのリストを返します。
// この関数は、AIアシスタントがDockerコンテナと対話するために
// 使用できる完全なツールカタログを定義します。
func GetTools() []Tool {
	return []Tool{
		// list_containers: Lists all Docker containers accessible through DockMCP
		// list_containers: DockMCPを通じてアクセス可能なすべてのDockerコンテナを一覧表示
		{
			Name:        "list_containers",
			Description: "List all accessible Docker containers with their status and basic information",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"all": {
						Type:        "boolean",
						Description: "Show all containers (default: true)",
						Default:     true,
					},
				},
			},
		},
		// get_logs: Retrieves logs from a specific container
		// get_logs: 特定のコンテナからログを取得
		{
			Name:        "get_logs",
			Description: "Get logs from a specific container. Useful for debugging and monitoring container output.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
					"tail": {
						Type:        "string",
						Description: "Number of lines to show from the end (default: all)",
						Default:     "all",
					},
					"since": {
						Type:        "string",
						Description: "Show logs since timestamp (e.g., 2013-01-02T13:23:37Z) or relative (e.g., 42m for 42 minutes)",
					},
				},
				Required: []string{"container"},
			},
		},
		// get_stats: Gets resource usage statistics for a container
		// get_stats: コンテナのリソース使用統計を取得
		{
			Name:        "get_stats",
			Description: "Get resource usage statistics (CPU, memory, network, etc.) for a container",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
				},
				Required: []string{"container"},
			},
		},
		// exec_command: Executes a whitelisted command inside a container
		// exec_command: コンテナ内でホワイトリストに登録されたコマンドを実行
		{
			Name:        "exec_command",
			Description: "Execute a command inside a container. Only whitelisted commands are allowed based on security policy. Use dangerously=true to execute commands from exec_dangerously list (file paths are still checked against blocked_paths).",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
					"command": {
						Type:        "string",
						Description: "Command to execute (must be whitelisted in config, or in exec_dangerously list if dangerously=true)",
					},
					"dangerously": {
						Type:        "boolean",
						Description: "Enable dangerous mode to execute commands from exec_dangerously list. File paths in the command are still checked against blocked_paths. Pipes and redirects are not allowed.",
					},
				},
				Required: []string{"container", "command"},
			},
		},
		// inspect_container: Gets detailed information about a container
		// inspect_container: コンテナに関する詳細情報を取得
		{
			Name:        "inspect_container",
			Description: "Get detailed information about a container including configuration, network settings, and mounts",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
				},
				Required: []string{"container"},
			},
		},
		// get_allowed_commands: Lists whitelisted commands for a container
		// get_allowed_commands: コンテナのホワイトリストに登録されたコマンドを一覧表示
		{
			Name:        "get_allowed_commands",
			Description: "Get the list of whitelisted commands that can be executed in a container. Use this to discover what commands are available before using exec_command.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name. If not specified, returns commands for all containers.",
					},
				},
			},
		},
		// get_security_policy: Returns the current security policy configuration
		// get_security_policy: 現在のセキュリティポリシー設定を返す
		{
			Name:        "get_security_policy",
			Description: "Get the current security policy configuration including mode, allowed containers, permissions, and command whitelists.",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]ToolProperty{},
			},
		},
		// search_logs: Searches container logs for a pattern
		// search_logs: コンテナログ内でパターンを検索
		{
			Name:        "search_logs",
			Description: "Search container logs for a specific pattern. Returns matching lines with context.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
					"pattern": {
						Type:        "string",
						Description: "Search pattern (case-insensitive substring match)",
					},
					"tail": {
						Type:        "string",
						Description: "Number of log lines to search (default: 1000)",
						Default:     "1000",
					},
					"context_lines": {
						Type:        "integer",
						Description: "Number of context lines before and after each match (default: 2)",
						Default:     2,
					},
				},
				Required: []string{"container", "pattern"},
			},
		},
		// list_files: Lists files in a container directory
		// list_files: コンテナディレクトリ内のファイルを一覧表示
		{
			Name:        "list_files",
			Description: "List files in a container directory. Blocked paths will be denied with detailed reason.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
					"path": {
						Type:        "string",
						Description: "Directory path to list (default: /)",
						Default:     "/",
					},
				},
				Required: []string{"container"},
			},
		},
		// read_file: Reads a file from a container
		// read_file: コンテナからファイルを読み取る
		{
			Name:        "read_file",
			Description: "Read a file from a container. Blocked paths will be denied with detailed reason.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name or ID",
					},
					"path": {
						Type:        "string",
						Description: "File path to read",
					},
					"max_lines": {
						Type:        "integer",
						Description: "Maximum number of lines to read (default: 0 = all)",
						Default:     0,
					},
				},
				Required: []string{"container", "path"},
			},
		},
		// get_blocked_paths: Returns the list of blocked file paths
		// get_blocked_paths: ブロックされたファイルパスのリストを返す
		{
			Name:        "get_blocked_paths",
			Description: "Get the list of blocked file paths for a container or all containers.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]ToolProperty{
					"container": {
						Type:        "string",
						Description: "Container name. If not specified, returns blocked paths for all containers.",
					},
				},
			},
		},
	}
}

// listTools returns the list of available tools wrapped in the MCP response format.
// This is called when an AI assistant sends a "tools/list" request.
//
// listToolsは利用可能なツールのリストをMCPレスポンス形式でラップして返します。
// これはAIアシスタントが"tools/list"リクエストを送信したときに呼び出されます。
func (s *Server) listTools() (any, error) {
	return map[string]any{
		"tools": GetTools(),
	}, nil
}

// callTool executes a tool based on the request parameters.
// It routes the call to the appropriate tool handler based on the tool name.
//
// callToolはリクエストパラメータに基づいてツールを実行します。
// ツール名に基づいて呼び出しを適切なツールハンドラにルーティングします。
func (s *Server) callTool(ctx context.Context, params any) (any, error) {
	// Parse params to extract tool name and arguments
	// paramsを解析してツール名と引数を抽出
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid params format")
	}

	toolName, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	// Extract arguments, defaulting to empty map if not provided
	// 引数を抽出し、提供されない場合は空のマップをデフォルトとする
	arguments, ok := paramsMap["arguments"].(map[string]any)
	if !ok {
		arguments = make(map[string]any)
	}

	// Route to the appropriate tool handler based on tool name
	// ツール名に基づいて適切なツールハンドラにルーティング
	switch toolName {
	case "list_containers":
		return s.toolListContainers(ctx, arguments)
	case "get_logs":
		return s.toolGetLogs(ctx, arguments)
	case "get_stats":
		return s.toolGetStats(ctx, arguments)
	case "exec_command":
		return s.toolExecCommand(ctx, arguments)
	case "inspect_container":
		return s.toolInspectContainer(ctx, arguments)
	case "get_allowed_commands":
		return s.toolGetAllowedCommands(ctx, arguments)
	case "get_security_policy":
		return s.toolGetSecurityPolicy(ctx, arguments)
	case "search_logs":
		return s.toolSearchLogs(ctx, arguments)
	case "list_files":
		return s.toolListFiles(ctx, arguments)
	case "read_file":
		return s.toolReadFile(ctx, arguments)
	case "get_blocked_paths":
		return s.toolGetBlockedPaths(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// toolListContainers implements the list_containers tool.
// It retrieves a list of all Docker containers accessible through DockMCP
// and returns them as JSON formatted text.
//
// toolListContainersはlist_containersツールを実装します。
// DockMCPを通じてアクセス可能なすべてのDockerコンテナのリストを取得し、
// JSONフォーマットのテキストとして返します。
func (s *Server) toolListContainers(ctx context.Context, args map[string]any) (any, error) {
	slog.Debug("Listing containers")
	containers, err := s.docker.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	// Convert container list to JSON for structured, readable output
	// 構造化された読みやすい出力のためにコンテナリストをJSONに変換
	jsonBytes, err := json.MarshalIndent(containers, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container list: %w", err)
	}

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedJSON := s.docker.GetPolicy().MaskHostPaths(string(jsonBytes))

	return textResponse(maskedJSON), nil
}

// toolGetLogs implements the get_logs tool.
// It retrieves logs from a specific container, optionally limiting
// the number of lines returned.
//
// toolGetLogsはget_logsツールを実装します。
// 特定のコンテナからログを取得し、オプションで返す行数を制限します。
func (s *Server) toolGetLogs(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	// Extract optional tail parameter with default value
	// デフォルト値を持つオプションのtailパラメータを抽出
	tail := "all"
	if t, ok := args["tail"].(string); ok {
		tail = t
	}

	// Extract optional since parameter for timestamp filtering
	// タイムスタンプフィルタ用のオプションsinceパラメータを抽出
	since := ""
	if s, ok := args["since"].(string); ok {
		since = s
	}

	slog.Debug("Getting logs", "container", container, "since", since)
	logs, err := s.docker.GetLogs(ctx, container, tail, since, false)
	if err != nil {
		return nil, err
	}

	// Apply output masking to hide sensitive data
	// 機密データを隠すために出力マスキングを適用
	maskedLogs := s.docker.GetPolicy().MaskLogs(logs)

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedLogs = s.docker.GetPolicy().MaskHostPaths(maskedLogs)

	return textResponse(fmt.Sprintf("Logs from container '%s':\n\n%s", container, maskedLogs)), nil
}

// toolGetStats implements the get_stats tool.
// It retrieves resource usage statistics (CPU, memory, network, etc.)
// for a specific container.
//
// toolGetStatsはget_statsツールを実装します。
// 特定のコンテナのリソース使用統計（CPU、メモリ、ネットワークなど）を取得します。
func (s *Server) toolGetStats(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	slog.Debug("Getting stats", "container", container)
	stats, err := s.docker.GetStats(ctx, container)
	if err != nil {
		return nil, err
	}

	// Format stats as JSON text
	// 統計情報をJSONテキストでフォーマット
	content := formatStats(stats)

	return textResponse(content), nil
}

// toolExecCommand implements the exec_command tool.
// It executes a command inside a container, subject to security policy restrictions.
// Only whitelisted commands are allowed to prevent arbitrary code execution.
// If dangerously=true, commands from exec_dangerously list are allowed with path validation.
//
// toolExecCommandはexec_commandツールを実装します。
// セキュリティポリシーの制限に従って、コンテナ内でコマンドを実行します。
// 任意のコード実行を防ぐため、ホワイトリストに登録されたコマンドのみが許可されます。
// dangerously=trueの場合、パス検証付きでexec_dangerouslyリストのコマンドが許可されます。
func (s *Server) toolExecCommand(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	// Extract required command parameter
	// 必須のcommandパラメータを抽出
	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid command parameter")
	}

	// Extract optional dangerously parameter (defaults to false)
	// オプションのdangerouslyパラメータを抽出（デフォルトはfalse）
	dangerously := false
	if d, ok := args["dangerously"].(bool); ok {
		dangerously = d
	}

	// Log the command execution for audit purposes
	// Use WARN level for dangerous mode to make it stand out
	// 監査目的でコマンド実行をログに記録
	// 危険モードは目立つようにWARNレベルを使用
	if dangerously {
		slog.Warn("Executing command (DANGEROUS MODE)",
			"container", container,
			"command", command,
		)
	} else {
		slog.Info("Executing command",
			"container", container,
			"command", command,
		)
	}

	// Execute the command through the Docker client
	// The Docker client checks the whitelist (or exec_dangerously list if dangerously=true)
	// Dockerクライアントを通じてコマンドを実行
	// Dockerクライアントはホワイトリスト（dangerously=trueの場合はexec_dangerouslyリスト）をチェック
	result, err := s.docker.Exec(ctx, container, command, dangerously)
	if err != nil {
		// Log the failure for audit and debugging purposes
		// 監査とデバッグ目的で失敗をログに記録
		slog.Warn("Command execution blocked",
			"container", container,
			"command", command,
			"dangerously", dangerously,
			"error", err.Error(),
		)
		return nil, err
	}

	// Apply output masking to hide sensitive data in command output
	// コマンド出力内の機密データを隠すために出力マスキングを適用
	maskedOutput := s.docker.GetPolicy().MaskExec(result.Output)

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedOutput = s.docker.GetPolicy().MaskHostPaths(maskedOutput)

	// Format the result with command, exit code, and output
	// コマンド、終了コード、出力を含めて結果をフォーマット
	content := fmt.Sprintf("Command: %s\nExit Code: %d\n\nOutput:\n%s",
		command, result.ExitCode, maskedOutput)

	return textResponse(content), nil
}

// toolInspectContainer implements the inspect_container tool.
// It retrieves detailed information about a container including
// configuration, network settings, and mount points.
//
// toolInspectContainerはinspect_containerツールを実装します。
// 設定、ネットワーク設定、マウントポイントを含むコンテナの詳細情報を取得します。
func (s *Server) toolInspectContainer(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	slog.Debug("Inspecting container", "container", container)
	info, err := s.docker.InspectContainer(ctx, container)
	if err != nil {
		return nil, err
	}

	// Convert the inspection result to JSON, then apply masking
	// 検査結果をJSONに変換してからマスキングを適用
	jsonBytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inspect result: %w", err)
	}

	// Apply output masking to hide sensitive data (e.g., env vars)
	// 機密データ（例：環境変数）を隠すために出力マスキングを適用
	maskedJSON := s.docker.GetPolicy().MaskInspect(string(jsonBytes))

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedJSON = s.docker.GetPolicy().MaskHostPaths(maskedJSON)

	// Return masked JSON as text response
	// マスクされたJSONをテキストレスポンスとして返す
	return textResponse(maskedJSON), nil
}

// formatStats formats container stats as JSON text.
// It converts the stats to indented JSON.
//
// formatStatsはコンテナの統計情報をJSONテキストにフォーマットします。
// 統計情報をインデント付きJSONに変換します。
func formatStats(stats any) string {
	// Convert stats to pretty-printed JSON
	// 統計情報を整形されたJSONに変換
	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting stats: %v", err)
	}

	return string(jsonData)
}

// toolGetAllowedCommands implements the get_allowed_commands tool.
// It returns the list of commands that are whitelisted for execution
// in a specific container or all containers.
//
// toolGetAllowedCommandsはget_allowed_commandsツールを実装します。
// 特定のコンテナまたはすべてのコンテナで実行がホワイトリストに登録されている
// コマンドのリストを返します。
func (s *Server) toolGetAllowedCommands(ctx context.Context, args map[string]any) (any, error) {
	slog.Debug("Getting allowed commands")

	// Check if a specific container was requested
	// 特定のコンテナがリクエストされたかどうかを確認
	container, hasContainer := args["container"].(string)

	// Check if dangerous mode is enabled
	// 危険モードが有効かどうかを確認
	dangerousEnabled := s.docker.IsDangerousModeEnabled()

	var result any
	if hasContainer && container != "" {
		// Get commands for the specific container
		// 特定のコンテナのコマンドを取得
		commands := s.docker.GetAllowedCommands(container)
		resultMap := map[string]any{
			"container":        container,
			"allowed_commands": commands,
			"note":             "Commands with '*' wildcard match any suffix (e.g., 'echo *' matches 'echo hello')",
		}

		// Add dangerous commands if enabled
		// 危険モードが有効な場合、危険コマンドを追加
		if dangerousEnabled {
			dangerousCommands := s.docker.GetDangerousCommandsForContainer(container)
			resultMap["dangerous_commands"] = dangerousCommands
			resultMap["dangerous_mode_enabled"] = true
			resultMap["note"] = "Commands with '*' wildcard match any suffix. Dangerous commands require dangerously=true parameter."
		}

		result = resultMap
	} else {
		// Get commands for all containers
		// すべてのコンテナのコマンドを取得
		allCommands := s.docker.GetAllContainersWithCommands()
		resultMap := map[string]any{
			"containers": allCommands,
			"note":       "The '*' key contains default commands available to all containers. Commands with '*' wildcard match any suffix.",
		}

		// Add dangerous commands if enabled
		// 危険モードが有効な場合、危険コマンドを追加
		if dangerousEnabled {
			dangerousCommands := s.docker.GetAllDangerousCommands()
			resultMap["dangerous_containers"] = dangerousCommands
			resultMap["dangerous_mode_enabled"] = true
			resultMap["note"] = "The '*' key contains default commands available to all containers. Commands with '*' wildcard match any suffix. Dangerous commands require dangerously=true parameter."
		}

		result = resultMap
	}

	return jsonTextResponse(result)
}

// toolGetSecurityPolicy implements the get_security_policy tool.
// It returns the current security policy configuration including
// allowed containers, permissions, and command whitelists.
//
// toolGetSecurityPolicyはget_security_policyツールを実装します。
// 許可されたコンテナ、権限、コマンドホワイトリストを含む
// 現在のセキュリティポリシー設定を返します。
func (s *Server) toolGetSecurityPolicy(ctx context.Context, args map[string]any) (any, error) {
	slog.Debug("Getting security policy")

	policy := s.docker.GetSecurityPolicy()

	// Convert to JSON for masking
	// マスキングのためにJSONに変換
	jsonBytes, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal security policy: %w", err)
	}

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedJSON := s.docker.GetPolicy().MaskHostPaths(string(jsonBytes))

	return textResponse(fmt.Sprintf("Current Security Policy:\n```json\n%s\n```", maskedJSON)), nil
}

// toolSearchLogs implements the search_logs tool.
// It searches container logs for a pattern and returns matching lines
// with surrounding context.
//
// toolSearchLogsはsearch_logsツールを実装します。
// コンテナログ内でパターンを検索し、周囲のコンテキストと共に
// マッチした行を返します。
func (s *Server) toolSearchLogs(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	// Extract required pattern parameter
	// 必須のpatternパラメータを抽出
	pattern, ok := args["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid pattern parameter")
	}

	// Extract optional tail parameter with default
	// デフォルト値を持つオプションのtailパラメータを抽出
	tail := "1000"
	if t, ok := args["tail"].(string); ok {
		tail = t
	}

	// Extract optional context_lines parameter with default
	// デフォルト値を持つオプションのcontext_linesパラメータを抽出
	contextLines := 2
	if c, ok := args["context_lines"].(float64); ok {
		contextLines = int(c)
	}

	slog.Debug("Searching logs", "container", container, "pattern", pattern)

	// Get logs from the container
	// コンテナからログを取得
	logs, err := s.docker.GetLogs(ctx, container, tail, "", false)
	if err != nil {
		return nil, err
	}

	// Apply output masking before searching to hide sensitive data
	// 検索前に機密データを隠すために出力マスキングを適用
	maskedLogs := s.docker.GetPolicy().MaskLogs(logs)

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedLogs = s.docker.GetPolicy().MaskHostPaths(maskedLogs)

	// Search for pattern in logs (case-insensitive)
	// ログ内でパターンを検索（大文字小文字を区別しない）
	lines := strings.Split(maskedLogs, "\n")
	var matches []SearchMatch
	patternLower := strings.ToLower(pattern)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), patternLower) {
			match := SearchMatch{
				LineNumber: i + 1,
				Line:       line,
				Context:    getContextLines(lines, i, contextLines),
			}
			matches = append(matches, match)
		}
	}

	// Build result with search statistics and matches
	// 検索統計とマッチを含む結果を構築
	result := map[string]any{
		"container":     container,
		"pattern":       pattern,
		"total_lines":   len(lines),
		"matches_count": len(matches),
		"matches":       matches,
	}

	return jsonTextResponse(result)
}

// SearchMatch represents a single match found in log search.
// It includes the line number, matched line, and surrounding context.
//
// SearchMatchはログ検索で見つかった単一のマッチを表します。
// 行番号、マッチした行、および周囲のコンテキストを含みます。
type SearchMatch struct {
	// LineNumber is the 1-indexed line number of the match
	// LineNumberはマッチの1から始まる行番号
	LineNumber int `json:"line_number"`

	// Line is the full text of the matched line
	// Lineはマッチした行の全文
	Line string `json:"line"`

	// Context contains the surrounding lines for context
	// Contextはコンテキストのための周囲の行を含む
	Context []string `json:"context,omitempty"`
}

// getContextLines returns the lines surrounding a match at the given index.
// It includes lines before (marked with "-") and after (marked with "+")
// the matched line.
//
// getContextLinesは指定されたインデックスのマッチの周囲の行を返します。
// マッチした行の前（"-"でマーク）と後（"+"でマーク）の行を含みます。
func getContextLines(lines []string, index int, contextSize int) []string {
	// Return nil if no context is requested
	// コンテキストがリクエストされない場合はnilを返す
	if contextSize <= 0 {
		return nil
	}

	// Calculate the range of lines to include
	// 含める行の範囲を計算
	start := index - contextSize
	if start < 0 {
		start = 0
	}

	end := index + contextSize + 1
	if end > len(lines) {
		end = len(lines)
	}

	// Build context lines with line numbers and position indicators
	// 行番号と位置インジケータを含むコンテキスト行を構築
	var context []string
	for i := start; i < end; i++ {
		if i != index {
			// Mark lines before match with "-" and after with "+"
			// マッチ前の行は"-"、後の行は"+"でマーク
			prefix := "  "
			if i < index {
				prefix = "- "
			} else {
				prefix = "+ "
			}
			context = append(context, fmt.Sprintf("%s%d: %s", prefix, i+1, lines[i]))
		}
	}

	return context
}

// toolListFiles implements the list_files tool.
// It lists files in a container directory, respecting security policy
// restrictions on blocked paths.
//
// toolListFilesはlist_filesツールを実装します。
// ブロックされたパスに対するセキュリティポリシーの制限を尊重しながら、
// コンテナディレクトリ内のファイルを一覧表示します。
func (s *Server) toolListFiles(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	// Extract optional path parameter with default
	// デフォルト値を持つオプションのpathパラメータを抽出
	path := "/"
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	slog.Debug("Listing files", "container", container, "path", path)

	result, err := s.docker.ListFiles(ctx, container, path)
	if err != nil {
		return nil, err
	}

	// If blocked by security policy, return detailed block information
	// セキュリティポリシーによりブロックされた場合、詳細なブロック情報を返す
	if result.Blocked {
		return s.formatBlockedResponse(container, path, result.Block)
	}

	// If the operation failed for other reasons, return the error
	// 他の理由で操作が失敗した場合、エラーを返す
	if !result.Success {
		return containerFileResponse("Error listing files in", container, path, result.Error), nil
	}

	return containerFileResponse("Files in", container, path, result.Data), nil
}

// toolReadFile implements the read_file tool.
// It reads a file from a container, respecting security policy
// restrictions on blocked paths.
//
// toolReadFileはread_fileツールを実装します。
// ブロックされたパスに対するセキュリティポリシーの制限を尊重しながら、
// コンテナからファイルを読み取ります。
func (s *Server) toolReadFile(ctx context.Context, args map[string]any) (any, error) {
	// Extract required container parameter
	// 必須のcontainerパラメータを抽出
	container, ok := args["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid container parameter")
	}

	// Extract required path parameter
	// 必須のpathパラメータを抽出
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid path parameter")
	}

	// Extract optional max_lines parameter
	// オプションのmax_linesパラメータを抽出
	maxLines := 0
	if m, ok := args["max_lines"].(float64); ok {
		maxLines = int(m)
	}

	slog.Debug("Reading file", "container", container, "path", path)

	result, err := s.docker.ReadFile(ctx, container, path, maxLines)
	if err != nil {
		return nil, err
	}

	// If blocked by security policy, return detailed block information
	// セキュリティポリシーによりブロックされた場合、詳細なブロック情報を返す
	if result.Blocked {
		return s.formatBlockedResponse(container, path, result.Block)
	}

	// If the operation failed for other reasons, return the error
	// 他の理由で操作が失敗した場合、エラーを返す
	if !result.Success {
		return containerFileResponse("Error reading file", container, path, result.Error), nil
	}

	// Apply host path masking to hide host OS username and directory structure in file contents
	// ファイル内容内のホストOSのユーザー名やディレクトリ構造を隠すためにホストパスマスキングを適用
	maskedData := s.docker.GetPolicy().MaskHostPaths(result.Data)

	return containerFileResponse("Contents of", container, path, maskedData), nil
}

// toolGetBlockedPaths implements the get_blocked_paths tool.
// It returns the list of file paths that are blocked by security policy
// for a specific container or all containers.
//
// toolGetBlockedPathsはget_blocked_pathsツールを実装します。
// 特定のコンテナまたはすべてのコンテナに対してセキュリティポリシーにより
// ブロックされているファイルパスのリストを返します。
func (s *Server) toolGetBlockedPaths(ctx context.Context, args map[string]any) (any, error) {
	slog.Debug("Getting blocked paths")

	// Check if a specific container was requested
	// 特定のコンテナがリクエストされたかどうかを確認
	container, hasContainer := args["container"].(string)

	var result any
	if hasContainer && container != "" {
		// Get blocked paths for the specific container
		// 特定のコンテナのブロックされたパスを取得
		paths := s.docker.GetBlockedPathsForContainer(container)
		result = map[string]any{
			"container":     container,
			"blocked_paths": paths,
		}
	} else {
		// Get blocked paths for all containers
		// すべてのコンテナのブロックされたパスを取得
		paths := s.docker.GetBlockedPaths()
		result = map[string]any{
			"all_blocked_paths": paths,
		}
	}

	// Convert to JSON and apply host path masking
	// JSONに変換してホストパスマスキングを適用
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal blocked paths: %w", err)
	}

	// Apply host path masking to hide host OS username and directory structure
	// ホストパスマスキングを適用してホストOSのユーザー名やディレクトリ構造を隠す
	maskedJSON := s.docker.GetPolicy().MaskHostPaths(string(jsonBytes))

	return textResponse(maskedJSON), nil
}

// formatBlockedResponse formats a response for when a path is blocked by security policy.
// It provides detailed information about why the path was blocked and helpful hints.
//
// formatBlockedResponseはセキュリティポリシーによりパスがブロックされた場合の
// レスポンスをフォーマットします。パスがブロックされた理由と役立つヒントに関する
// 詳細情報を提供します。
func (s *Server) formatBlockedResponse(container string, path string, block *security.BlockedPath) (any, error) {
	// Build hint message based on the block reason
	// ブロック理由に基づいてヒントメッセージを構築
	hint := "This path is blocked by security policy."
	switch block.Reason {
	case "auto_imported_block", "volume_mount_to_dev_null", "tmpfs_mount":
		// Paths blocked due to secret protection
		// シークレット保護によりブロックされたパス
		hint = "This file is hidden to protect secrets. The container has access, but AI assistants do not."
	case "manual_block":
		// Paths manually blocked in configuration
		// 設定で手動でブロックされたパス
		hint = "This path has been manually blocked in the DockMCP configuration."
	case "global_pattern":
		// Paths matching global block patterns
		// グローバルなブロックパターンにマッチしたパス
		hint = "This path matches a globally blocked file pattern (e.g., .env, *.key)."
	case "devcontainer_bind_mount", "devcontainer_tmpfs_mount":
		// Paths blocked due to DevContainer configuration
		// DevContainer設定によりブロックされたパス
		hint = "This path is hidden based on DevContainer configuration to protect sensitive data."
	case "claude_code_settings_deny":
		// Paths blocked by Claude Code settings sync
		// Claude Code設定の同期によりブロックされたパス
		hint = "This path is blocked by Claude Code settings (permissions.deny). The setting is synchronized with DockMCP."
	}

	// Build the detailed response
	// 詳細なレスポンスを構築
	response := map[string]any{
		"blocked":   true,
		"container": container,
		"path":      path,
		"reason":    block.Reason,
		"details": map[string]any{
			"pattern":       block.Pattern,
			"source":        block.Source,
			"original_path": block.OriginalPath,
		},
		"hint": hint,
	}

	return jsonCodeBlockResponse("⚠️ Access Blocked", response)
}
