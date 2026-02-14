// Package docker provides a mock Docker client for testing.
// This file contains the MockClient implementation which allows testing
// MCP handlers and other components without requiring a real Docker daemon.
//
// dockerパッケージはテスト用のモックDockerクライアントを提供します。
// このファイルは実際のDockerデーモンを必要とせずにMCPハンドラーや
// 他のコンポーネントをテストできるMockClient実装を含んでいます。
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// MockClient is a mock implementation of DockerClientInterface for testing.
// It allows customizing the behavior of each method through function fields.
// If a function field is not set, the method returns a default value or error.
//
// MockClientはテスト用のDockerClientInterfaceのモック実装です。
// 各メソッドの動作を関数フィールドを通じてカスタマイズできます。
// 関数フィールドが設定されていない場合、メソッドはデフォルト値またはエラーを返します。
type MockClient struct {
	// Function fields for customizing behavior
	// 動作をカスタマイズするための関数フィールド

	// ListContainersFunc is called by ListContainers if set.
	// ListContainersFuncが設定されている場合、ListContainersから呼び出されます。
	ListContainersFunc func(ctx context.Context) ([]ContainerInfo, error)

	// GetLogsFunc is called by GetLogs if set.
	// GetLogsFuncが設定されている場合、GetLogsから呼び出されます。
	GetLogsFunc func(ctx context.Context, containerName string, tail string, since string, follow bool) (string, error)

	// GetStatsFunc is called by GetStats if set.
	// GetStatsFuncが設定されている場合、GetStatsから呼び出されます。
	GetStatsFunc func(ctx context.Context, containerName string) (*container.StatsResponse, error)

	// ExecFunc is called by Exec if set.
	// ExecFuncが設定されている場合、Execから呼び出されます。
	ExecFunc func(ctx context.Context, containerName string, command string, dangerously bool) (*ExecResult, error)

	// InspectContainerFunc is called by InspectContainer if set.
	// InspectContainerFuncが設定されている場合、InspectContainerから呼び出されます。
	InspectContainerFunc func(ctx context.Context, containerName string) (*types.ContainerJSON, error)

	// RestartContainerFunc is called by RestartContainer if set.
	// RestartContainerFuncが設定されている場合、RestartContainerから呼び出されます。
	RestartContainerFunc func(ctx context.Context, containerName string, timeout *int) error

	// StopContainerFunc is called by StopContainer if set.
	// StopContainerFuncが設定されている場合、StopContainerから呼び出されます。
	StopContainerFunc func(ctx context.Context, containerName string, timeout *int) error

	// StartContainerFunc is called by StartContainer if set.
	// StartContainerFuncが設定されている場合、StartContainerから呼び出されます。
	StartContainerFunc func(ctx context.Context, containerName string) error

	// ListFilesFunc is called by ListFiles if set.
	// ListFilesFuncが設定されている場合、ListFilesから呼び出されます。
	ListFilesFunc func(ctx context.Context, containerName string, path string) (*FileAccessResult, error)

	// ReadFileFunc is called by ReadFile if set.
	// ReadFileFuncが設定されている場合、ReadFileから呼び出されます。
	ReadFileFunc func(ctx context.Context, containerName string, path string, maxLines int) (*FileAccessResult, error)

	// policy is the security policy used by this mock client.
	// policyはこのモッククライアントが使用するセキュリティポリシーです。
	policy *security.Policy
}

// NewMockClient creates a new MockClient with the given security policy.
// The policy is used for GetPolicy and related methods.
//
// NewMockClientは指定されたセキュリティポリシーで新しいMockClientを作成します。
// ポリシーはGetPolicyおよび関連メソッドで使用されます。
func NewMockClient(policy *security.Policy) *MockClient {
	return &MockClient{
		policy: policy,
	}
}

// ListContainers returns the result of ListContainersFunc if set,
// otherwise returns an empty slice.
//
// ListContainersはListContainersFuncが設定されている場合はその結果を返し、
// そうでなければ空のスライスを返します。
func (m *MockClient) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	if m.ListContainersFunc != nil {
		return m.ListContainersFunc(ctx)
	}
	return []ContainerInfo{}, nil
}

// GetLogs returns the result of GetLogsFunc if set,
// otherwise returns an error.
//
// GetLogsはGetLogsFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) GetLogs(ctx context.Context, containerName string, tail string, since string, follow bool) (string, error) {
	if m.GetLogsFunc != nil {
		return m.GetLogsFunc(ctx, containerName, tail, since, follow)
	}
	return "", fmt.Errorf("GetLogs not implemented in mock")
}

// GetStats returns the result of GetStatsFunc if set,
// otherwise returns an error.
//
// GetStatsはGetStatsFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) GetStats(ctx context.Context, containerName string) (*container.StatsResponse, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx, containerName)
	}
	return nil, fmt.Errorf("GetStats not implemented in mock")
}

// Exec returns the result of ExecFunc if set,
// otherwise returns an error.
//
// ExecはExecFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) Exec(ctx context.Context, containerName string, command string, dangerously bool) (*ExecResult, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, containerName, command, dangerously)
	}
	return nil, fmt.Errorf("Exec not implemented in mock")
}

// InspectContainer returns the result of InspectContainerFunc if set,
// otherwise returns an error.
//
// InspectContainerはInspectContainerFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) InspectContainer(ctx context.Context, containerName string) (*types.ContainerJSON, error) {
	if m.InspectContainerFunc != nil {
		return m.InspectContainerFunc(ctx, containerName)
	}
	return nil, fmt.Errorf("InspectContainer not implemented in mock")
}

