// Package mcp provides tests for the MCP tool definitions and implementations.
// This file contains unit tests that verify the structure and schema of all
// available tools, ensuring they meet the MCP specification requirements.
//
// mcpパッケージはMCPツールの定義と実装のテストを提供します。
// このファイルには、利用可能なすべてのツールの構造とスキーマが
// MCP仕様の要件を満たしていることを検証するユニットテストが含まれています。
package mcp

import (
	"encoding/json"
	"testing"
)

// TestServerVersion verifies that the ServerVersion variable can be set and
// is used correctly in the initialize response.
//
// TestServerVersionはServerVersion変数が設定可能で、
// initializeレスポンスで正しく使用されることを検証します。
func TestServerVersion(t *testing.T) {
	// Test default value
	// デフォルト値をテスト
	if ServerVersion == "" {
		t.Error("ServerVersion should have a default value")
	}

	// Test that ServerVersion can be changed
	// ServerVersionが変更可能であることをテスト
	originalVersion := ServerVersion
	defer func() { ServerVersion = originalVersion }() // Restore after test / テスト後に復元

	testVersion := "1.2.3-test"
	ServerVersion = testVersion
	if ServerVersion != testVersion {
		t.Errorf("ServerVersion = %q, want %q", ServerVersion, testVersion)
	}
}

// TestServerVersionInInitialize verifies that the ServerVersion is correctly
// included in the initialize response from the MCP server.
//
// TestServerVersionInInitializeはServerVersionがMCPサーバーの
// initializeレスポンスに正しく含まれることを検証します。
func TestServerVersionInInitialize(t *testing.T) {
	// Save and restore original version
	// 元のバージョンを保存して復元
	originalVersion := ServerVersion
	defer func() { ServerVersion = originalVersion }()

	// Set a test version
	// テストバージョンを設定
	testVersion := "2.0.0-test"
	ServerVersion = testVersion

	// Create a server and call initialize
	// サーバーを作成してinitializeを呼び出す
	server := NewServer(nil, 8080)
	response, _, _, err := server.initialize(nil)
	if err != nil {
		t.Fatalf("initialize() error = %v", err)
	}

	// Check that the response contains the correct version
	// レスポンスに正しいバージョンが含まれていることを確認
	respMap, ok := response.(map[string]any)
	if !ok {
		t.Fatal("response is not a map")
	}

	serverInfo, ok := respMap["serverInfo"].(map[string]string)
	if !ok {
		t.Fatal("serverInfo is not a map[string]string")
	}

	if serverInfo["version"] != testVersion {
		t.Errorf("serverInfo.version = %q, want %q", serverInfo["version"], testVersion)
	}

	if serverInfo["name"] != "dkmcp" {
		t.Errorf("serverInfo.name = %q, want %q", serverInfo["name"], "dkmcp")
	}
}

// TestGetTools verifies that GetTools returns the expected number of tools
// and that all expected tool names are present in the returned list.
//
// TestGetToolsは、GetToolsが期待される数のツールを返し、
// 期待されるすべてのツール名が返されたリストに存在することを検証します。
func TestGetTools(t *testing.T) {
	tools := GetTools()

	// Verify the total number of tools
	// ツールの総数を検証
	expectedToolCount := 14
	if len(tools) != expectedToolCount {
		t.Errorf("GetTools() returned %d tools, want %d", len(tools), expectedToolCount)
	}

	// Check that all expected tools are present by name
	// 期待されるすべてのツールが名前で存在することを確認
	expectedNames := map[string]bool{
		"list_containers":      false,
		"get_logs":             false,
		"get_stats":            false,
		"exec_command":         false,
		"inspect_container":    false,
		"get_allowed_commands": false,
		"get_security_policy":  false,
		"search_logs":          false,
		"list_files":           false,
		"read_file":            false,
		"get_blocked_paths":    false,
		"restart_container":    false,
		"stop_container":       false,
		"start_container":      false,
	}

	// Mark each found tool as present
	// 見つかった各ツールを存在としてマーク
	for _, tool := range tools {
		if _, exists := expectedNames[tool.Name]; exists {
			expectedNames[tool.Name] = true
		}
	}

	// Report any missing tools
	// 見つからないツールを報告
	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

// TestListContainersTool_Structure verifies that the list_containers tool
// has the correct structure including description and input schema properties.
//
// TestListContainersTool_Structureは、list_containersツールが
// 説明と入力スキーマプロパティを含む正しい構造を持っていることを検証します。
func TestListContainersTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the list_containers tool
	// list_containersツールを見つける
	var listTool *Tool
	for i := range tools {
		if tools[i].Name == "list_containers" {
			listTool = &tools[i]
			break
		}
	}

	if listTool == nil {
		t.Fatal("list_containers tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if listTool.Description == "" {
		t.Error("list_containers should have a description")
	}

	// Check input schema type is "object" as required by MCP
	// MCPで要求される通り、入力スキーマの型が"object"であることを確認
	if listTool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want object", listTool.InputSchema.Type)
	}

	// list_containers should have "all" parameter for filtering
	// list_containersはフィルタリング用の"all"パラメータを持つべき
	if _, exists := listTool.InputSchema.Properties["all"]; !exists {
		t.Error("list_containers should have 'all' parameter")
	}
}

