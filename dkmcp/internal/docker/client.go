// Package docker provides a secure Docker client wrapper for DockMCP.
// This package implements controlled access to Docker containers through
// a security policy layer, enabling AI assistants to interact with containers
// in a safe and restricted manner.
//
// The package wraps the official Docker SDK client and enforces security
// policies defined in the security package. All container operations are
// checked against the policy before execution.
//
// dockerパッケージはDockMCP用のセキュアなDockerクライアントラッパーを提供します。
// このパッケージはセキュリティポリシーレイヤーを通じてDockerコンテナへの
// 制御されたアクセスを実装し、AIアシスタントが安全かつ制限された方法で
// コンテナと対話できるようにします。
//
// このパッケージは公式Docker SDKクライアントをラップし、securityパッケージで
// 定義されたセキュリティポリシーを適用します。すべてのコンテナ操作は
// 実行前にポリシーに対してチェックされます。
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// Client wraps the Docker client with security policy enforcement.
// It provides methods for container operations that are filtered and
// controlled by the associated security policy.
//
// All operations check the security policy before accessing Docker,
// ensuring that only permitted containers and commands can be accessed.
//
// ClientはセキュリティポリシーをDockerクライアントにラップします。
// 関連付けられたセキュリティポリシーによってフィルタリングおよび
// 制御されるコンテナ操作用のメソッドを提供します。
//
// すべての操作はDockerにアクセスする前にセキュリティポリシーをチェックし、
// 許可されたコンテナとコマンドのみがアクセス可能であることを保証します。
type Client struct {
	// docker is the underlying Docker SDK client for container operations.
	// dockerはコンテナ操作用の基盤となるDocker SDKクライアントです。
	docker *client.Client

	// policy defines the security rules for container access.
	// policyはコンテナアクセスのセキュリティルールを定義します。
	policy *security.Policy
}

// NewClient creates a new Docker client with security policy enforcement.
// It initializes the Docker SDK client using environment variables
// (DOCKER_HOST, DOCKER_API_VERSION, etc.) and wraps it with the
// provided security policy.
//
// The policy parameter must not be nil - this ensures that all
// container operations are always subject to security checks.
//
// Returns an error if the policy is nil or if the Docker client
// cannot be created (e.g., Docker daemon is not running).
//
// NewClientはセキュリティポリシーを適用した新しいDockerクライアントを作成します。
// 環境変数（DOCKER_HOST、DOCKER_API_VERSIONなど）を使用してDocker SDKクライアントを
// 初期化し、提供されたセキュリティポリシーでラップします。
//
// policyパラメータはnilであってはなりません - これにより、すべての
// コンテナ操作が常にセキュリティチェックの対象となることが保証されます。
//
// ポリシーがnilの場合、またはDockerクライアントを作成できない場合
// （例：Dockerデーモンが実行されていない場合）にエラーを返します。
func NewClient(policy *security.Policy) (*Client, error) {
	// Validate that a security policy is provided.
	// This is a critical check - we never allow operations without policy.
	// セキュリティポリシーが提供されていることを検証します。
	// これは重要なチェックです - ポリシーなしの操作は許可しません。
	if policy == nil {
		return nil, fmt.Errorf("security policy cannot be nil")
	}

	// Create the Docker SDK client with environment-based configuration.
	// WithAPIVersionNegotiation ensures compatibility with different Docker versions.
	// 環境ベースの設定でDocker SDKクライアントを作成します。
	// WithAPIVersionNegotiationは異なるDockerバージョンとの互換性を確保します。
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Client{
		docker: dockerClient,
		policy: policy,
	}, nil
}

// Close closes the Docker client and releases associated resources.
// This should be called when the client is no longer needed.
//
// Closeは Dockerクライアントを閉じ、関連リソースを解放します。
// クライアントが不要になったときに呼び出す必要があります。
func (c *Client) Close() error {
	return c.docker.Close()
}

// GetPolicy returns the security policy associated with this client.
// This can be used to access policy features like output masking.
//
// GetPolicyはこのクライアントに関連付けられたセキュリティポリシーを返します。
// これは出力マスキングなどのポリシー機能にアクセスするために使用できます。
func (c *Client) GetPolicy() *security.Policy {
	return c.policy
}