// RestartContainer returns the result of RestartContainerFunc if set,
// otherwise returns an error.
func (m *MockClient) RestartContainer(ctx context.Context, containerName string, timeout *int) error {
	if m.RestartContainerFunc != nil {
		return m.RestartContainerFunc(ctx, containerName, timeout)
	}
	return fmt.Errorf("RestartContainer not implemented in mock")
}

// StopContainer returns the result of StopContainerFunc if set,
// otherwise returns an error.
func (m *MockClient) StopContainer(ctx context.Context, containerName string, timeout *int) error {
	if m.StopContainerFunc != nil {
		return m.StopContainerFunc(ctx, containerName, timeout)
	}
	return fmt.Errorf("StopContainer not implemented in mock")
}

// StartContainer returns the result of StartContainerFunc if set,
// otherwise returns an error.
func (m *MockClient) StartContainer(ctx context.Context, containerName string) error {
	if m.StartContainerFunc != nil {
		return m.StartContainerFunc(ctx, containerName)
	}
	return fmt.Errorf("StartContainer not implemented in mock")
}

// ListFiles returns the result of ListFilesFunc if set,
// otherwise returns an error.
//
// ListFilesはListFilesFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) ListFiles(ctx context.Context, containerName string, path string) (*FileAccessResult, error) {
	if m.ListFilesFunc != nil {
		return m.ListFilesFunc(ctx, containerName, path)
	}
	return nil, fmt.Errorf("ListFiles not implemented in mock")
}

// ReadFile returns the result of ReadFileFunc if set,
// otherwise returns an error.
//
// ReadFileはReadFileFuncが設定されている場合はその結果を返し、
// そうでなければエラーを返します。
func (m *MockClient) ReadFile(ctx context.Context, containerName string, path string, maxLines int) (*FileAccessResult, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(ctx, containerName, path, maxLines)
	}
	return nil, fmt.Errorf("ReadFile not implemented in mock")
}

// GetPolicy returns the security policy associated with this mock client.
//
// GetPolicyはこのモッククライアントに関連付けられたセキュリティポリシーを返します。
func (m *MockClient) GetPolicy() *security.Policy {
	return m.policy
}

// GetAllowedCommands returns the whitelisted commands for a container.
// Delegates to the policy if available.
//
// GetAllowedCommandsはコンテナのホワイトリストコマンドを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetAllowedCommands(containerName string) []string {
	if m.policy != nil {
		return m.policy.GetAllowedCommands(containerName)
	}
	return []string{}
}

// GetSecurityPolicy returns the current security policy configuration.
// Delegates to the policy if available.
//
// GetSecurityPolicyは現在のセキュリティポリシー設定を返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetSecurityPolicy() map[string]any {
	if m.policy != nil {
		return m.policy.GetSecurityPolicy()
	}
	return map[string]any{}
}

// GetAllContainersWithCommands returns all containers and their commands.
// Delegates to the policy if available.
//
// GetAllContainersWithCommandsはすべてのコンテナとそのコマンドを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetAllContainersWithCommands() map[string][]string {
	if m.policy != nil {
		return m.policy.GetAllContainersWithCommands()
	}
	return map[string][]string{}
}

// IsDangerousModeEnabled returns whether dangerous mode is enabled.
// Delegates to the policy if available.
//
// IsDangerousModeEnabledは危険モードが有効かどうかを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) IsDangerousModeEnabled() bool {
	if m.policy != nil {
		return m.policy.IsDangerousModeEnabled()
	}
	return false
}

// GetDangerousCommandsForContainer returns dangerous commands for a container.
// Delegates to the policy if available.
//
// GetDangerousCommandsForContainerはコンテナの危険コマンドを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetDangerousCommandsForContainer(containerName string) []string {
	if m.policy != nil {
		return m.policy.GetDangerousCommandsForContainer(containerName)
	}
	return []string{}
}

// GetAllDangerousCommands returns all containers and their dangerous commands.
// Delegates to the policy if available.
//
// GetAllDangerousCommandsはすべてのコンテナとその危険コマンドを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetAllDangerousCommands() map[string][]string {
	if m.policy != nil {
		return m.policy.GetAllDangerousCommands()
	}
	return map[string][]string{}
}

// GetBlockedPaths returns all blocked file paths.
// Delegates to the policy if available.
//
// GetBlockedPathsはすべてのブロックされたファイルパスを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetBlockedPaths() []security.BlockedPath {
	if m.policy != nil {
		return m.policy.GetBlockedPaths()
	}
	return []security.BlockedPath{}
}

// GetBlockedPathsForContainer returns blocked paths for a specific container.
// Delegates to the policy if available.
//
// GetBlockedPathsForContainerは特定のコンテナのブロックパスを返します。
// 利用可能な場合はポリシーに委譲します。
func (m *MockClient) GetBlockedPathsForContainer(containerName string) []security.BlockedPath {
	if m.policy != nil {
		return m.policy.GetBlockedPathsForContainer(containerName)
	}
	return []security.BlockedPath{}
}

// Close is a no-op for the mock client.
// Closeはモッククライアントでは何もしません。
func (m *MockClient) Close() error {
	return nil
}

// Verify that MockClient implements DockerClientInterface at compile time.
// コンパイル時にMockClientがDockerClientInterfaceを実装していることを検証します。
var _ DockerClientInterface = (*MockClient)(nil)
