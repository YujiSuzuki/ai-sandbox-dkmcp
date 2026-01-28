// Package mcp provides tests for the response helper functions.
// These tests verify that all response helpers produce correctly formatted
// MCP responses that AI assistants can properly parse and display.
//
// mcpパッケージはレスポンスヘルパー関数のテストを提供します。
// これらのテストは、すべてのレスポンスヘルパーがAIアシスタントが適切に解析して
// 表示できる正しくフォーマットされたMCPレスポンスを生成することを検証します。
package mcp

import (
	"strings"
	"testing"
)

// TestTextResponse verifies that textResponse creates a properly formatted
// MCP response with the text content wrapped in the expected structure.
//
// TestTextResponseは、textResponseがテキストコンテンツを期待される構造で
// ラップした適切にフォーマットされたMCPレスポンスを作成することを検証します。
func TestTextResponse(t *testing.T) {
	// Create a text response
	// テキストレスポンスを作成
	resp := textResponse("hello world")

	// Verify the content structure is correct
	// コンテンツ構造が正しいことを検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Should contain exactly one content item
	// 正確に1つのコンテンツアイテムを含むべき
	if len(content) != 1 {
		t.Fatalf("content should have 1 element, got %d", len(content))
	}

	// Verify the type is "text"
	// タイプが"text"であることを検証
	if content[0]["type"] != "text" {
		t.Errorf("type = %q, want %q", content[0]["type"], "text")
	}

	// Verify the text content matches
	// テキストコンテンツが一致することを検証
	if content[0]["text"] != "hello world" {
		t.Errorf("text = %q, want %q", content[0]["text"], "hello world")
	}
}

// TestJsonTextResponse verifies that jsonTextResponse correctly marshals
// data to JSON and returns it as a text response.
//
// TestJsonTextResponseは、jsonTextResponseがデータを正しくJSONに
// マーシャルし、テキストレスポンスとして返すことを検証します。
func TestJsonTextResponse(t *testing.T) {
	// Create test data
	// テストデータを作成
	data := map[string]string{"key": "value"}

	// Generate JSON text response
	// JSONテキストレスポンスを生成
	resp, err := jsonTextResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the content structure
	// コンテンツ構造を検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Extract the text content
	// テキストコンテンツを抽出
	text := content[0]["text"].(string)

	// Verify JSON key is present
	// JSONキーが存在することを検証
	if !strings.Contains(text, `"key"`) {
		t.Error("response should contain key")
	}

	// Verify JSON value is present
	// JSON値が存在することを検証
	if !strings.Contains(text, `"value"`) {
		t.Error("response should contain value")
	}
}