// ContainerInfo represents simplified container information returned
// by the ListContainers method. It contains the essential details
// about a container that are safe to expose.
//
// ContainerInfoはListContainersメソッドによって返される
// 簡略化されたコンテナ情報を表します。公開しても安全な
// コンテナに関する重要な詳細を含みます。
type ContainerInfo struct {
	// ID is the truncated container ID (first 12 characters).
	// IDは短縮されたコンテナID（最初の12文字）です。
	ID string `json:"id"`

	// Name is the container name without the leading slash.
	// Nameは先頭のスラッシュを除いたコンテナ名です。
	Name string `json:"name"`

	// Image is the name of the Docker image used by the container.
	// ImageはコンテナがつかっているDockerイメージの名前です。
	Image string `json:"image"`

	// State is the current state (e.g., "running", "exited").
	// Stateは現在の状態（例："running"、"exited"）です。
	State string `json:"state"`

	// Status is a human-readable status string (e.g., "Up 2 hours").
	// Statusは人が読める状態文字列（例："Up 2 hours"）です。
	Status string `json:"status"`

	// Created is the Unix timestamp when the container was created.
	// Createdはコンテナが作成されたUnixタイムスタンプです。
	Created int64 `json:"created"`

	// Labels contains the container's labels as key-value pairs.
	// Labelsはコンテナのラベルをキーと値のペアとして含みます。
	Labels map[string]string `json:"labels,omitempty"`

	// Ports contains the port mappings as formatted strings.
	// Example: ["0.0.0.0:80->80/tcp", "443/tcp"]
	// Portsはフォーマットされた文字列としてのポートマッピングを含みます。
	// 例: ["0.0.0.0:80->80/tcp", "443/tcp"]
	Ports []string `json:"ports,omitempty"`
}

// ListContainers retrieves a list of all containers that are accessible
// according to the security policy. It filters out containers that
// don't match the allowed container patterns defined in the policy.
//
// This method requires the "inspect" permission to be enabled in the policy.
//
// Returns a slice of ContainerInfo for accessible containers, or an error
// if the permission is denied or if the Docker API call fails.
//
// ListContainersはセキュリティポリシーに従ってアクセス可能な
// すべてのコンテナのリストを取得します。ポリシーで定義された
// 許可コンテナパターンに一致しないコンテナをフィルタリングします。
//
// このメソッドはポリシーで"inspect"権限が有効になっている必要があります。
//
// アクセス可能なコンテナのContainerInfoスライスを返します。
// 権限が拒否された場合やDocker API呼び出しが失敗した場合はエラーを返します。
func (c *Client) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	// Check if the policy allows container inspection.
	// ポリシーがコンテナ検査を許可しているかチェックします。
	if !c.policy.CanInspect() {
		return nil, fmt.Errorf("inspect permission denied")
	}

	// Retrieve all containers from Docker (both running and stopped).
	// Dockerからすべてのコンテナを取得します（実行中と停止中の両方）。
	containers, err := c.docker.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter containers based on security policy and build result slice.
	// セキュリティポリシーに基づいてコンテナをフィルタリングし、結果スライスを構築します。
	var result []ContainerInfo
	for _, ctr := range containers {
		// Container names from Docker API have a leading slash that we remove.
		// Docker APIからのコンテナ名には先頭にスラッシュがあるので削除します。
		name := strings.TrimPrefix(ctr.Names[0], "/")

		// Check if container is accessible according to security policy.
		// Skip containers that don't match allowed patterns.
		// セキュリティポリシーに従ってコンテナがアクセス可能かチェックします。
		// 許可パターンに一致しないコンテナはスキップします。
		if !c.policy.CanAccessContainer(name) {
			continue
		}

		// Add accessible container to results with truncated ID.
		// アクセス可能なコンテナを短縮IDで結果に追加します。
		result = append(result, ContainerInfo{
			ID:      ctr.ID[:12],
			Name:    name,
			Image:   ctr.Image,
			State:   ctr.State,
			Status:  ctr.Status,
			Created: ctr.Created,
			Labels:  ctr.Labels,
			Ports:   formatPorts(ctr.Ports),
		})
	}

	return result, nil
}

