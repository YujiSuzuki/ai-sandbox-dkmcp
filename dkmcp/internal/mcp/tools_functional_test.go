// Package mcp provides functional tests for MCP tool handlers.
// These tests use a mock Docker client to verify the behavior of tool handlers
// without requiring a real Docker daemon.
//
// mcpパッケージはMCPツールハンドラーの機能テストを提供します。
// これらのテストはモックDockerクライアントを使用して、
// 実際のDockerデーモンを必要とせずにツールハンドラーの動作を検証します。
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	configPkg "github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/docker"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/hosttools"
	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/security"
)

// createTestPolicy creates a security policy for testing.
// createTestPolicyはテスト用のセキュリティポリシーを作成します。
func createTestPolicy() *security.Policy {
	cfg := &configPkg.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-*"},
		Permissions: configPkg.SecurityPermissions{
			Logs:    true,
			Inspect: true,
			Stats:   true,
			Exec:    true,
		},
		ExecWhitelist: map[string][]string{
			"test-api": {"npm test", "npm run lint"},
			"*":        {"echo *"},
		},
	}
	return security.NewPolicy(cfg)
}

// createTestServer creates a test server with a mock Docker client.
// createTestServerはモックDockerクライアントを持つテストサーバーを作成します。
func createTestServer(mockClient *docker.MockClient) *Server {
	return NewServer(mockClient, 8080)
}