// TestJsonCodeBlockResponse verifies that jsonCodeBlockResponse creates
// a response with a title and JSON wrapped in a markdown code block.
//
// TestJsonCodeBlockResponseは、jsonCodeBlockResponseがタイトルと
// マークダウンコードブロックでラップされたJSONを含むレスポンスを作成することを検証します。
func TestJsonCodeBlockResponse(t *testing.T) {
	// Create test data
	// テストデータを作成
	data := map[string]string{"key": "value"}

	// Generate JSON code block response
	// JSONコードブロックレスポンスを生成
	resp, err := jsonCodeBlockResponse("Test Title", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the content structure
	// コンテンツ構造を検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Extract the text content
	// テキストコンテンツを抽出
	text := content[0]["text"].(string)

	// Verify title is present
	// タイトルが存在することを検証
	if !strings.Contains(text, "Test Title:") {
		t.Error("response should contain title")
	}

	// Verify markdown code block markers are present
	// マークダウンコードブロックマーカーが存在することを検証
	if !strings.Contains(text, "```json") {
		t.Error("response should contain json code block")
	}

	// Verify JSON content is present
	// JSONコンテンツが存在することを検証
	if !strings.Contains(text, `"key"`) {
		t.Error("response should contain key")
	}
}

// TestErrorTextResponse verifies that errorTextResponse correctly formats
// error messages using printf-style formatting.
//
// TestErrorTextResponseは、errorTextResponseがprintf形式のフォーマットを
// 使用してエラーメッセージを正しくフォーマットすることを検証します。
func TestErrorTextResponse(t *testing.T) {
	// Create error response with formatted message
	// フォーマットされたメッセージでエラーレスポンスを作成
	resp := errorTextResponse("error: %s", "something went wrong")

	// Verify the content structure
	// コンテンツ構造を検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Extract and verify the error message
	// エラーメッセージを抽出して検証
	text := content[0]["text"].(string)
	if text != "error: something went wrong" {
		t.Errorf("text = %q, want %q", text, "error: something went wrong")
	}
}

// TestPrefixedTextResponse verifies that prefixedTextResponse correctly
// combines a prefix and content with proper separation.
//
// TestPrefixedTextResponseは、prefixedTextResponseがプレフィックスと
// コンテンツを適切な区切りで正しく結合することを検証します。
func TestPrefixedTextResponse(t *testing.T) {
	// Create prefixed response
	// プレフィックス付きレスポンスを作成
	resp := prefixedTextResponse("Prefix", "Content")

	// Verify the content structure
	// コンテンツ構造を検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Extract and verify the combined text
	// 結合されたテキストを抽出して検証
	text := content[0]["text"].(string)

	// Verify prefix is present
	// プレフィックスが存在することを検証
	if !strings.Contains(text, "Prefix") {
		t.Error("response should contain prefix")
	}

	// Verify content is present
	// コンテンツが存在することを検証
	if !strings.Contains(text, "Content") {
		t.Error("response should contain content")
	}
}

// TestContainerFileResponse verifies that containerFileResponse correctly
// formats container file operation results with container name and path.
//
// TestContainerFileResponseは、containerFileResponseがコンテナ名とパスを含む
// コンテナファイル操作の結果を正しくフォーマットすることを検証します。
func TestContainerFileResponse(t *testing.T) {
	// Create container file response
	// コンテナファイルレスポンスを作成
	resp := containerFileResponse("Files in", "mycontainer", "/path", "file1\nfile2")

	// Verify the content structure
	// コンテンツ構造を検証
	content, ok := resp["content"].([]map[string]any)
	if !ok {
		t.Fatal("content should be []map[string]any")
	}

	// Extract and verify the formatted text
	// フォーマットされたテキストを抽出して検証
	text := content[0]["text"].(string)

	// Verify formatted path is present (operation container:path format)
	// フォーマットされたパスが存在することを検証（operation container:path形式）
	if !strings.Contains(text, "Files in mycontainer:/path") {
		t.Errorf("text should contain formatted path, got: %s", text)
	}

	// Verify file content is present
	// ファイルコンテンツが存在することを検証
	if !strings.Contains(text, "file1") {
		t.Error("response should contain file content")
	}
}

// TestJsonTextResponse_InvalidData verifies that jsonTextResponse returns
// an error when given data that cannot be serialized to JSON.
//
// TestJsonTextResponse_InvalidDataは、JSONにシリアライズできないデータが
// 渡されたときにjsonTextResponseがエラーを返すことを検証します。
func TestJsonTextResponse_InvalidData(t *testing.T) {
	// Create a channel which cannot be serialized to JSON
	// JSONにシリアライズできないチャネルを作成
	ch := make(chan int)

	// Attempt to create JSON response - should fail
	// JSONレスポンスの作成を試行 - 失敗するはず
	_, err := jsonTextResponse(ch)
	if err == nil {
		t.Error("expected error for non-serializable data")
	}
}

// TestJsonCodeBlockResponse_InvalidData verifies that jsonCodeBlockResponse
// returns an error when given data that cannot be serialized to JSON.
//
// TestJsonCodeBlockResponse_InvalidDataは、JSONにシリアライズできないデータが
// 渡されたときにjsonCodeBlockResponseがエラーを返すことを検証します。
func TestJsonCodeBlockResponse_InvalidData(t *testing.T) {
	// Create a channel which cannot be serialized to JSON
	// JSONにシリアライズできないチャネルを作成
	ch := make(chan int)

	// Attempt to create JSON code block response - should fail
	// JSONコードブロックレスポンスの作成を試行 - 失敗するはず
	_, err := jsonCodeBlockResponse("Title", ch)
	if err == nil {
		t.Error("expected error for non-serializable data")
	}
}