// formatPorts converts Docker SDK port bindings to human-readable strings.
// Example outputs: "0.0.0.0:80->80/tcp", "443/tcp", "0.0.0.0:8080->80/tcp, 80/tcp"
//
// formatPortsはDocker SDKのポートバインディングを人が読める文字列に変換します。
// 出力例: "0.0.0.0:80->80/tcp", "443/tcp", "0.0.0.0:8080->80/tcp, 80/tcp"
func formatPorts(ports []types.Port) []string {
	if len(ports) == 0 {
		return nil
	}

	var result []string
	for _, p := range ports {
		var portStr string
		if p.PublicPort != 0 {
			// Port is published to host
			// ポートがホストに公開されている
			portStr = fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
		} else {
			// Port is exposed but not published
			// ポートは公開されているが、ホストにはバインドされていない
			portStr = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		result = append(result, portStr)
	}

	return result
}

// GetLogs retrieves the logs from a specified container.
// The logs include both stdout and stderr with timestamps.
//
// Parameters:
//   - containerName: The name or ID of the container
//   - tail: Number of lines to retrieve from the end (e.g., "100", "all")
//   - since: Show logs since timestamp (e.g., "2024-01-01T00:00:00Z") or relative (e.g., "42m"). Empty string disables filter.
//   - follow: Whether to stream logs continuously (not typically used in MCP)
//
// This method requires both "logs" permission and access to the
// specified container according to the security policy.
//
// GetLogsは指定されたコンテナからログを取得します。
// ログには標準出力と標準エラー出力の両方がタイムスタンプ付きで含まれます。
//
// パラメータ:
//   - containerName: コンテナの名前またはID
//   - tail: 末尾から取得する行数（例："100"、"all"）
//   - since: ログ表示開始タイムスタンプ（例："2024-01-01T00:00:00Z"）または相対時間（例："42m"）。空文字列でフィルタ無効。
//   - follow: ログを継続的にストリームするかどうか（MCPでは通常使用しない）
//
// このメソッドはセキュリティポリシーに従って"logs"権限と
// 指定されたコンテナへのアクセスの両方が必要です。
func (c *Client) GetLogs(ctx context.Context, containerName string, tail string, since string, follow bool) (string, error) {
	// Verify logs permission is granted by policy.
	// ポリシーによってログ権限が付与されているか確認します。
	if !c.policy.CanGetLogs() {
		return "", fmt.Errorf("logs permission denied")
	}

	// Verify the specific container is accessible.
	// 特定のコンテナがアクセス可能か確認します。
	if !c.policy.CanAccessContainer(containerName) {
		return "", fmt.Errorf("access denied to container: %s", containerName)
	}

	// Configure log retrieval options.
	// ログ取得オプションを設定します。
	options := container.LogsOptions{
		ShowStdout: true,   // Include standard output / 標準出力を含める
		ShowStderr: true,   // Include standard error / 標準エラー出力を含める
		Tail:       tail,   // Number of lines from end / 末尾からの行数
		Since:      since,  // Show logs since timestamp / 指定タイムスタンプ以降のログを表示
		Follow:     follow, // Stream continuously / 継続的にストリーム
		Timestamps: true,   // Include timestamps / タイムスタンプを含める
	}

	// Retrieve logs from the Docker API.
	// Docker APIからログを取得します。
	logs, err := c.docker.ContainerLogs(ctx, containerName, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	// Read all log content into a string buffer.
	// すべてのログ内容を文字列バッファに読み込みます。
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, logs); err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// GetStats retrieves resource usage statistics for a container.
// This includes CPU, memory, network, and I/O statistics.
//
// The statistics are a point-in-time snapshot (not streaming).
//
// This method requires both "stats" permission and access to the
// specified container according to the security policy.
//
// GetStatsはコンテナのリソース使用統計を取得します。
// これにはCPU、メモリ、ネットワーク、I/O統計が含まれます。
//
// 統計はポイントインタイムのスナップショットです（ストリーミングではありません）。
//
// このメソッドはセキュリティポリシーに従って"stats"権限と
// 指定されたコンテナへのアクセスの両方が必要です。
func (c *Client) GetStats(ctx context.Context, containerName string) (*container.StatsResponse, error) {
	// Verify stats permission is granted by policy.
	// ポリシーによって統計権限が付与されているか確認します。
	if !c.policy.CanGetStats() {
		return nil, fmt.Errorf("stats permission denied")
	}

	// Verify the specific container is accessible.
	// 特定のコンテナがアクセス可能か確認します。
	if !c.policy.CanAccessContainer(containerName) {
		return nil, fmt.Errorf("access denied to container: %s", containerName)
	}

	// Get one-shot stats (stream=false means single response).
	// ワンショット統計を取得します（stream=falseは単一レスポンスを意味します）。
	stats, err := c.docker.ContainerStats(ctx, containerName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer stats.Body.Close()

	// Decode the JSON response into StatsResponse struct.
	// JSONレスポンスをStatsResponse構造体にデコードします。
	var result container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &result, nil
}

// ExecResult represents the result of executing a command in a container.
// It contains the exit code and the combined stdout/stderr output.
//
// ExecResultはコンテナ内でコマンドを実行した結果を表します。
// 終了コードと標準出力/標準エラー出力の組み合わせを含みます。
type ExecResult struct {
	// ExitCode is the exit status of the command (0 typically means success).
	// ExitCodeはコマンドの終了ステータスです（0は通常成功を意味します）。
	ExitCode int `json:"exit_code"`

	// Output contains the combined stdout and stderr from the command.
	// Outputはコマンドからの標準出力と標準エラー出力の組み合わせを含みます。
	Output string `json:"output"`
}

// Exec executes a whitelisted command in a container.
// The command must be explicitly allowed in the security policy's
// exec_whitelist for the specific container.
//
// This is a critical security feature - it prevents arbitrary command
// execution while still allowing specific, pre-approved commands
// (e.g., "npm test", "npm run lint").
//
// The command string is parsed by splitting on whitespace. For complex
// commands with quoted arguments, this parsing may need enhancement.
//
// Execはコンテナ内でホワイトリストに登録されたコマンドを実行します。
// コマンドは特定のコンテナに対するセキュリティポリシーのexec_whitelistで
// 明示的に許可されている必要があります。
//
// これは重要なセキュリティ機能です - 任意のコマンド実行を防止しながら、
// 特定の事前承認されたコマンド（例："npm test"、"npm run lint"）を
// 許可します。
//
// コマンド文字列は空白で分割して解析されます。引用符付き引数を含む
// 複雑なコマンドの場合、この解析は強化が必要かもしれません。
func (c *Client) Exec(ctx context.Context, containerName string, command string, dangerously bool) (*ExecResult, error) {
	// Check if the command is allowed for this container.
	// The policy validates container access and command permissions.
	// このコンテナに対してコマンドが許可されているかチェックします。
	// ポリシーはコンテナアクセスとコマンド権限を検証します。
	var allowed bool
	var err error

	if dangerously {
		// Dangerous mode: allows commands from exec_dangerously list with path blocking
		// 危険モード: パスブロック付きでexec_dangerouslyリストのコマンドを許可
		allowed, err = c.policy.CanExecDangerously(containerName, command)
	} else {
		// Normal mode: only whitelisted commands
		// 通常モード: ホワイトリストのコマンドのみ
		allowed, err = c.policy.CanExec(containerName, command)
	}

	if err != nil || !allowed {
		if err == nil {
			err = fmt.Errorf("exec permission denied")
		}
		return nil, err
	}

	// Parse the command string into individual arguments.
	// コマンド文字列を個々の引数に解析します。
	cmdParts := parseCommand(command)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Delegate to execInternal for actual Docker execution.
	// 実際のDocker実行をexecInternalに委譲します。
	return c.execInternal(ctx, containerName, cmdParts)
}

// parseCommand parses a command string into individual argument parts.
// It splits the command by whitespace (spaces, tabs, newlines).
//
// Design Decision: This implementation intentionally uses simple whitespace splitting.
// Complex commands with quotes (e.g., `echo "hello world"`) are NOT supported.
// This is by design - whitelisted commands should be simple, single-purpose commands
// for security reasons. If you need complex shell parsing, the command is likely
// too complex to be safely whitelisted.
//
// parseCommandはコマンド文字列を個々の引数部分に解析します。
// コマンドを空白（スペース、タブ、改行）で分割します。
//
// 設計上の決定: この実装は意図的に単純な空白分割を使用しています。
// 引用符を含む複雑なコマンド（例: `echo "hello world"`）はサポートしません。
// これは意図的な設計です - ホワイトリストコマンドはセキュリティ上の理由から
// シンプルで単一目的のコマンドであるべきです。複雑なシェル解析が必要な場合、
// そのコマンドはホワイトリストに登録するには複雑すぎる可能性があります。
func parseCommand(command string) []string {
	// Split by whitespace - handles spaces, tabs, and multiple spaces.
	// 空白で分割 - スペース、タブ、複数のスペースを処理します。
	parts := strings.Fields(command)
	return parts
}

// InspectContainer retrieves detailed information about a specific container.
// This includes configuration, network settings, mount points, and more.
//
// The returned ContainerJSON contains comprehensive container details
// from the Docker API.
//
// This method requires both "inspect" permission and access to the
// specified container according to the security policy.
//
// InspectContainerは特定のコンテナに関する詳細情報を取得します。
// これには設定、ネットワーク設定、マウントポイントなどが含まれます。
//
// 返されるContainerJSONにはDocker APIからの包括的な
// コンテナ詳細が含まれます。
//
// このメソッドはセキュリティポリシーに従って"inspect"権限と
// 指定されたコンテナへのアクセスの両方が必要です。
func (c *Client) InspectContainer(ctx context.Context, containerName string) (*types.ContainerJSON, error) {
	// Verify inspect permission is granted by policy.
	// ポリシーによって検査権限が付与されているか確認します。
	if !c.policy.CanInspect() {
		return nil, fmt.Errorf("inspect permission denied")
	}

	// Verify the specific container is accessible.
	// 特定のコンテナがアクセス可能か確認します。
	if !c.policy.CanAccessContainer(containerName) {
		return nil, fmt.Errorf("access denied to container: %s", containerName)
	}

	// Retrieve detailed container information from Docker API.
	// Docker APIから詳細なコンテナ情報を取得します。
	info, err := c.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &info, nil
}

// GetAllowedCommands returns the list of commands that are whitelisted
// for execution in the specified container.
//
// This is useful for discovering what commands an AI assistant can
// run in a particular container.
//
// GetAllowedCommandsは指定されたコンテナで実行がホワイトリストに
// 登録されているコマンドのリストを返します。
//
// これはAIアシスタントが特定のコンテナで実行できるコマンドを
// 発見するのに役立ちます。
func (c *Client) GetAllowedCommands(containerName string) []string {
	return c.policy.GetAllowedCommands(containerName)
}

// GetSecurityPolicy returns the current security policy configuration
// as a map. This provides visibility into the active security settings.
//
// The returned map includes mode, permissions, allowed containers,
// and other policy details.
//
// GetSecurityPolicyは現在のセキュリティポリシー設定をマップとして返します。
// これにより、アクティブなセキュリティ設定への可視性が提供されます。
//
// 返されるマップにはモード、権限、許可コンテナ、
// その他のポリシー詳細が含まれます。
func (c *Client) GetSecurityPolicy() map[string]any {
	return c.policy.GetSecurityPolicy()
}

// GetAllContainersWithCommands returns a map of all containers and
// their respective whitelisted commands.
//
// The map keys are container names (or patterns like "securenote-*"),
// and values are slices of allowed command strings.
//
// GetAllContainersWithCommandsはすべてのコンテナとそれぞれの
// ホワイトリストコマンドのマップを返します。
//
// マップのキーはコンテナ名（または"securenote-*"のようなパターン）、
// 値は許可されたコマンド文字列のスライスです。
func (c *Client) GetAllContainersWithCommands() map[string][]string {
	return c.policy.GetAllContainersWithCommands()
}

// IsDangerousModeEnabled returns whether dangerous mode is globally enabled.
//
// When dangerous mode is enabled, commands from exec_dangerously list can be
// executed using the dangerously=true parameter.
//
// IsDangerousModeEnabledは危険モードがグローバルに有効かどうかを返します。
//
// 危険モードが有効な場合、dangerously=trueパラメータを使用して
// exec_dangerouslyリストのコマンドを実行できます。
func (c *Client) IsDangerousModeEnabled() bool {
	return c.policy.IsDangerousModeEnabled()
}

// GetDangerousCommandsForContainer returns the dangerous commands allowed for a container.
// Includes both container-specific commands and global commands (*).
//
// This is useful for discovering what commands can be executed with
// dangerously=true in a particular container.
//
// GetDangerousCommandsForContainerはコンテナで許可される危険コマンドを返します。
// コンテナ固有のコマンドとグローバルコマンド（*）の両方を含みます。
//
// これはdangerously=trueで特定のコンテナで実行できるコマンドを
// 発見するのに役立ちます。
func (c *Client) GetDangerousCommandsForContainer(containerName string) []string {
	return c.policy.GetDangerousCommandsForContainer(containerName)
}

// GetAllDangerousCommands returns a map of all containers and their dangerous commands.
// Returns a map of container name to list of dangerous commands.
// Includes the special "*" entry for global commands.
//
// GetAllDangerousCommandsはすべてのコンテナとそれぞれの危険コマンドのマップを返します。
// コンテナ名から危険コマンドリストへのマップを返します。
// グローバルコマンドの特別な"*"エントリを含みます。
func (c *Client) GetAllDangerousCommands() map[string][]string {
	return c.policy.GetAllDangerousCommands()
}

// FileAccessResult represents the result of a file access operation
// (listing or reading files) in a container.
//
// It indicates whether the operation succeeded, was blocked by policy,
// or failed with an error.
//
// FileAccessResultはコンテナ内でのファイルアクセス操作
// （ファイルの一覧表示または読み取り）の結果を表します。
//
// 操作が成功したか、ポリシーによってブロックされたか、
// エラーで失敗したかを示します。
type FileAccessResult struct {
	// Success indicates whether the file operation completed successfully.
	// Successはファイル操作が正常に完了したかどうかを示します。
	Success bool `json:"success"`

	// Data contains the file content or directory listing if successful.
	// Dataは成功した場合のファイル内容またはディレクトリ一覧を含みます。
	Data string `json:"data,omitempty"`

	// Blocked indicates if the path was blocked by security policy.
	// Blockedはパスがセキュリティポリシーによってブロックされたかを示します。
	Blocked bool `json:"blocked,omitempty"`

	// Block contains details about why the path was blocked.
	// Blockはパスがブロックされた理由の詳細を含みます。
	Block *security.BlockedPath `json:"block_info,omitempty"`

	// Error contains the error message if the operation failed.
	// Errorは操作が失敗した場合のエラーメッセージを含みます。
	Error string `json:"error,omitempty"`
}

// ListFiles lists files and directories in a container at the specified path.
// The path is checked against the security policy's blocked paths before access.
//
// If the path is blocked, the result will indicate the block with details
// about why access was denied (e.g., security-sensitive directory).
//
// This method internally executes "ls -la" in the container.
//
// ListFilesはコンテナ内の指定されたパスにあるファイルとディレクトリを一覧表示します。
// アクセス前にパスはセキュリティポリシーのブロックパスに対してチェックされます。
//
// パスがブロックされている場合、結果はアクセスが拒否された理由
// （例：セキュリティ上重要なディレクトリ）の詳細とともにブロックを示します。
//
// このメソッドは内部的にコンテナ内で"ls -la"を実行します。
func (c *Client) ListFiles(ctx context.Context, containerName string, path string) (*FileAccessResult, error) {
	// Verify the container is accessible according to policy.
	// ポリシーに従ってコンテナがアクセス可能か確認します。
	if !c.policy.CanAccessContainer(containerName) {
		return nil, fmt.Errorf("access denied to container: %s", containerName)
	}

	// Check if the requested path is blocked by security policy.
	// This prevents access to sensitive directories like /etc/secrets.
	// 要求されたパスがセキュリティポリシーによってブロックされているかチェックします。
	// これは/etc/secretsのような機密ディレクトリへのアクセスを防ぎます。
	if blocked := c.policy.IsPathBlocked(containerName, path); blocked != nil {
		return &FileAccessResult{
			Success: false,
			Blocked: true,
			Block:   blocked,
		}, nil
	}

	// Execute ls command internally (bypasses whitelist check).
	// lsコマンドを内部的に実行します（ホワイトリストチェックをバイパス）。
	result, err := c.execInternal(ctx, containerName, []string{"ls", "-la", path})
	if err != nil {
		return &FileAccessResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &FileAccessResult{
		Success: true,
		Data:    result.Output,
	}, nil
}

// ReadFile reads the contents of a file from a container.
// The path is checked against the security policy's blocked paths before access.
//
// Parameters:
//   - containerName: The name or ID of the container
//   - path: The absolute path to the file in the container
//   - maxLines: Maximum number of lines to read (0 = all lines)
//
// If maxLines > 0, uses "head -n" to limit output; otherwise uses "cat".
//
// ReadFileはコンテナからファイルの内容を読み取ります。
// アクセス前にパスはセキュリティポリシーのブロックパスに対してチェックされます。
//
// パラメータ:
//   - containerName: コンテナの名前またはID
//   - path: コンテナ内のファイルへの絶対パス
//   - maxLines: 読み取る最大行数（0 = すべての行）
//
// maxLines > 0の場合は"head -n"を使用して出力を制限し、そうでなければ"cat"を使用します。
func (c *Client) ReadFile(ctx context.Context, containerName string, path string, maxLines int) (*FileAccessResult, error) {
	// Verify the container is accessible according to policy.
	// ポリシーに従ってコンテナがアクセス可能か確認します。
	if !c.policy.CanAccessContainer(containerName) {
		return nil, fmt.Errorf("access denied to container: %s", containerName)
	}

	// Check if the requested path is blocked by security policy.
	// 要求されたパスがセキュリティポリシーによってブロックされているかチェックします。
	if blocked := c.policy.IsPathBlocked(containerName, path); blocked != nil {
		return &FileAccessResult{
			Success: false,
			Blocked: true,
			Block:   blocked,
		}, nil
	}

	// Build the appropriate command based on maxLines parameter.
	// maxLinesパラメータに基づいて適切なコマンドを構築します。
	var cmd []string
	if maxLines > 0 {
		// Use head to limit output to specified number of lines.
		// headを使用して出力を指定行数に制限します。
		cmd = []string{"head", "-n", fmt.Sprintf("%d", maxLines), path}
	} else {
		// Read the entire file with cat.
		// catでファイル全体を読み取ります。
		cmd = []string{"cat", path}
	}

	// Execute the read command internally.
	// 読み取りコマンドを内部的に実行します。
	result, err := c.execInternal(ctx, containerName, cmd)
	if err != nil {
		return &FileAccessResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &FileAccessResult{
		Success: true,
		Data:    result.Output,
	}, nil
}

// execInternal executes a command in a container without checking
// the command whitelist. This is used internally for file operations
// (ls, cat, head) that are controlled by path blocking instead.
//
// IMPORTANT: This method should only be used for trusted internal
// commands. User-provided commands must go through the Exec method
// which validates against the whitelist.
//
// execInternalはコマンドホワイトリストをチェックせずにコンテナ内で
// コマンドを実行します。これは代わりにパスブロッキングで制御される
// ファイル操作（ls、cat、head）に内部的に使用されます。
//
// 重要: このメソッドは信頼できる内部コマンドにのみ使用すべきです。
// ユーザー提供のコマンドはホワイトリストに対して検証する
// Execメソッドを経由する必要があります。
func (c *Client) execInternal(ctx context.Context, containerName string, cmd []string) (*ExecResult, error) {
	// Configure exec with stdout/stderr capture.
	// 標準出力/標準エラー出力キャプチャでexecを設定します。
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	// Create exec instance in the container.
	// コンテナ内にexecインスタンスを作成します。
	execID, err := c.docker.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to the exec instance to capture output.
	// 出力をキャプチャするためにexecインスタンスにアタッチします。
	resp, err := c.docker.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	// Read all output from the command.
	// コマンドからすべての出力を読み取ります。
	output := new(strings.Builder)
	if _, err := io.Copy(output, resp.Reader); err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// Get the exit code from the completed exec.
	// 完了したexecから終了コードを取得します。
	inspect, err := c.docker.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return &ExecResult{
		ExitCode: inspect.ExitCode,
		Output:   output.String(),
	}, nil
}

// GetBlockedPaths returns all blocked file paths defined in the security policy.
// These are paths that cannot be accessed through ListFiles or ReadFile.
//
// GetBlockedPathsはセキュリティポリシーで定義されたすべての
// ブロックされたファイルパスを返します。
// これらはListFilesまたはReadFileを通じてアクセスできないパスです。
func (c *Client) GetBlockedPaths() []security.BlockedPath {
	return c.policy.GetBlockedPaths()
}

// GetBlockedPathsForContainer returns blocked paths for a specific container.
// This includes both container-specific and global blocked paths.
//
// GetBlockedPathsForContainerは特定のコンテナのブロックパスを返します。
// これにはコンテナ固有とグローバルの両方のブロックパスが含まれます。
func (c *Client) GetBlockedPathsForContainer(containerName string) []security.BlockedPath {
	return c.policy.GetBlockedPathsForContainer(containerName)
}

// InitBlockedPaths initializes the blocked paths manager with a list of containers.
// This should be called during startup to set up path blocking.
//
// InitBlockedPathsはコンテナのリストでブロックパスマネージャを初期化します。
// パスブロッキングを設定するために起動時に呼び出す必要があります。
func (c *Client) InitBlockedPaths(containers []string) error {
	return c.policy.InitBlockedPaths(containers)
}
