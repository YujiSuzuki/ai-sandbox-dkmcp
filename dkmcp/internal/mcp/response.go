// Package mcp provides response helper functions for formatting MCP tool responses.
// These helpers ensure consistent response formatting across all tools, making it
// easier for AI assistants to parse and display the results.
//
// mcpパッケージはMCPツールレスポンスをフォーマットするためのレスポンスヘルパー関数を提供します。
// これらのヘルパーは、すべてのツールで一貫したレスポンスフォーマットを保証し、
// AIアシスタントが結果を解析して表示することを容易にします。
package mcp

import (
	"encoding/json"
	"fmt"
)

// textResponse creates a standard MCP text response.
// It wraps the given text in the MCP content format with type "text".
// This is the most basic response type used when returning plain text results.
//
// textResponseは標準的なMCPテキストレスポンスを作成します。
// 指定されたテキストを"text"タイプのMCPコンテンツ形式でラップします。
// これはプレーンテキストの結果を返すときに使用される最も基本的なレスポンスタイプです。
func textResponse(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
	}
}

// jsonTextResponse creates an MCP response with JSON data formatted as text.
// It marshals the data to pretty-printed JSON and returns it as a text response.
// This is useful when returning structured data that should be human-readable.
//
// jsonTextResponseはJSONデータをテキストとしてフォーマットしたMCPレスポンスを作成します。
// データを整形されたJSONにマーシャルし、テキストレスポンスとして返します。
// これは人が読める形式で構造化されたデータを返すときに便利です。
func jsonTextResponse(data any) (map[string]any, error) {
	// Marshal data to indented JSON for readability
	// 読みやすさのためにデータをインデント付きJSONにマーシャル
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return textResponse(string(jsonData)), nil
}

// jsonCodeBlockResponse creates an MCP response with JSON in a markdown code block.
// It includes a title and wraps the JSON data in a ```json code block for
// better rendering in markdown-capable displays.
//
// jsonCodeBlockResponseはマークダウンコードブロック内にJSONを含むMCPレスポンスを作成します。
// タイトルを含み、JSONデータを```jsonコードブロックでラップして、
// マークダウン対応ディスプレイでより良いレンダリングを実現します。
func jsonCodeBlockResponse(title string, data any) (map[string]any, error) {
	// Marshal data to indented JSON for readability
	// 読みやすさのためにデータをインデント付きJSONにマーシャル
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Format with title and markdown code block
	// タイトルとマークダウンコードブロックでフォーマット
	text := fmt.Sprintf("%s:\n\n```json\n%s\n```", title, string(jsonData))
	return textResponse(text), nil
}

// errorTextResponse creates an MCP response for error messages.
// It formats the error message using printf-style formatting and returns
// it as a text response. This provides consistent error message formatting.
//
// errorTextResponseはエラーメッセージ用のMCPレスポンスを作成します。
// printf形式のフォーマットを使用してエラーメッセージをフォーマットし、
// テキストレスポンスとして返します。これにより一貫したエラーメッセージのフォーマットが提供されます。
func errorTextResponse(format string, args ...any) map[string]any {
	return textResponse(fmt.Sprintf(format, args...))
}

// prefixedTextResponse creates an MCP response with a prefix and content.
// The prefix and content are separated by two newlines for clear visual separation.
// This is useful for adding context or headers to response content.
//
// prefixedTextResponseはプレフィックスとコンテンツを含むMCPレスポンスを作成します。
// プレフィックスとコンテンツは明確な視覚的分離のために2つの改行で区切られます。
// これはレスポンスコンテンツにコンテキストやヘッダーを追加するときに便利です。
func prefixedTextResponse(prefix string, content string) map[string]any {
	return textResponse(fmt.Sprintf("%s\n\n%s", prefix, content))
}

// containerFileResponse creates an MCP response for container file operations.
// It formats the response with the operation type, container name, path, and content.
// This provides a consistent format for file listing and reading operations.
//
// Example output:
//
//	Files in mycontainer:/app
//
//	file1.txt
//	file2.txt
//
// containerFileResponseはコンテナファイル操作用のMCPレスポンスを作成します。
// 操作タイプ、コンテナ名、パス、コンテンツを含むフォーマットでレスポンスを作成します。
// これにより、ファイルの一覧表示と読み取り操作の一貫したフォーマットが提供されます。
//
// 出力例：
//
//	Files in mycontainer:/app
//
//	file1.txt
//	file2.txt
func containerFileResponse(operation, container, path, content string) map[string]any {
	return textResponse(fmt.Sprintf("%s %s:%s\n\n%s", operation, container, path, content))
}