// TestGetLogsTool_Structure verifies that the get_logs tool has the correct
// structure with required and optional parameters properly defined.
//
// TestGetLogsTool_Structureは、get_logsツールが必須パラメータと
// オプションパラメータが適切に定義された正しい構造を持っていることを検証します。
func TestGetLogsTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the get_logs tool
	// get_logsツールを見つける
	var logsTool *Tool
	for i := range tools {
		if tools[i].Name == "get_logs" {
			logsTool = &tools[i]
			break
		}
	}

	if logsTool == nil {
		t.Fatal("get_logs tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if logsTool.Description == "" {
		t.Error("get_logs should have a description")
	}

	// Check that "container" is listed as a required parameter
	// "container"が必須パラメータとしてリストされていることを確認
	requiredParams := []string{"container"}
	for _, param := range requiredParams {
		found := false
		for _, req := range logsTool.InputSchema.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("get_logs should require parameter %q", param)
		}
	}

	// Check that optional parameters exist in the schema
	// オプションパラメータがスキーマに存在することを確認
	optionalParams := []string{"tail", "since"}
	for _, param := range optionalParams {
		if _, exists := logsTool.InputSchema.Properties[param]; !exists {
			t.Errorf("get_logs should have optional parameter %q", param)
		}
	}
}

// TestGetStatsTool_Structure verifies that the get_stats tool has the correct
// structure with the required "container" parameter.
//
// TestGetStatsTool_Structureは、get_statsツールが必須の"container"
// パラメータを持つ正しい構造を持っていることを検証します。
func TestGetStatsTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the get_stats tool
	// get_statsツールを見つける
	var statsTool *Tool
	for i := range tools {
		if tools[i].Name == "get_stats" {
			statsTool = &tools[i]
			break
		}
	}

	if statsTool == nil {
		t.Fatal("get_stats tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if statsTool.Description == "" {
		t.Error("get_stats should have a description")
	}

	// Check that "container" is a required parameter
	// "container"が必須パラメータであることを確認
	found := false
	for _, req := range statsTool.InputSchema.Required {
		if req == "container" {
			found = true
			break
		}
	}
	if !found {
		t.Error("get_stats should require 'container' parameter")
	}
}

// TestExecCommandTool_Structure verifies that the exec_command tool has
// both "container" and "command" as required parameters.
//
// TestExecCommandTool_Structureは、exec_commandツールが
// "container"と"command"の両方を必須パラメータとして持っていることを検証します。
func TestExecCommandTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the exec_command tool
	// exec_commandツールを見つける
	var execTool *Tool
	for i := range tools {
		if tools[i].Name == "exec_command" {
			execTool = &tools[i]
			break
		}
	}

	if execTool == nil {
		t.Fatal("exec_command tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if execTool.Description == "" {
		t.Error("exec_command should have a description")
	}

	// Check that both container and command are required parameters
	// containerとcommandの両方が必須パラメータであることを確認
	requiredParams := []string{"container", "command"}
	for _, param := range requiredParams {
		found := false
		for _, req := range execTool.InputSchema.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("exec_command should require parameter %q", param)
		}
	}
}

