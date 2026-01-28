// Package client provides tests for the DockMCP HTTP client.
// These tests verify the client's ability to connect to the DockMCP server,
// perform health checks, and call MCP tools via the SSE/HTTP protocol.
//
// clientパッケージのテストファイルです。
// DockMCPサーバーへの接続、ヘルスチェックの実行、
// SSE/HTTPプロトコル経由でのMCPツール呼び出しの機能を検証します。
package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewClient verifies that NewClient correctly initializes a new Client instance
// with the provided base URL and all required internal structures.
//
// TestNewClientは、NewClientが指定されたベースURLと
// 必要な内部構造を持つ新しいClientインスタンスを正しく初期化することを検証します。
func TestNewClient(t *testing.T) {
	// Create a new client with a test URL
	// テストURLで新しいクライアントを作成
	c := NewClient("http://localhost:8080")

	// Verify the client was created (not nil)
	// クライアントが作成されたことを確認（nilでない）
	if c == nil {
		t.Fatal("NewClient returned nil")
	}

	// Verify the base URL was set correctly
	// ベースURLが正しく設定されたことを確認
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL http://localhost:8080, got %s", c.baseURL)
	}
}

// TestHealthCheck tests the HealthCheck method with various server responses.
// It uses table-driven tests to verify both healthy and unhealthy server scenarios.
//
// TestHealthCheckは様々なサーバーレスポンスでHealthCheckメソッドをテストします。
// テーブル駆動テストを使用して、正常なサーバーと異常なサーバーの両方のシナリオを検証します。
func TestHealthCheck(t *testing.T) {
	// Define test cases for different server health scenarios
	// 異なるサーバーヘルスシナリオのテストケースを定義
	tests := []struct {
		name       string // Test case name / テストケース名
		statusCode int    // HTTP status code to return / 返すHTTPステータスコード
		wantErr    bool   // Whether an error is expected / エラーが期待されるかどうか
	}{
		{
			// Test case: Server responds with 200 OK (healthy)
			// テストケース：サーバーが200 OKを返す（正常）
			name:       "healthy server",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			// Test case: Server responds with 500 Internal Server Error (unhealthy)
			// テストケース：サーバーが500 Internal Server Errorを返す（異常）
			name:       "unhealthy server",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	// Run each test case
	// 各テストケースを実行
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP server that responds with the test status code
			// テストステータスコードで応答するモックHTTPサーバーを作成
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request is to the correct path
				// リクエストが正しいパスへのものか確認
				if r.URL.Path != "/health" {
					t.Errorf("Expected /health, got %s", r.URL.Path)
				}
				// Write the test status code and a JSON response
				// テストステータスコードとJSONレスポンスを書き込む
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}))
			defer server.Close()

			// Create client pointing to the mock server
			// モックサーバーを指すクライアントを作成
			c := NewClient(server.URL)
			err := c.HealthCheck()

			// Verify error matches expectation
			// エラーが期待と一致することを確認
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// mockSSEServer creates a test server that simulates DockMCP's SSE behavior.
// It handles:
// - GET /sse: Establishes SSE connection and sends session ID via "endpoint" event
// - POST /message: Receives JSON-RPC requests and sends responses via SSE channel
// - GET /health: Returns health status
//
// This mock server is essential for testing the client's SSE-based communication
// without requiring a real DockMCP server.
//
// mockSSEServerはDockMCPのSSE動作をシミュレートするテストサーバーを作成します。
// 以下を処理します：
// - GET /sse: SSE接続を確立し、「endpoint」イベント経由でセッションIDを送信
// - POST /message: JSON-RPCリクエストを受信し、SSEチャネル経由でレスポンスを送信
// - GET /health: ヘルスステータスを返す
//
// このモックサーバーは、実際のDockMCPサーバーなしで
// クライアントのSSEベース通信をテストするために不可欠です。
//
// Parameters:
//   - t: Testing context for error reporting
//   - responses: Map of tool names to their expected JSON-RPC responses
//
// Returns:
//   - *httptest.Server: A running test server that simulates DockMCP behavior
//
// パラメータ：
//   - t: エラー報告用のテストコンテキスト
//   - responses: ツール名から期待されるJSON-RPCレスポンスへのマップ
//
// 戻り値：
//   - *httptest.Server: DockMCPの動作をシミュレートする実行中のテストサーバー
func mockSSEServer(t *testing.T, responses map[string]JSONRPCResponse) *httptest.Server {
	// Mutex for thread-safe access to shared state
	// 共有状態へのスレッドセーフなアクセス用ミューテックス
	var mu sync.Mutex

	// Fixed session ID for testing purposes
	// テスト用の固定セッションID
	sessionID := "test-session-123"

	// Map of session IDs to their response channels for SSE message delivery
	// SSEメッセージ配信用のセッションIDからレスポンスチャネルへのマップ
	responseChannels := make(map[string]chan []byte)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle SSE connection endpoint
		// SSE接続エンドポイントを処理
		if r.URL.Path == "/sse" && r.Method == "GET" {
			// SSE connection: Send "endpoint" event to notify client of session ID
			// SSE接続：クライアントにセッションIDを通知する「endpoint」イベントを送信

			// Set SSE-specific headers
			// SSE固有のヘッダーを設定
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send the endpoint event with the message URL containing session ID
			// セッションIDを含むメッセージURLでendpointイベントを送信
			endpointURL := fmt.Sprintf("/message?sessionId=%s", sessionID)
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)

			// Flush the response to ensure immediate delivery
			// 即座の配信を確保するためにレスポンスをフラッシュ
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Create response channel for this session to receive tool call responses
			// ツール呼び出しレスポンスを受信するためのこのセッション用レスポンスチャネルを作成
			mu.Lock()
			respChan := make(chan []byte, 10)
			responseChannels[sessionID] = respChan
			mu.Unlock()

			// Keep connection alive and stream responses as they arrive
			// 接続を維持し、到着したレスポンスをストリーミング
			for {
				select {
				case <-r.Context().Done():
					// Client disconnected, exit the handler
					// クライアントが切断、ハンドラを終了
					return
				case msg := <-respChan:
					// Send message as SSE data event
					// メッセージをSSEデータイベントとして送信
					fmt.Fprintf(w, "data: %s\n\n", msg)
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				case <-time.After(30 * time.Second):
					// Keep-alive timeout to prevent hanging connections
					// ハングした接続を防ぐためのキープアライブタイムアウト
					return
				}
			}

		} else if r.URL.Path == "/message" && r.Method == "POST" {
			// Message endpoint: Handle JSON-RPC requests (initialize and tools/call)
			// メッセージエンドポイント：JSON-RPCリクエスト（initializeとtools/call）を処理

			// Get session ID from query parameter
			// クエリパラメータからセッションIDを取得
			sessionID := r.URL.Query().Get("sessionId")

			// Decode the JSON-RPC request
			// JSON-RPCリクエストをデコード
			var req JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Failed to decode request", http.StatusBadRequest)
				return
			}

			var respBytes []byte
			var err error

			// Handle different JSON-RPC methods
			// 異なるJSON-RPCメソッドを処理
			if req.Method == "initialize" {
				// Handle initialize handshake - required before any tool calls
				// 初期化ハンドシェイクを処理 - ツール呼び出しの前に必要
				initResp := InitializeResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  map[string]interface{}{"status": "ok"},
				}
				respBytes, err = json.Marshal(initResp)
				if err != nil {
					http.Error(w, "Failed to marshal initialize response", http.StatusInternalServerError)
					return
				}
			} else if req.Method == "tools/call" {
				// Handle tool call - return the pre-configured response
				// ツール呼び出しを処理 - 事前設定されたレスポンスを返す
				var response JSONRPCResponse
				for _, resp := range responses {
					response = resp
					break // Just return the first response for simplicity / 簡単のため最初のレスポンスを返す
				}
				response.ID = req.ID
				respBytes, err = json.Marshal(response)
				if err != nil {
					http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
					return
				}
			} else {
				// Unexpected method - return error
				// 予期しないメソッド - エラーを返す
				http.Error(w, fmt.Sprintf("Expected method 'initialize' or 'tools/call', got '%s'", req.Method), http.StatusBadRequest)
				return
			}

			// Send response via SSE channel for the session
			// セッションのSSEチャネル経由でレスポンスを送信
			mu.Lock()
			respChan, exists := responseChannels[sessionID]
			mu.Unlock()

			if exists {
				select {
				case respChan <- respBytes:
					// Successfully queued response, return 202 Accepted
					// レスポンスを正常にキューに追加、202 Acceptedを返す
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
				case <-time.After(5 * time.Second):
					// Timeout trying to send response
					// レスポンス送信中のタイムアウト
					http.Error(w, "Timeout sending response", http.StatusInternalServerError)
				}
			} else {
				// Session not found - return error
				// セッションが見つからない - エラーを返す
				http.Error(w, "Invalid session ID", http.StatusBadRequest)
			}
		} else if r.URL.Path == "/health" && r.Method == "GET" {
			// Health check endpoint - return OK status
			// ヘルスチェックエンドポイント - OKステータスを返す
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		} else {
			// Unknown endpoint - return 404
			// 不明なエンドポイント - 404を返す
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
}

