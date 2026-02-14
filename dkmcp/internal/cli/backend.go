// backend.go defines the Backend interface and its implementations for container operations.
// It provides two backends: DirectBackend (direct Docker access) and HTTPBackend (via DockMCP server).
// This abstraction allows CLI commands to work both locally and remotely.
//
// backend.goはコンテナ操作のためのBackendインターフェースとその実装を定義します。
// 2つのバックエンドを提供します：DirectBackend（直接Docker接続）とHTTPBackend（DockMCPサーバー経由）。
// この抽象化により、CLIコマンドはローカルとリモートの両方で動作できます。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/client"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/docker"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// Backend represents a backend for executing container operations.
// It abstracts the underlying transport mechanism (direct Docker or HTTP/MCP).
// This interface is implemented by both DirectBackend and HTTPBackend.
//
// Backendはコンテナ操作を実行するバックエンドを表します。
// 基盤となるトランスポートメカニズム（直接DockerまたはHTTP/MCP）を抽象化します。
// このインターフェースはDirectBackendとHTTPBackendの両方で実装されています。
type Backend interface {
	// ListContainers lists all accessible containers.
	// Returns a slice of ContainerInfo for containers matching the security policy.
	//
	// ListContainersはアクセス可能なすべてのコンテナを一覧表示します。
	// セキュリティポリシーに一致するコンテナのContainerInfoスライスを返します。
	ListContainers(ctx context.Context) ([]docker.ContainerInfo, error)

	// GetLogs retrieves logs from a container.
	// The tail parameter specifies the number of lines from the end.
	// The since parameter filters logs to entries after the given timestamp (empty string disables).
	//
	// GetLogsはコンテナからログを取得します。
	// tailパラメータは末尾からの行数を指定します。
	// sinceパラメータは指定タイムスタンプ以降のログにフィルタします（空文字列で無効）。
	GetLogs(ctx context.Context, container string, tail string, since string) (string, error)

	// Exec executes a command in a container.
	// Only whitelisted commands are allowed (enforced by security policy).
	// If dangerously is true, commands from exec_dangerously list are allowed
	// with file path validation against blocked_paths.
	//
	// Execはコンテナ内でコマンドを実行します。
	// ホワイトリストに登録されたコマンドのみが許可されます（セキュリティポリシーで強制）。
	// dangerouslyがtrueの場合、exec_dangerouslyリストのコマンドが
	// blocked_pathsに対するファイルパス検証付きで許可されます。
	Exec(ctx context.Context, container string, command string, dangerously bool) (*docker.ExecResult, error)

	// Close releases any resources held by the backend.
	// Should be called when the backend is no longer needed.
	//
	// Closeはバックエンドが保持するリソースを解放します。
	// バックエンドが不要になったときに呼び出す必要があります。
	Close() error
}

// DirectBackend implements Backend using direct Docker access.
// It communicates directly with the Docker daemon via the Docker socket.
// This is used by the main CLI commands (list, logs, exec) when running on the host.
//
// DirectBackendは直接Dockerアクセスを使用してBackendを実装します。
// Dockerソケット経由でDockerデーモンと直接通信します。
// ホストで実行される際のメインCLIコマンド（list, logs, exec）で使用されます。
type DirectBackend struct {
	// docker is the Docker client instance.
	// It handles all communication with the Docker daemon.
	//
	// dockerはDockerクライアントインスタンスです。
	// Dockerデーモンとのすべての通信を処理します。
	docker *docker.Client
}