// TestInspectContainerTool_Structure verifies that the inspect_container tool
// has the correct structure with "container" as a required parameter.
//
// TestInspectContainerTool_Structureは、inspect_containerツールが
// "container"を必須パラメータとして持つ正しい構造を持っていることを検証します。
func TestInspectContainerTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the inspect_container tool
	// inspect_containerツールを見つける
	var inspectTool *Tool
	for i := range tools {
		if tools[i].Name == "inspect_container" {
			inspectTool = &tools[i]
			break
		}
	}

	if inspectTool == nil {
		t.Fatal("inspect_container tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if inspectTool.Description == "" {
		t.Error("inspect_container should have a description")
	}

	// Check that "container" is a required parameter
	// "container"が必須パラメータであることを確認
	found := false
	for _, req := range inspectTool.InputSchema.Required {
		if req == "container" {
			found = true
			break
		}
	}
	if !found {
		t.Error("inspect_container should require 'container' parameter")
	}
}

// TestToolInputSchemas_AllHaveType verifies that all tools have their
// InputSchema.Type set to "object" as required by the MCP specification.
//
// TestToolInputSchemas_AllHaveTypeは、MCP仕様で要求される通り、
// すべてのツールのInputSchema.Typeが"object"に設定されていることを検証します。
func TestToolInputSchemas_AllHaveType(t *testing.T) {
	tools := GetTools()

	for _, tool := range tools {
		// Check that Type is not empty
		// Typeが空でないことを確認
		if tool.InputSchema.Type == "" {
			t.Errorf("tool %q has empty InputSchema.Type", tool.Name)
		}

		// Check that Type is "object" (required by MCP)
		// Typeが"object"であることを確認（MCPで要求）
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %q has InputSchema.Type = %q, want object",
				tool.Name, tool.InputSchema.Type)
		}
	}
}

// TestToolInputSchemas_RequiredParamsExist verifies that all parameters
// listed as "required" actually exist in the Properties map.
//
// TestToolInputSchemas_RequiredParamsExistは、"required"としてリストされた
// すべてのパラメータがPropertiesマップに実際に存在することを検証します。
func TestToolInputSchemas_RequiredParamsExist(t *testing.T) {
	tools := GetTools()

	for _, tool := range tools {
		// For each required parameter, verify it exists in Properties
		// 各必須パラメータについて、Propertiesに存在することを検証
		for _, requiredParam := range tool.InputSchema.Required {
			if _, exists := tool.InputSchema.Properties[requiredParam]; !exists {
				t.Errorf("tool %q requires parameter %q but it's not in Properties",
					tool.Name, requiredParam)
			}
		}
	}
}

// TestToolDescriptions_NotEmpty verifies that all tools and their parameters
// have non-empty descriptions for documentation purposes.
//
// TestToolDescriptions_NotEmptyは、ドキュメント目的で、
// すべてのツールとそのパラメータに空でない説明があることを検証します。
func TestToolDescriptions_NotEmpty(t *testing.T) {
	tools := GetTools()

	for _, tool := range tools {
		// Check tool description
		// ツールの説明を確認
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}

		// Check that each parameter has a description
		// 各パラメータに説明があることを確認
		for paramName, param := range tool.InputSchema.Properties {
			if param.Description == "" {
				t.Errorf("tool %q parameter %q has empty description",
					tool.Name, paramName)
			}
		}
	}
}

// TestToolNames_Unique verifies that all tool names are unique.
// Duplicate names would cause routing issues in the tool dispatcher.
//
// TestToolNames_Uniqueは、すべてのツール名が一意であることを検証します。
// 重複した名前はツールディスパッチャでルーティングの問題を引き起こします。
func TestToolNames_Unique(t *testing.T) {
	tools := GetTools()
	names := make(map[string]bool)

	for _, tool := range tools {
		// Check for duplicates
		// 重複をチェック
		if names[tool.Name] {
			t.Errorf("duplicate tool name: %q", tool.Name)
		}
		names[tool.Name] = true
	}
}