// TestToolListContainers_Functional tests the list_containers tool handler.
// TestToolListContainers_Functionalはlist_containersツールハンドラーをテストします。
func TestToolListContainers_Functional(t *testing.T) {
	// Create a mock client with test data
	// テストデータを持つモッククライアントを作成
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ListContainersFunc = func(ctx context.Context) ([]docker.ContainerInfo, error) {
		return []docker.ContainerInfo{
			{
				ID:     "abc123def456",
				Name:   "test-api",
				Image:  "node:18",
				State:  "running",
				Status: "Up 2 hours",
			},
			{
				ID:     "xyz789abc123",
				Name:   "test-db",
				Image:  "postgres:15",
				State:  "running",
				Status: "Up 3 hours",
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolListContainers(ctx, map[string]any{})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolListContainers returned error: %v", err)
	}

	// Verify result contains expected data
	// 結果に期待されるデータが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	// Check that the response contains container data
	// レスポンスにコンテナデータが含まれていることを確認
	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") {
		t.Errorf("expected result to contain 'test-api', got: %s", text)
	}
	if !strings.Contains(text, "test-db") {
		t.Errorf("expected result to contain 'test-db', got: %s", text)
	}
}

// TestToolListContainers_Error tests error handling in list_containers.
// TestToolListContainers_Errorはlist_containersのエラーハンドリングをテストします。
func TestToolListContainers_Error(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ListContainersFunc = func(ctx context.Context) ([]docker.ContainerInfo, error) {
		return nil, errors.New("Docker daemon not available")
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	_, err := server.toolListContainers(ctx, map[string]any{})

	// Verify error is returned
	// エラーが返されることを検証
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Docker daemon not available") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestToolGetLogs_Functional tests the get_logs tool handler.
// TestToolGetLogs_Functionalはget_logsツールハンドラーをテストします。
func TestToolGetLogs_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.GetLogsFunc = func(ctx context.Context, name, tail, since string, follow bool) (string, error) {
		if name != "test-api" {
			return "", errors.New("container not found")
		}
		return "2024-01-01T00:00:00Z Server started\n2024-01-01T00:00:01Z Listening on port 3000\n", nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler with valid parameters
	// 有効なパラメータでツールハンドラーを呼び出す
	result, err := server.toolGetLogs(ctx, map[string]any{
		"container": "test-api",
		"tail":      "100",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolGetLogs returned error: %v", err)
	}

	// Verify result contains log data
	// 結果にログデータが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "Server started") {
		t.Errorf("expected result to contain log data, got: %s", text)
	}
}

// TestToolGetLogs_MissingContainer tests missing container parameter.
// TestToolGetLogs_MissingContainerは欠落しているcontainerパラメータをテストします。
func TestToolGetLogs_MissingContainer(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call without container parameter
	// containerパラメータなしで呼び出す
	_, err := server.toolGetLogs(ctx, map[string]any{})

	// Verify error is returned
	// エラーが返されることを検証
	if err == nil {
		t.Fatal("expected error for missing container, got nil")
	}
	if !strings.Contains(err.Error(), "container") {
		t.Errorf("error should mention 'container': %v", err)
	}
}

// TestToolGetLogs_WithSince tests that the since parameter is passed to GetLogs.
// TestToolGetLogs_WithSinceはsinceパラメータがGetLogsに渡されることをテストします。
func TestToolGetLogs_WithSince(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)

	// Capture the since parameter passed to GetLogs
	// GetLogsに渡されたsinceパラメータをキャプチャ
	var capturedSince string
	mockClient.GetLogsFunc = func(ctx context.Context, name, tail, since string, follow bool) (string, error) {
		capturedSince = since
		return "2024-01-01T12:00:00Z Log entry after since\n", nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call with since parameter
	// sinceパラメータ付きで呼び出す
	result, err := server.toolGetLogs(ctx, map[string]any{
		"container": "test-api",
		"tail":      "100",
		"since":     "2024-01-01T00:00:00Z",
	})

	if err != nil {
		t.Fatalf("toolGetLogs returned error: %v", err)
	}

	// Verify since was passed through
	// sinceが正しく渡されたことを検証
	if capturedSince != "2024-01-01T00:00:00Z" {
		t.Errorf("expected since='2024-01-01T00:00:00Z', got '%s'", capturedSince)
	}

	// Verify result contains log data
	// 結果にログデータが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "Log entry after since") {
		t.Errorf("expected result to contain log data, got: %s", text)
	}
}

// TestToolGetStats_Functional tests the get_stats tool handler.
// TestToolGetStats_Functionalはget_statsツールハンドラーをテストします。
func TestToolGetStats_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.GetStatsFunc = func(ctx context.Context, name string) (*container.StatsResponse, error) {
		if name != "test-api" {
			return nil, errors.New("container not found")
		}
		return &container.StatsResponse{
			Name: "test-api",
			MemoryStats: container.MemoryStats{
				Usage: 104857600, // 100 MiB
				Limit: 536870912, // 512 MiB
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolGetStats(ctx, map[string]any{
		"container": "test-api",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolGetStats returned error: %v", err)
	}

	// Verify result contains stats data
	// 結果に統計データが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") {
		t.Errorf("expected result to contain container name, got: %s", text)
	}
}

// TestToolExecCommand_Functional tests the exec_command tool handler.
// TestToolExecCommand_Functionalはexec_commandツールハンドラーをテストします。
func TestToolExecCommand_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ExecFunc = func(ctx context.Context, name, cmd string, danger bool) (*docker.ExecResult, error) {
		if name != "test-api" {
			return nil, errors.New("container not found")
		}
		if cmd == "npm test" {
			return &docker.ExecResult{
				ExitCode: 0,
				Output:   "All tests passed!\n5 tests, 0 failures\n",
			}, nil
		}
		return nil, errors.New("command not allowed")
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolExecCommand(ctx, map[string]any{
		"container": "test-api",
		"command":   "npm test",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolExecCommand returned error: %v", err)
	}

	// Verify result contains command output
	// 結果にコマンド出力が含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "All tests passed") {
		t.Errorf("expected result to contain test output, got: %s", text)
	}
	if !strings.Contains(text, "Exit Code: 0") {
		t.Errorf("expected result to contain exit code, got: %s", text)
	}
}

// TestToolExecCommand_Blocked tests command rejection by security policy.
// TestToolExecCommand_Blockedはセキュリティポリシーによるコマンド拒否をテストします。
func TestToolExecCommand_Blocked(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ExecFunc = func(ctx context.Context, name, cmd string, danger bool) (*docker.ExecResult, error) {
		return nil, errors.New("exec permission denied: command not whitelisted")
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call with a non-whitelisted command
	// ホワイトリストにないコマンドで呼び出す
	_, err := server.toolExecCommand(ctx, map[string]any{
		"container": "test-api",
		"command":   "rm -rf /",
	})

	// Verify error is returned
	// エラーが返されることを検証
	if err == nil {
		t.Fatal("expected error for blocked command, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "not whitelisted") {
		t.Errorf("error should mention permission denial: %v", err)
	}
}

// TestToolInspectContainer_Functional tests the inspect_container tool handler.
// TestToolInspectContainer_Functionalはinspect_containerツールハンドラーをテストします。
func TestToolInspectContainer_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.InspectContainerFunc = func(ctx context.Context, name string) (*types.ContainerJSON, error) {
		if name != "test-api" {
			return nil, errors.New("container not found")
		}
		return &types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				ID:    "abc123def456789",
				Name:  "/test-api",
				Image: "sha256:abcdef123456",
				State: &types.ContainerState{
					Status:  "running",
					Running: true,
				},
			},
			Config: &container.Config{
				Image: "node:18",
				Cmd:   []string{"npm", "start"},
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolInspectContainer(ctx, map[string]any{
		"container": "test-api",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolInspectContainer returned error: %v", err)
	}

	// Verify result contains inspection data
	// 結果に検査データが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") {
		t.Errorf("expected result to contain container name, got: %s", text)
	}
}

// TestToolSearchLogs_Functional tests the search_logs tool handler.
// TestToolSearchLogs_Functionalはsearch_logsツールハンドラーをテストします。
func TestToolSearchLogs_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.GetLogsFunc = func(ctx context.Context, name, tail, since string, follow bool) (string, error) {
		return "2024-01-01T00:00:00Z Info: Starting server\n" +
			"2024-01-01T00:00:01Z Error: Connection failed\n" +
			"2024-01-01T00:00:02Z Info: Retrying...\n" +
			"2024-01-01T00:00:03Z Error: Connection timeout\n" +
			"2024-01-01T00:00:04Z Info: Server running\n", nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler to search for errors
	// エラーを検索するためにツールハンドラーを呼び出す
	result, err := server.toolSearchLogs(ctx, map[string]any{
		"container":     "test-api",
		"pattern":       "Error",
		"tail":          "1000",
		"context_lines": float64(1),
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolSearchLogs returned error: %v", err)
	}

	// Verify result contains search matches
	// 結果に検索マッチが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)

	// Parse the JSON result
	// JSON結果をパース
	var searchResult map[string]any
	if err := json.Unmarshal([]byte(text), &searchResult); err != nil {
		t.Fatalf("failed to parse search result: %v", err)
	}

	matchesCount, ok := searchResult["matches_count"].(float64)
	if !ok {
		t.Fatal("expected matches_count in result")
	}
	if matchesCount != 2 {
		t.Errorf("expected 2 matches, got %v", matchesCount)
	}
}

// TestToolListFiles_Functional tests the list_files tool handler.
// TestToolListFiles_Functionalはlist_filesツールハンドラーをテストします。
func TestToolListFiles_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ListFilesFunc = func(ctx context.Context, name, path string) (*docker.FileAccessResult, error) {
		if name != "test-api" {
			return nil, errors.New("container not found")
		}
		return &docker.FileAccessResult{
			Success: true,
			Data:    "total 16\ndrwxr-xr-x 2 node node 4096 Jan 1 00:00 .\ndrwxr-xr-x 3 node node 4096 Jan 1 00:00 ..\n-rw-r--r-- 1 node node  123 Jan 1 00:00 package.json\n-rw-r--r-- 1 node node  456 Jan 1 00:00 index.js\n",
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolListFiles(ctx, map[string]any{
		"container": "test-api",
		"path":      "/app",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolListFiles returned error: %v", err)
	}

	// Verify result contains file listing
	// 結果にファイル一覧が含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "package.json") {
		t.Errorf("expected result to contain file listing, got: %s", text)
	}
}

// TestToolListFiles_Blocked tests file access blocked by security policy.
// TestToolListFiles_Blockedはセキュリティポリシーによるファイルアクセスブロックをテストします。
func TestToolListFiles_Blocked(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ListFilesFunc = func(ctx context.Context, name, path string) (*docker.FileAccessResult, error) {
		return &docker.FileAccessResult{
			Success: false,
			Blocked: true,
			Block: &security.BlockedPath{
				Pattern: "/etc/secrets/*",
				Reason:  "manual_block",
				Source:  "dkmcp.yaml",
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call with a blocked path
	// ブロックされたパスで呼び出す
	result, err := server.toolListFiles(ctx, map[string]any{
		"container": "test-api",
		"path":      "/etc/secrets",
	})

	// Verify no error (blocked is returned as result, not error)
	// エラーがないことを検証（ブロックはエラーではなく結果として返される）
	if err != nil {
		t.Fatalf("toolListFiles returned error: %v", err)
	}

	// Verify result indicates blocked
	// 結果がブロックを示していることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "blocked") {
		t.Errorf("expected result to indicate blocked access, got: %s", text)
	}
}

// TestToolReadFile_Functional tests the read_file tool handler.
// TestToolReadFile_Functionalはread_fileツールハンドラーをテストします。
func TestToolReadFile_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.ReadFileFunc = func(ctx context.Context, name, path string, maxLines int) (*docker.FileAccessResult, error) {
		if name != "test-api" {
			return nil, errors.New("container not found")
		}
		return &docker.FileAccessResult{
			Success: true,
			Data:    "{\n  \"name\": \"test-api\",\n  \"version\": \"1.0.0\"\n}\n",
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolReadFile(ctx, map[string]any{
		"container": "test-api",
		"path":      "/app/package.json",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolReadFile returned error: %v", err)
	}

	// Verify result contains file content
	// 結果にファイル内容が含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") {
		t.Errorf("expected result to contain file content, got: %s", text)
	}
}

// TestToolGetAllowedCommands_Functional tests the get_allowed_commands tool handler.
// TestToolGetAllowedCommands_Functionalはget_allowed_commandsツールハンドラーをテストします。
func TestToolGetAllowedCommands_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call for a specific container
	// 特定のコンテナに対して呼び出す
	result, err := server.toolGetAllowedCommands(ctx, map[string]any{
		"container": "test-api",
	})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolGetAllowedCommands returned error: %v", err)
	}

	// Verify result contains command list
	// 結果にコマンドリストが含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "npm test") {
		t.Errorf("expected result to contain allowed commands, got: %s", text)
	}
}

// TestToolGetSecurityPolicy_Functional tests the get_security_policy tool handler.
// TestToolGetSecurityPolicy_Functionalはget_security_policyツールハンドラーをテストします。
func TestToolGetSecurityPolicy_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolGetSecurityPolicy(ctx, map[string]any{})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolGetSecurityPolicy returned error: %v", err)
	}

	// Verify result contains policy information
	// 結果にポリシー情報が含まれていることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in result")
	}

	text := content[0]["text"].(string)
	if !strings.Contains(text, "moderate") {
		t.Errorf("expected result to contain security mode, got: %s", text)
	}
}

// TestToolGetBlockedPaths_Functional tests the get_blocked_paths tool handler.
// TestToolGetBlockedPaths_Functionalはget_blocked_pathsツールハンドラーをテストします。
func TestToolGetBlockedPaths_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	server := createTestServer(mockClient)
	ctx := context.Background()

	// Call the tool handler
	// ツールハンドラーを呼び出す
	result, err := server.toolGetBlockedPaths(ctx, map[string]any{})

	// Verify no error
	// エラーがないことを検証
	if err != nil {
		t.Fatalf("toolGetBlockedPaths returned error: %v", err)
	}

	// Verify result is a map
	// 結果がマップであることを検証
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	// Should have content
	// contentを持つべき
	if _, ok := resultMap["content"]; !ok {
		t.Error("expected content in result")
	}
}

// createTestPolicyWithHostPathMasking creates a security policy with host path masking enabled.
// createTestPolicyWithHostPathMaskingはホストパスマスキングが有効なセキュリティポリシーを作成します。
func createTestPolicyWithHostPathMasking() *security.Policy {
	cfg := &configPkg.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-*"},
		Permissions: configPkg.SecurityPermissions{
			Logs:    true,
			Inspect: true,
			Stats:   true,
			Exec:    true,
		},
		ExecWhitelist: map[string][]string{
			"test-api": {"npm test", "npm run lint"},
			"*":        {"echo *"},
		},
		HostPathMasking: configPkg.HostPathMaskingConfig{
			Enabled:     true,
			Replacement: "[HOST_PATH]",
		},
	}
	return security.NewPolicy(cfg)
}

// TestToolListContainers_HostPathMasking tests that host paths are masked in list_containers output.
// TestToolListContainers_HostPathMaskingはlist_containers出力でホストパスがマスクされることをテストします。
func TestToolListContainers_HostPathMasking(t *testing.T) {
	// Create a policy with host path masking enabled
	// ホストパスマスキングが有効なポリシーを作成
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	// Mock containers with labels containing host paths
	// ホストパスを含むラベルを持つコンテナをモック
	mockClient.ListContainersFunc = func(ctx context.Context) ([]docker.ContainerInfo, error) {
		return []docker.ContainerInfo{
			{
				ID:     "abc123",
				Name:   "test-api",
				Image:  "node:18",
				State:  "running",
				Status: "Up 2 hours",
				Labels: map[string]string{
					"com.docker.compose.project.config_files": "/Users/john/workspace/demo-apps/docker-compose.yml",
				},
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolListContainers(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("toolListContainers returned error: %v", err)
	}

	// Extract the response text
	// レスポンステキストを抽出
	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host path is masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/Users/john") {
		t.Errorf("expected host path to be masked, but found '/Users/john' in: %s", text)
	}

	// Verify masking replacement is present
	// マスキング置換が存在することを検証
	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// TestToolGetLogs_HostPathMasking tests that host paths are masked in log output.
// TestToolGetLogs_HostPathMaskingはログ出力でホストパスがマスクされることをテストします。
func TestToolGetLogs_HostPathMasking(t *testing.T) {
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	// Mock logs containing host paths
	// ホストパスを含むログをモック
	mockClient.GetLogsFunc = func(ctx context.Context, name, tail, since string, follow bool) (string, error) {
		return "Loading config from /Users/suzu/my_work/dev/app/config.json\nServer started", nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolGetLogs(ctx, map[string]any{
		"container": "test-api",
	})
	if err != nil {
		t.Fatalf("toolGetLogs returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host path is masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/Users/suzu") {
		t.Errorf("expected host path to be masked, but found '/Users/suzu' in: %s", text)
	}

	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// TestToolExecCommand_HostPathMasking tests that host paths are masked in exec command output.
// TestToolExecCommand_HostPathMaskingはexecコマンド出力でホストパスがマスクされることをテストします。
func TestToolExecCommand_HostPathMasking(t *testing.T) {
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	// Mock exec returning host paths
	// ホストパスを返すexecをモック
	mockClient.ExecFunc = func(ctx context.Context, name, command string, dangerously bool) (*docker.ExecResult, error) {
		return &docker.ExecResult{
			Output:   "File path: /home/developer/projects/app/src/main.go",
			ExitCode: 0,
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolExecCommand(ctx, map[string]any{
		"container": "test-api",
		"command":   "echo test",
	})
	if err != nil {
		t.Fatalf("toolExecCommand returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host path is masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/home/developer") {
		t.Errorf("expected host path to be masked, but found '/home/developer' in: %s", text)
	}

	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// TestToolInspectContainer_HostPathMasking tests that host paths are masked in inspect output.
// TestToolInspectContainer_HostPathMaskingはinspect出力でホストパスがマスクされることをテストします。
func TestToolInspectContainer_HostPathMasking(t *testing.T) {
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	mockClient.InspectContainerFunc = func(ctx context.Context, name string) (*types.ContainerJSON, error) {
		return &types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				ID:   "abc123",
				Name: "/test-api",
				HostConfig: &container.HostConfig{
					Binds: []string{"/Users/jane/workspace/app:/app"},
				},
			},
			Config: &container.Config{
				Image: "node:18",
				Env:   []string{"APP_PATH=/home/user/app"},
			},
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolInspectContainer(ctx, map[string]any{
		"container": "test-api",
	})
	if err != nil {
		t.Fatalf("toolInspectContainer returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host paths are masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/Users/jane") {
		t.Errorf("expected host path to be masked, but found '/Users/jane' in: %s", text)
	}

	if strings.Contains(text, "/home/user") {
		t.Errorf("expected host path to be masked, but found '/home/user' in: %s", text)
	}

	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// TestToolSearchLogs_HostPathMasking tests that host paths are masked in search_logs output.
// TestToolSearchLogs_HostPathMaskingはsearch_logs出力でホストパスがマスクされることをテストします。
func TestToolSearchLogs_HostPathMasking(t *testing.T) {
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	mockClient.GetLogsFunc = func(ctx context.Context, name, tail, since string, follow bool) (string, error) {
		return "Error loading /Users/admin/config.json\nFile not found", nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolSearchLogs(ctx, map[string]any{
		"container": "test-api",
		"pattern":   "Error",
	})
	if err != nil {
		t.Fatalf("toolSearchLogs returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host path is masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/Users/admin") {
		t.Errorf("expected host path to be masked, but found '/Users/admin' in: %s", text)
	}

	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// TestToolReadFile_HostPathMasking tests that host paths are masked in read_file output.
// TestToolReadFile_HostPathMaskingはread_file出力でホストパスがマスクされることをテストします。
func TestToolReadFile_HostPathMasking(t *testing.T) {
	policy := createTestPolicyWithHostPathMasking()
	mockClient := docker.NewMockClient(policy)

	mockClient.ReadFileFunc = func(ctx context.Context, containerName, path string, maxLines int) (*docker.FileAccessResult, error) {
		return &docker.FileAccessResult{
			Success: true,
			Data:    "source_path: /home/jenkins/workspace/project/src",
		}, nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolReadFile(ctx, map[string]any{
		"container": "test-api",
		"path":      "/app/config.yaml",
	})
	if err != nil {
		t.Fatalf("toolReadFile returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)

	// Verify host path is masked
	// ホストパスがマスクされていることを検証
	if strings.Contains(text, "/home/jenkins") {
		t.Errorf("expected host path to be masked, but found '/home/jenkins' in: %s", text)
	}

	if !strings.Contains(text, "[HOST_PATH]") {
		t.Errorf("expected '[HOST_PATH]' in result, got: %s", text)
	}
}

// createTestPolicyWithLifecycle creates a security policy with lifecycle permission enabled.
// createTestPolicyWithLifecycleはlifecycleパーミッションが有効なセキュリティポリシーを作成します。
func createTestPolicyWithLifecycle() *security.Policy {
	cfg := &configPkg.SecurityConfig{
		Mode:              "moderate",
		AllowedContainers: []string{"test-*", "demo-*"},
		Permissions: configPkg.SecurityPermissions{
			Logs:      true,
			Inspect:   true,
			Stats:     true,
			Exec:      true,
			Lifecycle: true,
		},
		ExecWhitelist: map[string][]string{
			"test-api": {"npm test", "npm run lint"},
			"*":        {"echo *"},
		},
	}
	return security.NewPolicy(cfg)
}

// TestToolRestartContainer_Functional tests the restart_container tool handler.
// TestToolRestartContainer_Functionalはrestart_containerツールハンドラーをテストします。
func TestToolRestartContainer_Functional(t *testing.T) {
	policy := createTestPolicyWithLifecycle()
	mockClient := docker.NewMockClient(policy)
	var calledContainer string
	var calledTimeout *int
	mockClient.RestartContainerFunc = func(ctx context.Context, containerName string, timeout *int) error {
		calledContainer = containerName
		calledTimeout = timeout
		return nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	// Test without timeout
	result, err := server.toolRestartContainer(ctx, map[string]any{
		"container": "test-api",
	})
	if err != nil {
		t.Fatalf("toolRestartContainer returned error: %v", err)
	}
	if calledContainer != "test-api" {
		t.Errorf("expected container 'test-api', got '%s'", calledContainer)
	}
	if calledTimeout != nil {
		t.Errorf("expected nil timeout, got %v", *calledTimeout)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") || !strings.Contains(text, "restarted") {
		t.Errorf("expected success message with container name, got: %s", text)
	}

	// Test with timeout
	_, err = server.toolRestartContainer(ctx, map[string]any{
		"container": "test-api",
		"timeout":   float64(30),
	})
	if err != nil {
		t.Fatalf("toolRestartContainer with timeout returned error: %v", err)
	}
	if calledTimeout == nil || *calledTimeout != 30 {
		t.Errorf("expected timeout 30, got %v", calledTimeout)
	}
}

// TestToolStopContainer_Functional tests the stop_container tool handler.
// TestToolStopContainer_Functionalはstop_containerツールハンドラーをテストします。
func TestToolStopContainer_Functional(t *testing.T) {
	policy := createTestPolicyWithLifecycle()
	mockClient := docker.NewMockClient(policy)
	var calledContainer string
	mockClient.StopContainerFunc = func(ctx context.Context, containerName string, timeout *int) error {
		calledContainer = containerName
		return nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolStopContainer(ctx, map[string]any{
		"container": "test-api",
	})
	if err != nil {
		t.Fatalf("toolStopContainer returned error: %v", err)
	}
	if calledContainer != "test-api" {
		t.Errorf("expected container 'test-api', got '%s'", calledContainer)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") || !strings.Contains(text, "stopped") {
		t.Errorf("expected success message with container name, got: %s", text)
	}
}

// TestToolStartContainer_Functional tests the start_container tool handler.
// TestToolStartContainer_Functionalはstart_containerツールハンドラーをテストします。
func TestToolStartContainer_Functional(t *testing.T) {
	policy := createTestPolicyWithLifecycle()
	mockClient := docker.NewMockClient(policy)
	var calledContainer string
	mockClient.StartContainerFunc = func(ctx context.Context, containerName string) error {
		calledContainer = containerName
		return nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	result, err := server.toolStartContainer(ctx, map[string]any{
		"container": "test-api",
	})
	if err != nil {
		t.Fatalf("toolStartContainer returned error: %v", err)
	}
	if calledContainer != "test-api" {
		t.Errorf("expected container 'test-api', got '%s'", calledContainer)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "test-api") || !strings.Contains(text, "started") {
		t.Errorf("expected success message with container name, got: %s", text)
	}
}

// TestToolLifecycle_PermissionDenied tests lifecycle tools when permission is disabled.
// TestToolLifecycle_PermissionDeniedはパーミッション無効時のlifecycleツールをテストします。
func TestToolLifecycle_PermissionDenied(t *testing.T) {
	// Use createTestPolicy() which does NOT have Lifecycle enabled
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	mockClient.RestartContainerFunc = func(ctx context.Context, containerName string, timeout *int) error {
		t.Error("RestartContainerFunc should not be called when permission denied")
		return nil
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	_, err := server.toolRestartContainer(ctx, map[string]any{"container": "test-api"})
	if err == nil {
		t.Error("expected error for restart_container with lifecycle disabled")
	}

	_, err = server.toolStopContainer(ctx, map[string]any{"container": "test-api"})
	if err == nil {
		t.Error("expected error for stop_container with lifecycle disabled")
	}

	_, err = server.toolStartContainer(ctx, map[string]any{"container": "test-api"})
	if err == nil {
		t.Error("expected error for start_container with lifecycle disabled")
	}
}

// TestToolLifecycle_DockerError tests lifecycle tools when Docker returns an error.
// TestToolLifecycle_DockerErrorはDockerがエラーを返した場合のlifecycleツールをテストします。
func TestToolLifecycle_DockerError(t *testing.T) {
	policy := createTestPolicyWithLifecycle()
	mockClient := docker.NewMockClient(policy)
	mockClient.RestartContainerFunc = func(ctx context.Context, containerName string, timeout *int) error {
		return errors.New("container not found")
	}

	server := createTestServer(mockClient)
	ctx := context.Background()

	_, err := server.toolRestartContainer(ctx, map[string]any{"container": "test-api"})
	if err == nil {
		t.Error("expected error when Docker returns error")
	}
	if !strings.Contains(err.Error(), "container not found") {
		t.Errorf("expected 'container not found' in error, got: %v", err)
	}
}

// --- Host Tool Handler Tests ---

// TestToolListHostTools_Functional tests the list_host_tools tool handler.
// TestToolListHostTools_Functionalはlist_host_tools MCPツールハンドラーをテストします。
func TestToolListHostTools_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	server := createTestServer(mockClient)
	ctx := context.Background()

	// Test: nil manager returns error
	// nilマネージャーはエラーを返す
	_, err := server.toolListHostTools(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error when hostToolsManager is nil")
	}
	if err != nil && !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' error, got: %v", err)
	}

	// Test: with enabled manager
	// 有効なマネージャーでのテスト
	dir := t.TempDir()
	toolsDir := dir + "/tools"
	os.MkdirAll(toolsDir, 0755)
	os.WriteFile(toolsDir+"/hello.sh", []byte("#!/bin/bash\n# hello.sh\n# Hello tool\n"), 0755)

	htCfg := &configPkg.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	mgr := hosttools.NewManager(htCfg, dir)
	server = NewServer(mockClient, 8080, WithHostToolsManager(mgr))

	result, err := server.toolListHostTools(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("toolListHostTools returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "hello.sh") {
		t.Errorf("expected tool list to contain 'hello.sh', got: %s", text)
	}
}

// TestToolGetHostToolInfo_Functional tests the get_host_tool_info tool handler.
// TestToolGetHostToolInfo_Functionalはget_host_tool_info MCPツールハンドラーをテストします。
func TestToolGetHostToolInfo_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	ctx := context.Background()

	dir := t.TempDir()
	toolsDir := dir + "/tools"
	os.MkdirAll(toolsDir, 0755)
	os.WriteFile(toolsDir+"/info-test.sh", []byte("#!/bin/bash\n# info-test.sh\n# Info test tool\n"), 0755)

	htCfg := &configPkg.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	mgr := hosttools.NewManager(htCfg, dir)
	server := NewServer(mockClient, 8080, WithHostToolsManager(mgr))

	// Test: missing name parameter
	// nameパラメータが欠落している場合
	_, err := server.toolGetHostToolInfo(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error for missing name parameter")
	}

	// Test: valid name
	// 有効な名前
	result, err := server.toolGetHostToolInfo(ctx, map[string]any{"name": "info-test.sh"})
	if err != nil {
		t.Fatalf("toolGetHostToolInfo returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "info-test.sh") {
		t.Errorf("expected info to contain 'info-test.sh', got: %s", text)
	}

	// Test: tool not found
	// ツールが見つからない場合
	_, err = server.toolGetHostToolInfo(ctx, map[string]any{"name": "nonexistent.sh"})
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// TestToolRunHostTool_Functional tests the run_host_tool tool handler.
// TestToolRunHostTool_Functionalはrun_host_tool MCPツールハンドラーをテストします。
func TestToolRunHostTool_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	ctx := context.Background()

	dir := t.TempDir()
	toolsDir := dir + "/tools"
	os.MkdirAll(toolsDir, 0755)
	os.WriteFile(toolsDir+"/greet.sh", []byte("#!/bin/bash\n# greet.sh\n# Greet tool\necho \"Hello $1\"\n"), 0755)

	htCfg := &configPkg.HostToolsConfig{
		Enabled:           true,
		Directories:       []string{"tools"},
		AllowedExtensions: []string{".sh"},
		Timeout:           30,
	}
	mgr := hosttools.NewManager(htCfg, dir)
	server := NewServer(mockClient, 8080, WithHostToolsManager(mgr))

	// Test: nil manager
	// nilマネージャー
	serverNil := createTestServer(mockClient)
	_, err := serverNil.toolRunHostTool(ctx, map[string]any{"name": "greet.sh"})
	if err == nil {
		t.Error("expected error when hostToolsManager is nil")
	}

	// Test: missing name
	// nameが欠落
	_, err = server.toolRunHostTool(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error for missing name parameter")
	}

	// Test: happy path
	// 正常系
	result, err := server.toolRunHostTool(ctx, map[string]any{
		"name": "greet.sh",
		"args": []any{"World"},
	})
	if err != nil {
		t.Fatalf("toolRunHostTool returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "Hello World") {
		t.Errorf("expected output to contain 'Hello World', got: %s", text)
	}
	if !strings.Contains(text, "Exit Code: 0") {
		t.Errorf("expected exit code 0 in output, got: %s", text)
	}

	// Test: tool not found
	// ツールが見つからない
	_, err = server.toolRunHostTool(ctx, map[string]any{"name": "nonexistent.sh"})
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// TestToolExecHostCommand_Functional tests the exec_host_command tool handler.
// TestToolExecHostCommand_Functionalはexec_host_command MCPツールハンドラーをテストします。
func TestToolExecHostCommand_Functional(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	ctx := context.Background()

	// Test: nil policy returns error
	// nilポリシーはエラーを返す
	serverNil := createTestServer(mockClient)
	_, err := serverNil.toolExecHostCommand(ctx, map[string]any{"command": "echo hello"})
	if err == nil {
		t.Error("expected error when hostCommandPolicy is nil")
	}

	// Test: missing command parameter
	// commandパラメータが欠落
	hcCfg := &configPkg.HostCommandsConfig{
		Enabled: true,
		Whitelist: map[string][]string{
			"echo": {"*"},
		},
		Deny:        map[string][]string{},
		Dangerously: configPkg.HostCommandsDangerously{Commands: map[string][]string{}},
	}
	hcPolicy := security.NewHostCommandPolicy(hcCfg)
	server := NewServer(mockClient, 8080,
		WithHostCommandPolicy(hcPolicy, t.TempDir(), 30*time.Second),
	)

	_, err = server.toolExecHostCommand(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error for missing command parameter")
	}

	// Test: whitelisted command executes successfully
	// ホワイトリストに登録されたコマンドが正常に実行される
	result, err := server.toolExecHostCommand(ctx, map[string]any{
		"command": "echo hello from host",
	})
	if err != nil {
		t.Fatalf("toolExecHostCommand returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, "hello from host") {
		t.Errorf("expected output to contain 'hello from host', got: %s", text)
	}
	if !strings.Contains(text, "Exit Code: 0") {
		t.Errorf("expected exit code 0, got: %s", text)
	}

	// Test: non-whitelisted command is rejected
	// ホワイトリスト外のコマンドが拒否される
	_, err = server.toolExecHostCommand(ctx, map[string]any{
		"command": "rm -rf /",
	})
	if err == nil {
		t.Error("expected error for non-whitelisted command")
	}

	// Test: pipe in command is rejected
	// コマンド内のパイプが拒否される
	_, err = server.toolExecHostCommand(ctx, map[string]any{
		"command": "echo hello | cat",
	})
	if err == nil {
		t.Error("expected error for pipe in command")
	}
}

// TestToolExecHostCommand_DangerousMode tests the dangerously flag behavior.
// TestToolExecHostCommand_DangerousModeはdangerouslyフラグの動作をテストします。
func TestToolExecHostCommand_DangerousMode(t *testing.T) {
	policy := createTestPolicy()
	mockClient := docker.NewMockClient(policy)
	ctx := context.Background()

	hcCfg := &configPkg.HostCommandsConfig{
		Enabled: true,
		Whitelist: map[string][]string{
			"echo": {"*"},
		},
		Deny: map[string][]string{},
		Dangerously: configPkg.HostCommandsDangerously{
			Enabled: true,
			Commands: map[string][]string{
				"echo": {"danger *"},
			},
		},
	}
	hcPolicy := security.NewHostCommandPolicy(hcCfg)
	server := NewServer(mockClient, 8080,
		WithHostCommandPolicy(hcPolicy, t.TempDir(), 30*time.Second),
	)

	// Test: dangerously=true with allowed dangerous command
	// dangerously=trueで許可された危険コマンド
	result, err := server.toolExecHostCommand(ctx, map[string]any{
		"command":     "echo danger test",
		"dangerously": true,
	})
	if err != nil {
		t.Fatalf("toolExecHostCommand (dangerous) returned error: %v", err)
	}

	resultMap := result.(map[string]any)
	content := resultMap["content"].([]map[string]any)
	text := content[0]["text"].(string)
	// Should contain DANGEROUS MODE warning
	// DANGEROUS MODEの警告を含むべき
	if !strings.Contains(text, "DANGEROUS MODE") {
		t.Errorf("expected dangerous mode warning, got: %s", text)
	}

	// Test: dangerously=false for dangerous-only command should fail
	// dangerously=falseで危険専用コマンドは失敗すべき
	hcCfg2 := &configPkg.HostCommandsConfig{
		Enabled:   true,
		Whitelist: map[string][]string{},
		Deny:      map[string][]string{},
		Dangerously: configPkg.HostCommandsDangerously{
			Enabled: true,
			Commands: map[string][]string{
				"echo": {"danger *"},
			},
		},
	}
	hcPolicy2 := security.NewHostCommandPolicy(hcCfg2)
	server2 := NewServer(mockClient, 8080,
		WithHostCommandPolicy(hcPolicy2, t.TempDir(), 30*time.Second),
	)

	_, err = server2.toolExecHostCommand(ctx, map[string]any{
		"command":     "echo danger test",
		"dangerously": false,
	})
	if err == nil {
		t.Error("expected error when dangerously=false for dangerous-only command")
	}
}