// TestCallTool tests the CallTool method with various response scenarios.
// It verifies that the client correctly handles:
// - Successful tool calls with valid responses
// - JSON-RPC level errors (protocol errors)
// - Tool execution errors (tool-level errors)
//
// This test uses the mockSSEServer to simulate the DockMCP server's behavior
// and verify the client's response handling.
//
// TestCallToolは様々なレスポンスシナリオでCallToolメソッドをテストします。
// クライアントが以下を正しく処理することを検証します：
// - 有効なレスポンスを持つ成功したツール呼び出し
// - JSON-RPCレベルのエラー（プロトコルエラー）
// - ツール実行エラー（ツールレベルのエラー）
//
// このテストはmockSSEServerを使用してDockMCPサーバーの動作をシミュレートし、
// クライアントのレスポンス処理を検証します。
func TestCallTool(t *testing.T) {
	// Define test cases for different tool call scenarios
	// 異なるツール呼び出しシナリオのテストケースを定義
	tests := []struct {
		name       string           // Test case name / テストケース名
		toolName   string           // Tool name to call / 呼び出すツール名
		response   JSONRPCResponse  // Expected response from server / サーバーからの期待レスポンス
		wantErr    bool             // Whether an error is expected / エラーが期待されるかどうか
		errMessage string           // Expected error message substring (if wantErr is true) / 期待されるエラーメッセージの部分文字列（wantErrがtrueの場合）
	}{
		{
			// Test case: Successful tool call returning container list
			// テストケース：コンテナリストを返す成功したツール呼び出し
			name:     "successful tool call",
			toolName: "list_containers",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: &ToolResult{
					Content: []Content{
						{
							Type: "text",
							Text: `[{"name":"test","id":"123","image":"nginx","state":"running","status":"Up"}]`,
						},
					},
					IsError: false,
				},
			},
			wantErr: false,
		},
		{
			// Test case: JSON-RPC protocol error (e.g., internal server error)
			// テストケース：JSON-RPCプロトコルエラー（例：内部サーバーエラー）
			name:     "JSON-RPC error",
			toolName: "list_containers",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &JSONRPCError{
					Code:    -32603,
					Message: "Internal error",
				},
			},
			wantErr:    true,
			errMessage: "JSON-RPC error",
		},
		{
			// Test case: Tool execution error (tool returned IsError: true)
			// テストケース：ツール実行エラー（ツールがIsError: trueを返した）
			name:     "tool execution error",
			toolName: "list_containers",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: &ToolResult{
					Content: []Content{
						{
							Type: "text",
							Text: "Container not found",
						},
					},
					IsError: true,
				},
			},
			wantErr:    true,
			errMessage: "tool call failed",
		},
	}

	// Run each test case
	// 各テストケースを実行
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server with the test response
			// テストレスポンスを持つモックサーバーを作成
			server := mockSSEServer(t, map[string]JSONRPCResponse{
				tt.toolName: tt.response,
			})
			defer server.Close()

			// Create client and ensure cleanup
			// クライアントを作成しクリーンアップを確保
			c := NewClient(server.URL)
			defer c.Close()

			// Connect to SSE endpoint first (required before CallTool)
			// まずSSEエンドポイントに接続（CallToolの前に必要）
			err := c.Connect()
			if err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Give the SSE reader goroutine time to start and be ready
			// SSEリーダーgoroutineが開始して準備完了するまで待機
			time.Sleep(100 * time.Millisecond)

			// Call the tool with empty arguments
			// 空の引数でツールを呼び出す
			result, err := c.CallTool(tt.toolName, map[string]interface{}{})

			// Verify error matches expectation
			// エラーが期待と一致することを確認
			if (err != nil) != tt.wantErr {
				t.Errorf("CallTool() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If error was expected, verify error message contains expected substring
			// エラーが期待されていた場合、エラーメッセージに期待される部分文字列が含まれることを確認
			if tt.wantErr && tt.errMessage != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errMessage)
				} else if !strings.Contains(err.Error(), tt.errMessage) {
					t.Errorf("Expected error containing %q, got %q", tt.errMessage, err.Error())
				}
			}

			// If no error was expected, verify result is not nil
			// エラーが期待されていなかった場合、結果がnilでないことを確認
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result for successful call")
			}
		})
	}
}