// TestParameterTypes_Valid verifies that all parameter types are valid
// JSON Schema types (string, integer, boolean, number, array, object).
//
// TestParameterTypes_Validは、すべてのパラメータの型が有効な
// JSON Schemaの型（string、integer、boolean、number、array、object）
// であることを検証します。
func TestParameterTypes_Valid(t *testing.T) {
	tools := GetTools()

	// Define the set of valid JSON Schema types
	// 有効なJSON Schemaの型のセットを定義
	validTypes := map[string]bool{
		"string":  true,
		"integer": true,
		"boolean": true,
		"number":  true,
		"array":   true,
		"object":  true,
	}

	for _, tool := range tools {
		for paramName, param := range tool.InputSchema.Properties {
			// Check that the parameter type is valid
			// パラメータの型が有効であることを確認
			if !validTypes[param.Type] {
				t.Errorf("tool %q parameter %q has invalid type: %q",
					tool.Name, paramName, param.Type)
			}
		}
	}
}

// TestIntegerParameters_Constraints verifies that integer parameters
// that should have constraints (like "tail") have minimum values defined.
// Note: This test currently expects "tail" parameters to have Minimum set,
// but the actual implementation uses string type for "tail".
//
// TestIntegerParameters_Constraintsは、制約を持つべき整数パラメータ
// （"tail"など）に最小値が定義されていることを検証します。
// 注：このテストは現在"tail"パラメータにMinimumが設定されていることを期待していますが、
// 実際の実装では"tail"に文字列型を使用しています。
func TestIntegerParameters_Constraints(t *testing.T) {
	tools := GetTools()

	for _, tool := range tools {
		for paramName, param := range tool.InputSchema.Properties {
			if param.Type == "integer" {
				// Parameters like "tail" should have minimum value
				// "tail"のようなパラメータは最小値を持つべき
				if paramName == "tail" {
					if param.Minimum == nil {
						t.Errorf("tool %q parameter %q should have minimum constraint",
							tool.Name, paramName)
					}
				}
			}
		}
	}
}

// Tests for security-related tools
// セキュリティ関連ツールのテスト

// TestGetAllowedCommandsTool_Structure verifies that the get_allowed_commands tool
// has the correct structure with an optional "container" parameter.
//
// TestGetAllowedCommandsTool_Structureは、get_allowed_commandsツールが
// オプションの"container"パラメータを持つ正しい構造を持っていることを検証します。
func TestGetAllowedCommandsTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the get_allowed_commands tool
	// get_allowed_commandsツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "get_allowed_commands" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("get_allowed_commands tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("get_allowed_commands should have a description")
	}

	// Check that optional container parameter exists
	// オプションのcontainerパラメータが存在することを確認
	if _, exists := tool.InputSchema.Properties["container"]; !exists {
		t.Error("get_allowed_commands should have 'container' parameter")
	}

	// Should not require any parameters (container is optional)
	// パラメータを必要としないはず（containerはオプション）
	if len(tool.InputSchema.Required) != 0 {
		t.Error("get_allowed_commands should not have required parameters")
	}
}

// TestGetSecurityPolicyTool_Structure verifies that the get_security_policy tool
// has the correct structure with no required parameters.
//
// TestGetSecurityPolicyTool_Structureは、get_security_policyツールが
// 必須パラメータなしの正しい構造を持っていることを検証します。
func TestGetSecurityPolicyTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the get_security_policy tool
	// get_security_policyツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "get_security_policy" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("get_security_policy tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("get_security_policy should have a description")
	}

	// Should not require any parameters (no parameters at all)
	// パラメータを必要としないはず（パラメータなし）
	if len(tool.InputSchema.Required) != 0 {
		t.Error("get_security_policy should not have required parameters")
	}
}

