// Package docker provides a secure Docker client wrapper for DockMCP.
// This file defines the DockerClientInterface that allows for dependency
// injection and mock-based testing of Docker operations.
//
// dockerパッケージはDockMCP用のセキュアなDockerクライアントラッパーを提供します。
// このファイルはDocker操作の依存性注入とモックベースのテストを可能にする
// DockerClientInterfaceを定義しています。
package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// DockerClientInterface defines the interface for Docker client operations.
// This interface allows for dependency injection and mock-based testing
// without requiring a real Docker daemon connection.
//
// All implementations must enforce security policies for container access,
// command execution, and file operations.
//
// DockerClientInterfaceはDockerクライアント操作のインターフェースを定義します。
// このインターフェースは実際のDockerデーモン接続を必要とせずに、
// 依存性注入とモックベースのテストを可能にします。
//
// すべての実装はコンテナアクセス、コマンド実行、ファイル操作に対する
// セキュリティポリシーを適用する必要があります。
type DockerClientInterface interface {
	// Container Operations
	// コンテナ操作

	// ListContainers retrieves a list of all containers accessible
	// according to the security policy.
	// ListContainersはセキュリティポリシーに従ってアクセス可能な
	// すべてのコンテナのリストを取得します。
	ListContainers(ctx context.Context) ([]ContainerInfo, error)

	// GetLogs retrieves logs from a specified container.
	// The since parameter filters logs to only show entries after the given timestamp
	// (e.g., "2024-01-01T00:00:00Z") or relative time (e.g., "42m" for 42 minutes ago).
	// Pass an empty string to disable the filter.
	//
	// GetLogsは指定されたコンテナからログを取得します。
	// sinceパラメータは指定されたタイムスタンプ（例："2024-01-01T00:00:00Z"）
	// または相対時間（例："42m"で42分前）以降のログのみを表示するフィルタです。
	// フィルタを無効にするには空文字列を渡します。
	GetLogs(ctx context.Context, containerName string, tail string, since string, follow bool) (string, error)

	// GetStats retrieves resource usage statistics for a container.
	// GetStatsはコンテナのリソース使用統計を取得します。
	GetStats(ctx context.Context, containerName string) (*container.StatsResponse, error)

	// Exec executes a whitelisted command in a container.
	// Execはコンテナ内でホワイトリストに登録されたコマンドを実行します。
	Exec(ctx context.Context, containerName string, command string, dangerously bool) (*ExecResult, error)

	// InspectContainer retrieves detailed information about a container.
	// InspectContainerはコンテナの詳細情報を取得します。
	InspectContainer(ctx context.Context, containerName string) (*types.ContainerJSON, error)

	// File Operations
	// ファイル操作

	// ListFiles lists files in a container directory.
	// ListFilesはコンテナディレクトリ内のファイルを一覧表示します。
	ListFiles(ctx context.Context, containerName string, path string) (*FileAccessResult, error)

	// ReadFile reads the contents of a file from a container.
	// ReadFileはコンテナからファイルの内容を読み取ります。
	ReadFile(ctx context.Context, containerName string, path string, maxLines int) (*FileAccessResult, error)

	// Policy and Security Operations
	// ポリシーとセキュリティ操作

	// GetPolicy returns the security policy associated with this client.
	// GetPolicyはこのクライアントに関連付けられたセキュリティポリシーを返します。
	GetPolicy() *security.Policy

	// GetAllowedCommands returns the whitelisted commands for a container.
	// GetAllowedCommandsはコンテナのホワイトリストコマンドを返します。
	GetAllowedCommands(containerName string) []string

	// GetSecurityPolicy returns the current security policy configuration.
	// GetSecurityPolicyは現在のセキュリティポリシー設定を返します。
	GetSecurityPolicy() map[string]any

	// GetAllContainersWithCommands returns all containers and their commands.
	// GetAllContainersWithCommandsはすべてのコンテナとそのコマンドを返します。
	GetAllContainersWithCommands() map[string][]string

	// IsDangerousModeEnabled returns whether dangerous mode is enabled.
	// IsDangerousModeEnabledは危険モードが有効かどうかを返します。
	IsDangerousModeEnabled() bool

	// GetDangerousCommandsForContainer returns dangerous commands for a container.
	// GetDangerousCommandsForContainerはコンテナの危険コマンドを返します。
	GetDangerousCommandsForContainer(containerName string) []string

	// GetAllDangerousCommands returns all containers and their dangerous commands.
	// GetAllDangerousCommandsはすべてのコンテナとその危険コマンドを返します。
	GetAllDangerousCommands() map[string][]string

	// GetBlockedPaths returns all blocked file paths.
	// GetBlockedPathsはすべてのブロックされたファイルパスを返します。
	GetBlockedPaths() []security.BlockedPath

	// GetBlockedPathsForContainer returns blocked paths for a specific container.
	// GetBlockedPathsForContainerは特定のコンテナのブロックパスを返します。
	GetBlockedPathsForContainer(containerName string) []security.BlockedPath

	// Resource Management
	// リソース管理

	// Close closes the Docker client and releases resources.
	// CloseはDockerクライアントを閉じ、リソースを解放します。
	Close() error
}

// Verify that Client implements DockerClientInterface at compile time.
// コンパイル時にClientがDockerClientInterfaceを実装していることを検証します。
var _ DockerClientInterface = (*Client)(nil)