// NewDirectBackend creates a new DirectBackend with the loaded configuration.
// It initializes the security policy and Docker client.
// Returns an error if configuration loading or Docker client creation fails.
//
// NewDirectBackendは読み込まれた設定で新しいDirectBackendを作成します。
// セキュリティポリシーとDockerクライアントを初期化します。
// 設定の読み込みまたはDockerクライアント作成が失敗した場合はエラーを返します。
func NewDirectBackend() (*DirectBackend, error) {
	// Load configuration from the specified config file.
	// 指定された設定ファイルから設定を読み込みます。
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create security policy from configuration.
	// 設定からセキュリティポリシーを作成します。
	policy := security.NewPolicy(&cfg.Security)

	// Create Docker client with the security policy.
	// セキュリティポリシーでDockerクライアントを作成します。
	dockerClient, err := docker.NewClient(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DirectBackend{docker: dockerClient}, nil
}

// ListContainers returns all containers accessible according to the security policy.
// It delegates to the underlying Docker client.
//
// ListContainersはセキュリティポリシーに従ってアクセス可能なすべてのコンテナを返します。
// 基盤となるDockerクライアントに委譲します。
func (b *DirectBackend) ListContainers(ctx context.Context) ([]docker.ContainerInfo, error) {
	return b.docker.ListContainers(ctx)
}

// GetLogs retrieves logs from the specified container.
// The tail parameter controls how many lines from the end to retrieve.
// The since parameter filters logs to entries after the given timestamp.
//
// GetLogsは指定されたコンテナからログを取得します。
// tailパラメータは末尾から取得する行数を制御します。
// sinceパラメータは指定タイムスタンプ以降のログにフィルタします。
func (b *DirectBackend) GetLogs(ctx context.Context, container string, tail string, since string) (string, error) {
	return b.docker.GetLogs(ctx, container, tail, since, false)
}

// Exec executes a command in the specified container.
// The command must be whitelisted in the security policy.
// If dangerously is true, commands from exec_dangerously list are allowed.
//
// Execは指定されたコンテナでコマンドを実行します。
// コマンドはセキュリティポリシーでホワイトリストに登録されている必要があります。
// dangerouslyがtrueの場合、exec_dangerouslyリストのコマンドが許可されます。
func (b *DirectBackend) Exec(ctx context.Context, container string, command string, dangerously bool) (*docker.ExecResult, error) {
	return b.docker.Exec(ctx, container, command, dangerously)
}

// Close closes the Docker client and releases associated resources.
//
// CloseはDockerクライアントを閉じて関連リソースを解放します。
func (b *DirectBackend) Close() error {
	return b.docker.Close()
}

// HTTPBackend implements Backend using DockMCP HTTP server.
// It connects to a running DockMCP server via HTTP/SSE and uses MCP.
// This is used by 'client' subcommands for remote access (e.g., from DevContainers).
//
// HTTPBackendはDockMCP HTTPサーバーを使用してBackendを実装します。
// HTTP/SSE経由で実行中のDockMCPサーバーに接続し、MCPプロトコルを使用します。
// リモートアクセス用の'client'サブコマンドで使用されます（例：DevContainerから）。
type HTTPBackend struct {
	// client is the DockMCP HTTP/SSE client instance.
	// It handles MCP communication with the server.
	//
	// clientはDockMCP HTTP/SSEクライアントインスタンスです。
	// サーバーとのMCPプロトコル通信を処理します。
	client *client.Client
}

// NewHTTPBackend creates a new HTTPBackend connected to the specified server URL.
// It performs a health check and establishes an SSE connection before returning.
// Returns an error if health check fails or connection cannot be established.
//
// NewHTTPBackendは指定されたサーバーURLに接続する新しいHTTPBackendを作成します。
// 返す前にヘルスチェックを実行し、SSE接続を確立します。
// ヘルスチェックが失敗するか接続が確立できない場合はエラーを返します。
func NewHTTPBackend(url string) (*HTTPBackend, error) {
	return NewHTTPBackendWithSuffix(url, clientSuffix)
}

// NewHTTPBackendWithSuffix creates a new HTTPBackend with an optional client suffix.
// The suffix is appended to the client name for identification purposes.
//
// NewHTTPBackendWithSuffixはオプションのクライアントサフィックス付きで新しいHTTPBackendを作成します。
// サフィックスは識別目的でクライアント名に追加されます。
func NewHTTPBackendWithSuffix(url string, suffix string) (*HTTPBackend, error) {
	// Create a new HTTP client for the specified URL.
	// 指定されたURLの新しいHTTPクライアントを作成します。
	c := client.NewClient(url)

	// Set client suffix if provided
	// クライアントサフィックスが指定されている場合は設定
	if suffix != "" {
		c.SetClientSuffix(suffix)
	}

	// Perform health check to verify server is running.
	// サーバーが実行中であることを確認するためにヘルスチェックを実行します。
	if err := c.HealthCheck(); err != nil {
		c.Close()
		return nil, fmt.Errorf("server health check failed: %w", err)
	}

	// Establish SSE connection for MCP communication.
	// MCP通信用のSSE接続を確立します。
	if err := c.Connect(); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &HTTPBackend{client: c}, nil
}

// ListContainers retrieves the container list via the MCP 'list_containers' tool.
// It parses the JSON response and returns the container information.
//
// ListContainersはMCPの'list_containers'ツール経由でコンテナリストを取得します。
// JSONレスポンスを解析してコンテナ情報を返します。
func (b *HTTPBackend) ListContainers(ctx context.Context) ([]docker.ContainerInfo, error) {
	// Call the list_containers MCP tool.
	// list_containers MCPツールを呼び出します。
	resp, err := b.client.CallTool("list_containers", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if len(resp.Content) == 0 {
		return []docker.ContainerInfo{}, nil
	}

	// Parse JSON response into ContainerInfo slice.
	// JSONレスポンスをContainerInfoスライスに解析します。
	var containers []docker.ContainerInfo
	if err := parseJSON(resp.Content[0].Text, &containers); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return containers, nil
}

// GetLogs retrieves container logs via the MCP 'get_logs' tool.
// The container name, tail count, and since timestamp are passed as arguments.
//
// GetLogsはMCPの'get_logs'ツール経由でコンテナログを取得します。
// コンテナ名、tailカウント、sinceタイムスタンプは引数として渡されます。
func (b *HTTPBackend) GetLogs(ctx context.Context, container string, tail string, since string) (string, error) {
	// Prepare arguments for the get_logs tool.
	// get_logsツールの引数を準備します。
	arguments := map[string]interface{}{
		"container": container,
		"tail":      tail,
	}
	if since != "" {
		arguments["since"] = since
	}

	// Call the get_logs MCP tool.
	// get_logs MCPツールを呼び出します。
	resp, err := b.client.CallTool("get_logs", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if len(resp.Content) == 0 {
		return "", nil
	}

	return resp.Content[0].Text, nil
}

// parseExitCode extracts the exit code from MCP response text.
// The expected format is: "Command: ...\nExit Code: N\n\nOutput:\n..."
// Returns 0 if the exit code cannot be parsed (assumes success).
//
// parseExitCodeはMCPレスポンステキストから終了コードを抽出します。
// 期待される形式は："Command: ...\nExit Code: N\n\nOutput:\n..."
// 終了コードを解析できない場合は0を返します（成功と見なす）。
func parseExitCode(text string) int {
	re := regexp.MustCompile(`Exit Code: (\d+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		if code, err := strconv.Atoi(matches[1]); err == nil {
			return code
		}
	}
	return 0
}

// Exec executes a command via the MCP 'exec_command' tool.
// The command must be whitelisted on the server side.
// If dangerously is true, commands from exec_dangerously list are allowed.
// Returns the execution result including exit code and output.
//
// ExecはMCPの'exec_command'ツール経由でコマンドを実行します。
// コマンドはサーバー側でホワイトリストに登録されている必要があります。
// dangerouslyがtrueの場合、exec_dangerouslyリストのコマンドが許可されます。
// 終了コードと出力を含む実行結果を返します。
func (b *HTTPBackend) Exec(ctx context.Context, container string, command string, dangerously bool) (*docker.ExecResult, error) {
	// Prepare arguments for the exec_command tool.
	// exec_commandツールの引数を準備します。
	arguments := map[string]interface{}{
		"container":   container,
		"command":     command,
		"dangerously": dangerously,
	}

	// Call the exec_command MCP tool.
	// exec_command MCPツールを呼び出します。
	resp, err := b.client.CallTool("exec_command", arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Handle empty response with success status.
	// 空のレスポンスを成功ステータスで処理します。
	if len(resp.Content) == 0 {
		return &docker.ExecResult{ExitCode: 0, Output: ""}, nil
	}

	// Parse exit code from response.
	// The response format is: "Command: ...\nExit Code: N\n\nOutput:\n..."
	//
	// レスポンスから終了コードを解析します。
	// レスポンス形式は："Command: ...\nExit Code: N\n\nOutput:\n..."
	exitCode := parseExitCode(resp.Content[0].Text)
	return &docker.ExecResult{ExitCode: exitCode, Output: resp.Content[0].Text}, nil
}

// Close closes the HTTP client connection.
//
// CloseはHTTPクライアント接続を閉じます。
func (b *HTTPBackend) Close() error {
	return b.client.Close()
}

// GetStats retrieves container statistics via the MCP 'get_stats' tool.
// Returns the raw JSON response from the server.
//
// GetStatsはMCPの'get_stats'ツール経由でコンテナ統計を取得します。
// サーバーからの生のJSONレスポンスを返します。
func (b *HTTPBackend) GetStats(ctx context.Context, container string) (string, error) {
	// Prepare arguments for the get_stats tool.
	// get_statsツールの引数を準備します。
	arguments := map[string]interface{}{
		"container": container,
	}

	// Call the get_stats MCP tool.
	// get_stats MCPツールを呼び出します。
	resp, err := b.client.CallTool("get_stats", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to get stats: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if len(resp.Content) == 0 {
		return "", nil
	}

	return resp.Content[0].Text, nil
}

// InspectContainer retrieves container details via the MCP 'inspect_container' tool.
// Returns the raw JSON response from the server.
//
// InspectContainerはMCPの'inspect_container'ツール経由でコンテナ詳細を取得します。
// サーバーからの生のJSONレスポンスを返します。
func (b *HTTPBackend) InspectContainer(ctx context.Context, container string) (string, error) {
	// Prepare arguments for the inspect_container tool.
	// inspect_containerツールの引数を準備します。
	arguments := map[string]interface{}{
		"container": container,
	}

	// Call the inspect_container MCP tool.
	// inspect_container MCPツールを呼び出します。
	resp, err := b.client.CallTool("inspect_container", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	// Handle empty response.
	// 空のレスポンスを処理します。
	if len(resp.Content) == 0 {
		return "", nil
	}

	return resp.Content[0].Text, nil
}

// ListHostTools lists available host tools via the MCP 'list_host_tools' tool.
// Returns the raw text response from the server.
//
// ListHostToolsはMCPの'list_host_tools'ツール経由で利用可能なホストツールを一覧表示します。
// サーバーからのテキストレスポンスを返します。
func (b *HTTPBackend) ListHostTools(ctx context.Context) (string, error) {
	resp, err := b.client.CallTool("list_host_tools", map[string]interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to list host tools: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// GetHostToolInfo retrieves detailed info about a host tool via the MCP 'get_host_tool_info' tool.
// Returns the raw text response from the server.
//
// GetHostToolInfoはMCPの'get_host_tool_info'ツール経由でホストツールの詳細情報を取得します。
// サーバーからのテキストレスポンスを返します。
func (b *HTTPBackend) GetHostToolInfo(ctx context.Context, name string) (string, error) {
	arguments := map[string]interface{}{
		"name": name,
	}
	resp, err := b.client.CallTool("get_host_tool_info", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to get host tool info: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// RunHostTool executes a host tool via the MCP 'run_host_tool' tool.
// Returns the raw text response from the server.
//
// RunHostToolはMCPの'run_host_tool'ツール経由でホストツールを実行します。
// サーバーからのテキストレスポンスを返します。
func (b *HTTPBackend) RunHostTool(ctx context.Context, name string, args []string) (string, error) {
	arguments := map[string]interface{}{
		"name": name,
	}
	if len(args) > 0 {
		arguments["args"] = args
	}
	resp, err := b.client.CallTool("run_host_tool", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to run host tool: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// ExecHostCommand executes a host CLI command via the MCP 'exec_host_command' tool.
// Returns the raw text response from the server.
//
// ExecHostCommandはMCPの'exec_host_command'ツール経由でホストCLIコマンドを実行します。
// サーバーからのテキストレスポンスを返します。
func (b *HTTPBackend) ExecHostCommand(ctx context.Context, command string, dangerously bool) (string, error) {
	arguments := map[string]interface{}{
		"command":     command,
		"dangerously": dangerously,
	}
	resp, err := b.client.CallTool("exec_host_command", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to execute host command: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// RestartContainer restarts a container via the MCP 'restart_container' tool.
//
// RestartContainerはMCPの'restart_container'ツール経由でコンテナを再起動します。
func (b *HTTPBackend) RestartContainer(ctx context.Context, container string, timeout *int) (string, error) {
	arguments := map[string]interface{}{"container": container}
	if timeout != nil {
		arguments["timeout"] = *timeout
	}
	resp, err := b.client.CallTool("restart_container", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to restart container: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// StopContainer stops a container via the MCP 'stop_container' tool.
//
// StopContainerはMCPの'stop_container'ツール経由でコンテナを停止します。
func (b *HTTPBackend) StopContainer(ctx context.Context, container string, timeout *int) (string, error) {
	arguments := map[string]interface{}{"container": container}
	if timeout != nil {
		arguments["timeout"] = *timeout
	}
	resp, err := b.client.CallTool("stop_container", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to stop container: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// StartContainer starts a container via the MCP 'start_container' tool.
//
// StartContainerはMCPの'start_container'ツール経由でコンテナを起動します。
func (b *HTTPBackend) StartContainer(ctx context.Context, container string) (string, error) {
	arguments := map[string]interface{}{"container": container}
	resp, err := b.client.CallTool("start_container", arguments)
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", nil
	}
	return resp.Content[0].Text, nil
}

// parseJSON is a helper function to parse JSON string into a Go value.
// It wraps json.Unmarshal for convenience.
//
// parseJSONはJSON文字列をGo値に解析するヘルパー関数です。
// 便利のためにjson.Unmarshalをラップしています。
func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