// TestSearchLogsTool_Structure verifies that the search_logs tool has the correct
// structure with required "container" and "pattern" parameters.
//
// TestSearchLogsTool_Structureは、search_logsツールが必須の"container"と
// "pattern"パラメータを持つ正しい構造を持っていることを検証します。
func TestSearchLogsTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the search_logs tool
	// search_logsツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "search_logs" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("search_logs tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("search_logs should have a description")
	}

	// Check that container and pattern are required parameters
	// containerとpatternが必須パラメータであることを確認
	requiredParams := []string{"container", "pattern"}
	for _, param := range requiredParams {
		found := false
		for _, req := range tool.InputSchema.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("search_logs should require parameter %q", param)
		}
	}

	// Check that optional parameters exist
	// オプションパラメータが存在することを確認
	optionalParams := []string{"tail", "context_lines"}
	for _, param := range optionalParams {
		if _, exists := tool.InputSchema.Properties[param]; !exists {
			t.Errorf("search_logs should have optional parameter %q", param)
		}
	}
}

// Tests for file access tools
// ファイルアクセスツールのテスト

// TestListFilesTool_Structure verifies that the list_files tool has the correct
// structure with required "container" and optional "path" parameters.
//
// TestListFilesTool_Structureは、list_filesツールが必須の"container"と
// オプションの"path"パラメータを持つ正しい構造を持っていることを検証します。
func TestListFilesTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the list_files tool
	// list_filesツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "list_files" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("list_files tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("list_files should have a description")
	}

	// Check that container is a required parameter
	// containerが必須パラメータであることを確認
	found := false
	for _, req := range tool.InputSchema.Required {
		if req == "container" {
			found = true
			break
		}
	}
	if !found {
		t.Error("list_files should require 'container' parameter")
	}

	// Check that optional path parameter exists
	// オプションのpathパラメータが存在することを確認
	if _, exists := tool.InputSchema.Properties["path"]; !exists {
		t.Error("list_files should have 'path' parameter")
	}
}

// TestReadFileTool_Structure verifies that the read_file tool has the correct
// structure with required "container" and "path" parameters.
//
// TestReadFileTool_Structureは、read_fileツールが必須の"container"と
// "path"パラメータを持つ正しい構造を持っていることを検証します。
func TestReadFileTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the read_file tool
	// read_fileツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "read_file" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("read_file tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("read_file should have a description")
	}

	// Check that container and path are required parameters
	// containerとpathが必須パラメータであることを確認
	requiredParams := []string{"container", "path"}
	for _, param := range requiredParams {
		found := false
		for _, req := range tool.InputSchema.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("read_file should require parameter %q", param)
		}
	}

	// Check that optional max_lines parameter exists
	// オプションのmax_linesパラメータが存在することを確認
	if _, exists := tool.InputSchema.Properties["max_lines"]; !exists {
		t.Error("read_file should have 'max_lines' parameter")
	}
}

// TestGetBlockedPathsTool_Structure verifies that the get_blocked_paths tool
// has the correct structure with an optional "container" parameter.
//
// TestGetBlockedPathsTool_Structureは、get_blocked_pathsツールが
// オプションの"container"パラメータを持つ正しい構造を持っていることを検証します。
func TestGetBlockedPathsTool_Structure(t *testing.T) {
	tools := GetTools()

	// Find the get_blocked_paths tool
	// get_blocked_pathsツールを見つける
	var tool *Tool
	for i := range tools {
		if tools[i].Name == "get_blocked_paths" {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		t.Fatal("get_blocked_paths tool not found")
	}

	// Verify tool has a description
	// ツールに説明があることを検証
	if tool.Description == "" {
		t.Error("get_blocked_paths should have a description")
	}

	// Check that optional container parameter exists
	// オプションのcontainerパラメータが存在することを確認
	if _, exists := tool.InputSchema.Properties["container"]; !exists {
		t.Error("get_blocked_paths should have 'container' parameter")
	}

	// Should not require any parameters (container is optional)
	// パラメータを必要としないはず（containerはオプション）
	if len(tool.InputSchema.Required) != 0 {
		t.Error("get_blocked_paths should not have required parameters")
	}
}

// TestGetContextLines tests the getContextLines helper function that extracts
// surrounding lines for search result context.
//
// TestGetContextLinesは、検索結果のコンテキスト用に周囲の行を抽出する
// getContextLinesヘルパー関数をテストします。
func TestGetContextLines(t *testing.T) {
	// Sample log lines for testing
	// テスト用のサンプルログ行
	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	tests := []struct {
		name        string // Test case name / テストケース名
		index       int    // Index of the matched line / マッチした行のインデックス
		contextSize int    // Number of context lines / コンテキスト行数
		wantLen     int    // Expected number of context lines returned / 期待される返されるコンテキスト行数
	}{
		{"middle with context", 2, 1, 2},  // Line 3 with 1 line before and after / 前後1行のLine 3
		{"start with context", 0, 2, 2},   // Line 1 with 2 lines after only / 後のみ2行のLine 1
		{"end with context", 4, 2, 2},     // Line 5 with 2 lines before only / 前のみ2行のLine 5
		{"zero context", 2, 0, 0},         // No context requested / コンテキストなし
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := getContextLines(lines, tt.index, tt.contextSize)
			if len(context) != tt.wantLen {
				t.Errorf("getContextLines(%d, %d) returned %d lines, want %d",
					tt.index, tt.contextSize, len(context), tt.wantLen)
			}
		})
	}
}

// Note: Functional tests using mock Docker client are now in tools_functional_test.go
// 注：モックDockerクライアントを使用した機能テストはtools_functional_test.goにあります
//
// Implemented functional tests (see tools_functional_test.go):
// 実装された機能テスト（tools_functional_test.goを参照）:
// - TestToolListContainers_Functional
// - TestToolListContainers_Error
// - TestToolGetLogs_Functional
// - TestToolGetLogs_MissingContainer
// - TestToolGetStats_Functional
// - TestToolExecCommand_Functional
// - TestToolExecCommand_Blocked
// - TestToolInspectContainer_Functional
// - TestToolSearchLogs_Functional
// - TestToolListFiles_Functional
// - TestToolListFiles_Blocked
// - TestToolReadFile_Functional
// - TestToolGetAllowedCommands_Functional
// - TestToolGetSecurityPolicy_Functional
// - TestToolGetBlockedPaths_Functional

// TestFormatStats tests the formatStats function that converts container stats to JSON.
// TestFormatStatsはコンテナの統計情報をJSONに変換するformatStats関数をテストします。
func TestFormatStats(t *testing.T) {
	tests := []struct {
		name     string // Test case name / テストケース名
		stats    any    // Input stats / 入力統計情報
		contains string // String that should be in output / 出力に含まれるべき文字列
	}{
		{
			name: "simple struct",
			stats: struct {
				CPU    float64 `json:"cpu"`
				Memory int     `json:"memory"`
			}{
				CPU:    50.5,
				Memory: 1024,
			},
			contains: `"cpu": 50.5`,
		},
		{
			name: "map",
			stats: map[string]any{
				"cpu_percent": 25.0,
				"mem_usage":   512,
			},
			contains: `"cpu_percent": 25`,
		},
		{
			name:     "nil stats",
			stats:    nil,
			contains: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStats(tt.stats)

			// Should not contain Markdown code block markers
			// Markdownコードブロックマーカーを含まないべき
			if contains(result, "```") {
				t.Errorf("formatStats() should not contain Markdown code blocks, got: %s", result)
			}

			// Should contain expected content
			// 期待されるコンテンツを含むべき
			if !contains(result, tt.contains) {
				t.Errorf("formatStats() should contain %q, got: %s", tt.contains, result)
			}
		})
	}
}

// TestFormatStats_ReturnsValidJSON tests that formatStats returns valid JSON.
// TestFormatStats_ReturnsValidJSONはformatStatsが有効なJSONを返すことをテストします。
func TestFormatStats_ReturnsValidJSON(t *testing.T) {
	stats := map[string]any{
		"cpu":    50.0,
		"memory": 1024,
	}

	result := formatStats(stats)

	// The result should be parseable as JSON
	// 結果はJSONとしてパース可能であるべき
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("formatStats() did not return valid JSON: %v", err)
	}

	// Verify the values
	// 値を検証
	if parsed["cpu"] != 50.0 {
		t.Errorf("Expected cpu=50.0, got %v", parsed["cpu"])
	}
}

// contains is a helper function to check if a string contains a substring.
// containsは文字列がサブ文字列を含むかどうかを確認するヘルパー関数です。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring checks if substr exists in s.
// findSubstringはsubstrがsに存在するかどうかを確認します。
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
