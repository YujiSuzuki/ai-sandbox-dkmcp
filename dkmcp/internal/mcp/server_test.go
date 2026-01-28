// Package mcp provides tests for the MCP server implementation.
// This file contains integration tests that verify the SSE connection flow
// and JSON-RPC message handling between clients and the server.
//
// mcpパッケージはMCPサーバー実装のテストを提供します。
// このファイルにはSSE接続フローとクライアント・サーバー間の
// JSON-RPCメッセージ処理を検証する統合テストが含まれています。
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/docker"
)

// TestWithVerbosity tests that the WithVerbosity option correctly sets verbosity levels.
//
// TestWithVerbosityはWithVerbosityオプションが正しくverbosityレベルを設定することをテストします。
func TestWithVerbosity(t *testing.T) {
	dockerClient := &docker.Client{}

	tests := []struct {
		name            string
		verbosityLevel  int
		expectVerbosity int
	}{
		{
			name:            "level_0_normal",
			verbosityLevel:  0,
			expectVerbosity: 0,
		},
		{
			name:            "level_1_json_output",
			verbosityLevel:  1,
			expectVerbosity: 1,
		},
		{
			name:            "level_2_debug_json",
			verbosityLevel:  2,
			expectVerbosity: 2,
		},
		{
			name:            "level_3_full_debug",
			verbosityLevel:  3,
			expectVerbosity: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(dockerClient, 8080, WithVerbosity(tt.verbosityLevel))
			if server.verbosity != tt.expectVerbosity {
				t.Errorf("Expected verbosity=%v, got %v", tt.expectVerbosity, server.verbosity)
			}
		})
	}
}

// TestNewServerWithoutOptions tests that NewServer works without any options.
//
// TestNewServerWithoutOptionsはNewServerがオプションなしで動作することをテストします。
func TestNewServerWithoutOptions(t *testing.T) {
	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 8080)

	if server.verbosity != 0 {
		t.Errorf("Expected verbosity=0 by default, got %v", server.verbosity)
	}
	if server.port != 8080 {
		t.Errorf("Expected port=8080, got %v", server.port)
	}
	if server.clients == nil {
		t.Error("Expected clients map to be initialized")
	}
}

// TestLogVerboseRequest tests the verbose request logging function.
// It verifies that the function doesn't panic and handles various request types.
//
// TestLogVerboseRequestは詳細リクエストログ関数をテストします。
// 関数がパニックせず、様々なリクエストタイプを処理することを検証します。
func TestLogVerboseRequest(t *testing.T) {
	// Suppress log output during test
	// テスト中はログ出力を抑制
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 8080, WithVerbosity(3))

	// Create mock clients for testing
	// テスト用のモッククライアントを作成
	initializedClient := &client{
		id:         "test-client-1",
		clientName: "claude-code",
		remoteAddr: "127.0.0.1:12345",
	}
	uninitializedClient := &client{
		id:         "test-client-2",
		clientName: "",
		remoteAddr: "127.0.0.1:12346",
	}

	tests := []struct {
		name   string
		client *client
		req    *JSONRPCRequest
	}{
		{
			name:   "initialize_request",
			client: uninitializedClient,
			req: &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
				Params: map[string]any{
					"clientInfo": map[string]string{
						"name":    "test-client",
						"version": "1.0.0",
					},
				},
			},
		},
		{
			name:   "tools_list_request",
			client: initializedClient,
			req: &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/list",
			},
		},
		{
			name:   "tools_call_request",
			client: initializedClient,
			req: &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "tools/call",
				Params: map[string]any{
					"name": "list_containers",
					"arguments": map[string]any{
						"all": true,
					},
				},
			},
		},
		{
			name:   "request_with_nil_params",
			client: initializedClient,
			req: &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "tools/list",
				Params:  nil,
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal request to simulate raw JSON bytes from client
			// クライアントからの生JSONバイトをシミュレートするためにリクエストをmarshal
			rawJSON, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal test request: %v", err)
			}
			// Should not panic
			// パニックしないはず
			server.logVerboseRequest(tt.client, tt.req, rawJSON, uint64(i+1))
		})
	}
}

// TestLogVerboseResponse tests the verbose response logging function.
// It verifies that the function doesn't panic and handles various response types.
//
// TestLogVerboseResponseは詳細レスポンスログ関数をテストします。
// 関数がパニックせず、様々なレスポンスタイプを処理することを検証します。
func TestLogVerboseResponse(t *testing.T) {
	// Suppress log output during test
	// テスト中はログ出力を抑制
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 8080, WithVerbosity(3))

	// Create mock client for testing
	// テスト用のモッククライアントを作成
	mockClient := &client{
		id:         "test-client-1",
		clientName: "claude-code",
		remoteAddr: "127.0.0.1:12345",
	}

	tests := []struct {
		name string
		resp *JSONRPCResponse
	}{
		{
			name: "success_response",
			resp: &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]any{
					"tools": []string{"list_containers", "get_logs"},
				},
			},
		},
		{
			name: "error_response",
			resp: &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      2,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
		},
		{
			name: "response_with_nil_result",
			resp: &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      3,
				Result:  nil,
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			// パニックしないはず
			server.logVerboseResponse(mockClient, tt.resp, uint64(i+1))
		})
	}
}

// TestOriginValidationMiddleware tests the Origin header validation middleware.
// Per MCP specification, servers MUST validate the Origin header to prevent DNS rebinding attacks.
//
// TestOriginValidationMiddlewareはOriginヘッダー検証ミドルウェアをテストします。
// MCP仕様に従い、サーバーはDNSリバインディング攻撃を防ぐためにOriginヘッダーを検証しなければなりません。
func TestOriginValidationMiddleware(t *testing.T) {
	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 0)

	// Create test handler that records if it was called
	// 呼び出されたかどうかを記録するテストハンドラを作成
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with origin validation middleware
	// Origin検証ミドルウェアでラップ
	handler := server.originValidationMiddleware(testHandler)

	tests := []struct {
		name           string // Test case name / テストケース名
		origin         string // Origin header value / Originヘッダーの値
		expectAllowed  bool   // Should request be allowed / リクエストが許可されるべきか
		expectStatus   int    // Expected HTTP status / 期待されるHTTPステータス
	}{
		// Allowed cases - localhost variations
		// 許可されるケース - localhost のバリエーション
		{
			name:          "no_origin_header",
			origin:        "",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "localhost_http",
			origin:        "http://localhost",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "localhost_https",
			origin:        "https://localhost",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "localhost_with_port",
			origin:        "http://localhost:3000",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "127.0.0.1_http",
			origin:        "http://127.0.0.1",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "127.0.0.1_with_port",
			origin:        "http://127.0.0.1:8080",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "ipv6_localhost",
			origin:        "http://[::1]",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "ipv6_localhost_with_port",
			origin:        "http://[::1]:8080",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},

		// Blocked cases - external origins (DNS rebinding attack prevention)
		// ブロックされるケース - 外部オリジン（DNSリバインディング攻撃防止）
		{
			name:          "external_domain",
			origin:        "http://example.com",
			expectAllowed: false,
			expectStatus:  http.StatusForbidden,
		},
		{
			name:          "external_domain_https",
			origin:        "https://evil.com",
			expectAllowed: false,
			expectStatus:  http.StatusForbidden,
		},
		{
			name:          "external_with_localhost_substring",
			origin:        "http://localhost.evil.com",
			expectAllowed: false,
			expectStatus:  http.StatusForbidden,
		},
		{
			name:          "external_ip",
			origin:        "http://192.168.1.1",
			expectAllowed: false,
			expectStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset handler called flag
			// ハンドラ呼び出しフラグをリセット
			handlerCalled = false

			// Create test request with Origin header
			// Originヘッダー付きのテストリクエストを作成
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			// Execute request through middleware
			// ミドルウェア経由でリクエストを実行
			handler.ServeHTTP(rr, req)

			// Verify response status
			// レスポンスステータスを検証
			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, rr.Code)
			}

			// Verify handler was called or blocked
			// ハンドラが呼び出されたかブロックされたかを検証
			if handlerCalled != tt.expectAllowed {
				t.Errorf("Handler called = %v, expected %v", handlerCalled, tt.expectAllowed)
			}
		})
	}
}

// TestSSEMessageEventFormat verifies that SSE messages are sent with the correct
// "event: message" format as required by the MCP 2024-11-05 specification.
//
// TestSSEMessageEventFormatは、MCP 2024-11-05仕様で要求される
// 正しい "event: message" 形式でSSEメッセージが送信されることを検証します。
func TestSSEMessageEventFormat(t *testing.T) {
	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 0)

	// Create test handler
	// テストハンドラを作成
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sse", server.handleSSE)
	mux.HandleFunc("POST /message", server.handleMessage)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Connect to SSE endpoint with a context that can be cancelled
	// キャンセル可能なコンテキストでSSEエンドポイントに接続
	client := &http.Client{Timeout: 10 * time.Second}
	sseReq, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
	sseReq.Header.Set("Accept", "text/event-stream")

	sseResp, err := client.Do(sseReq)
	if err != nil {
		t.Fatalf("Failed to connect to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	// Create channels for communication between goroutines
	// ゴルーチン間の通信用チャネルを作成
	sessionIDChan := make(chan string, 1)
	eventMessageChan := make(chan bool, 1)
	doneChan := make(chan struct{})

	// Read SSE events in a goroutine
	// ゴルーチンでSSEイベントを読み取り
	go func() {
		defer close(doneChan)
		scanner := bufio.NewScanner(sseResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for session ID in endpoint event
			// endpointイベントでセッションIDを探す
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if strings.Contains(data, "sessionId=") {
					idx := strings.Index(data, "sessionId=")
					select {
					case sessionIDChan <- data[idx+len("sessionId="):]:
					default:
					}
				}
			}
			// Check for "event: message" format
			// "event: message" 形式を確認
			if line == "event: message" {
				select {
				case eventMessageChan <- true:
				default:
				}
			}
		}
	}()

	// Wait for session ID
	// セッションIDを待機
	var sessionID string
	select {
	case sessionID = <-sessionIDChan:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for session ID")
	}

	// Initialize session
	// セッションを初期化
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]any{
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	initBody, _ := json.Marshal(initReq)

	initResp, err := http.Post(
		ts.URL+"/message?sessionId="+sessionID,
		"application/json",
		bytes.NewReader(initBody),
	)
	if err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}
	initResp.Body.Close()

	// Wait for "event: message" format in SSE stream
	// SSEストリームで "event: message" 形式を待機
	select {
	case <-eventMessageChan:
		// Success - found "event: message" format
		// 成功 - "event: message" 形式を発見
	case <-time.After(3 * time.Second):
		t.Error("Expected 'event: message' format in SSE stream but not found")
	}
}

// TestUnknownToolReturnsErrorViaSSE tests that calling an unknown tool returns
// an error via the SSE channel rather than the HTTP response. This validates
// MCP's requirement that all responses (including errors) are
// delivered through the SSE event stream.
//
// TestUnknownToolReturnsErrorViaSSEは、不明なツールを呼び出すと
// HTTPレスポンスではなくSSEチャネル経由でエラーが返されることをテストします。
// これはすべてのレスポンス（エラーを含む）がSSEイベントストリームを通じて
// 配信されるというMCPプロトコルの要件を検証します。
func TestUnknownToolReturnsErrorViaSSE(t *testing.T) {
	// Create mock docker client for testing
	// テスト用のモックDockerクライアントを作成
	dockerClient := &docker.Client{}

	// Create server instance with port 0 (not actually listening)
	// ポート0でサーバーインスタンスを作成（実際にはリッスンしない）
	server := NewServer(dockerClient, 0)

	// Create test HTTP handler with same routes as the actual server
	// 実際のサーバーと同じルートを持つテストHTTPハンドラを作成
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sse", server.handleSSE)
	mux.HandleFunc("POST /message", server.handleMessage)
	mux.HandleFunc("GET /health", server.handleHealth)

	// Start httptest server for isolated testing
	// 分離されたテスト用にhttptestサーバーを起動
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Step 1: Connect to SSE endpoint and get session ID
	// ステップ1：SSEエンドポイントに接続してセッションIDを取得
	sseReq, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
	sseReq.Header.Set("Accept", "text/event-stream")

	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("Failed to connect to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	// Read session ID from SSE endpoint event
	// The MCP spec requires the server to send an endpoint event with the message URL
	// SSEのendpointイベントからセッションIDを読み取る
	// MCP仕様ではサーバーがメッセージURLを含むendpointイベントを送信することが求められる
	scanner := bufio.NewScanner(sseResp.Body)
	var sessionID string
	timeout := time.After(5 * time.Second)

	// Parse SSE events to extract session ID from the endpoint URL
	// SSEイベントを解析してエンドポイントURLからセッションIDを抽出
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for session ID")
		default:
			if scanner.Scan() {
				line := scanner.Text()
				// Look for the data line containing the sessionId parameter
				// sessionIdパラメータを含むdataラインを探す
				if strings.HasPrefix(line, "data:") {
					data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
					if strings.Contains(data, "sessionId=") {
						idx := strings.Index(data, "sessionId=")
						sessionID = data[idx+len("sessionId="):]
						goto gotSessionID
					}
				}
			}
		}
	}
gotSessionID:

	if sessionID == "" {
		t.Fatal("Failed to get session ID")
	}

	// Step 2: Initialize the session before calling tools
	// MCP requires initialization before any other method can be called
	// ステップ2：ツール呼び出しの前にセッションを初期化
	// MCPでは他のメソッドを呼び出す前に初期化が必要
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	initBody, _ := json.Marshal(initReq)

	// Send initialize request to establish the session
	// セッションを確立するためにinitializeリクエストを送信
	initResp, err := http.Post(
		ts.URL+"/message?sessionId="+sessionID,
		"application/json",
		bytes.NewReader(initBody),
	)
	if err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}
	initResp.Body.Close()

	// Wait for initialize response to be processed via SSE
	// SSE経由でinitializeレスポンスが処理されるのを待つ
	time.Sleep(100 * time.Millisecond)

	// Step 3: Call an unknown tool - this should return error via SSE
	// This tests the error handling path for invalid tool names
	// ステップ3：不明なツールを呼び出す - SSE経由でエラーが返されるはず
	// これは無効なツール名に対するエラー処理パスをテストする
	unknownToolReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "unknown_tool_that_does_not_exist",
			"arguments": map[string]interface{}{},
		},
	}
	unknownBody, _ := json.Marshal(unknownToolReq)

	// Send the unknown tool request
	// 不明なツールリクエストを送信
	toolResp, err := http.Post(
		ts.URL+"/message?sessionId="+sessionID,
		"application/json",
		bytes.NewReader(unknownBody),
	)
	if err != nil {
		t.Fatalf("Failed to send unknown tool request: %v", err)
	}
	defer toolResp.Body.Close()

	// The POST should return 202 Accepted because the actual error is sent via SSE
	// 実際のエラーはSSE経由で送信されるため、POSTは202 Acceptedを返すべき
	if toolResp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", toolResp.StatusCode)
	}

	// Read SSE messages to find the error response
	// This verifies that errors are correctly routed through SSE
	// SSEメッセージを読み取ってエラーレスポンスを見つける
	// これはエラーがSSE経由で正しくルーティングされることを検証する
	errorReceived := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "" {
					continue
				}

				// Parse the JSON-RPC response from the SSE data
				// SSEデータからJSON-RPCレスポンスを解析
				var resp JSONRPCResponse
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					continue
				}

				// Check if this is our error response for the unknown tool
				// これが不明なツールに対するエラーレスポンスかどうかを確認
				if resp.Error != nil && strings.Contains(resp.Error.Message, "unknown tool") {
					errorReceived <- true
					return
				}
			}
		}
	}()

	// Wait for the error response or timeout
	// エラーレスポンスを待つか、タイムアウト
	select {
	case <-errorReceived:
		// Success - error was received via SSE as expected
		// 成功 - 期待通りSSE経由でエラーを受信
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for error response via SSE")
	}
}

// TestIsAllowedOrigin tests the origin validation helper function.
// This function validates that requests come from localhost origins only,
// preventing DNS rebinding attacks.
//
// TestIsAllowedOriginはオリジン検証ヘルパー関数をテストします。
// この関数はリクエストがlocalhostオリジンからのみ来ることを検証し、
// DNSリバインディング攻撃を防ぎます。
func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		name   string // Test case name / テストケース名
		origin string // Origin to test / テストするオリジン
		want   bool   // Expected result / 期待される結果
	}{
		// Allowed localhost origins / 許可されるlocalhostオリジン
		{"localhost http", "http://localhost", true},
		{"localhost https", "https://localhost", true},
		{"localhost with port", "http://localhost:3000", true},
		{"localhost https with port", "https://localhost:8080", true},

		// Allowed 127.0.0.1 origins / 許可される127.0.0.1オリジン
		{"127.0.0.1 http", "http://127.0.0.1", true},
		{"127.0.0.1 https", "https://127.0.0.1", true},
		{"127.0.0.1 with port", "http://127.0.0.1:8080", true},
		{"127.0.0.1 https with port", "https://127.0.0.1:443", true},

		// Allowed IPv6 localhost origins / 許可されるIPv6 localhostオリジン
		{"ipv6 localhost http", "http://[::1]", true},
		{"ipv6 localhost https", "https://[::1]", true},
		{"ipv6 localhost with port", "http://[::1]:8080", true},
		{"ipv6 localhost https with port", "https://[::1]:443", true},

		// Blocked external origins / ブロックされる外部オリジン
		{"external domain", "http://example.com", false},
		{"external https", "https://evil.com", false},
		{"external with port", "http://attacker.com:8080", false},
		{"external ip", "http://192.168.1.1", false},
		{"external ip with port", "http://10.0.0.1:3000", false},

		// Blocked subdomain attempts / ブロックされるサブドメイン試行
		{"localhost subdomain attack", "http://localhost.evil.com", false},
		{"localhost prefix attack", "http://localhost-evil.com", false},
		{"127 prefix attack", "http://127.0.0.1.evil.com", false},

		// Edge cases / エッジケース
		{"empty origin", "", false},
		{"just localhost", "localhost", false},
		{"missing protocol", "//localhost:8080", false},
		{"file protocol", "file://localhost", false},
		{"ftp protocol", "ftp://localhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedOrigin(tt.origin)
			if got != tt.want {
				t.Errorf("isAllowedOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

// TestCORSMiddleware tests that CORS headers are set correctly based on origin.
// The middleware should only set Access-Control-Allow-Origin for allowed origins.
//
// TestCORSMiddlewareはオリジンに基づいてCORSヘッダーが正しく設定されることをテストします。
// ミドルウェアは許可されたオリジンに対してのみAccess-Control-Allow-Originを設定すべきです。
func TestCORSMiddleware(t *testing.T) {
	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 0)

	// Create test handler that records if it was called
	// 呼び出されたかどうかを記録するテストハンドラを作成
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	// CORSミドルウェアでラップ
	handler := server.corsMiddleware(testHandler)

	tests := []struct {
		name              string // Test case name / テストケース名
		origin            string // Origin header / Originヘッダー
		expectCORSHeader  bool   // Should CORS header be set / CORSヘッダーが設定されるべきか
		expectOriginValue string // Expected origin in header / ヘッダーに期待されるオリジン値
	}{
		{
			name:              "allowed localhost origin",
			origin:            "http://localhost:3000",
			expectCORSHeader:  true,
			expectOriginValue: "http://localhost:3000",
		},
		{
			name:              "allowed 127.0.0.1 origin",
			origin:            "http://127.0.0.1:8080",
			expectCORSHeader:  true,
			expectOriginValue: "http://127.0.0.1:8080",
		},
		{
			name:              "blocked external origin",
			origin:            "http://evil.com",
			expectCORSHeader:  false,
			expectOriginValue: "",
		},
		{
			name:              "no origin header",
			origin:            "",
			expectCORSHeader:  false,
			expectOriginValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			corsHeader := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORSHeader {
				if corsHeader != tt.expectOriginValue {
					t.Errorf("Expected CORS header %q, got %q", tt.expectOriginValue, corsHeader)
				}
			} else {
				if corsHeader != "" {
					t.Errorf("Expected no CORS header, got %q", corsHeader)
				}
			}
		})
	}
}

// =============================================================================
// Logging Tests / ログテスト
// =============================================================================

// logCapture is a helper to capture slog output for testing.
// logCaptureはテスト用にslog出力をキャプチャするヘルパーです。
type logCapture struct {
	buf    *bytes.Buffer
	logger *slog.Logger
}

// newLogCapture creates a new log capture helper with the specified level.
// newLogCaptureは指定されたレベルで新しいログキャプチャヘルパーを作成します。
func newLogCapture(level slog.Level) *logCapture {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	return &logCapture{buf: buf, logger: logger}
}

// install sets this logger as the default and returns a restore function.
// installはこのロガーをデフォルトに設定し、復元関数を返します。
func (lc *logCapture) install() func() {
	original := slog.Default()
	slog.SetDefault(lc.logger)
	return func() { slog.SetDefault(original) }
}

// String returns the captured log output.
// Stringはキャプチャされたログ出力を返します。
func (lc *logCapture) String() string {
	return lc.buf.String()
}

// TestDisconnectLoggingIntegration tests the actual disconnect logging behavior
// by connecting to the SSE endpoint and closing the connection.
// This is an integration test that verifies the real handleSSE behavior.
//
// TestDisconnectLoggingIntegrationは、SSEエンドポイントに接続して
// 接続を閉じることで実際の切断ログ動作をテストします。
// これは実際のhandleSSE動作を検証する統合テストです。
func TestDisconnectLoggingIntegration(t *testing.T) {
	tests := []struct {
		name            string
		verbosity       int
		doInitialize    bool
		clientName      string
		expectLog       bool
		expectLogSubstr string
	}{
		{
			name:         "uninitialized_verbosity_0_no_log",
			verbosity:    0,
			doInitialize: false,
			expectLog:    false,
		},
		{
			name:            "uninitialized_verbosity_3_shows_log",
			verbosity:       3,
			doInitialize:    false,
			expectLog:       true,
			expectLogSubstr: "(not initialized)",
		},
		{
			name:            "initialized_claude_code_shows_info",
			verbosity:       0,
			doInitialize:    true,
			clientName:      "claude-code",
			expectLog:       true,
			expectLogSubstr: "claude-code",
		},
		{
			name:            "initialized_dkmcp_client_shows_debug",
			verbosity:       2, // Need DEBUG level to see dkmcp-go-client logs
			doInitialize:    true,
			clientName:      "dkmcp-go-client",
			expectLog:       true,
			expectLogSubstr: "dkmcp-go-client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture logs at DEBUG level to see all logs
			// すべてのログを見るためにDEBUGレベルでキャプチャ
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			// Create server with specified verbosity
			// 指定されたverbosityでサーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(tt.verbosity))

			// Create test HTTP server
			// テストHTTPサーバーを作成
			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			mux.HandleFunc("POST /message", server.handleMessage)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			// Connect to SSE endpoint
			// SSEエンドポイントに接続
			sseReq, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
			sseReq.Header.Set("Accept", "text/event-stream")

			sseResp, err := http.DefaultClient.Do(sseReq)
			if err != nil {
				t.Fatalf("Failed to connect to SSE: %v", err)
			}

			// Read the endpoint event to get session ID
			// セッションIDを取得するためにendpointイベントを読み取る
			scanner := bufio.NewScanner(sseResp.Body)
			var sessionID string
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: /message?sessionId=") {
					sessionID = strings.TrimPrefix(line, "data: /message?sessionId=")
					break
				}
			}

			if sessionID == "" {
				sseResp.Body.Close()
				t.Fatal("Failed to get session ID")
			}

			// If we need to initialize, send initialize request
			// 初期化が必要な場合、initializeリクエストを送信
			if tt.doInitialize {
				initReq := JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      0,
					Method:  "initialize",
					Params: map[string]any{
						"clientInfo": map[string]string{
							"name":    tt.clientName,
							"version": "1.0.0",
						},
					},
				}
				initBody, _ := json.Marshal(initReq)

				initResp, err := http.Post(
					ts.URL+"/message?sessionId="+sessionID,
					"application/json",
					bytes.NewReader(initBody),
				)
				if err != nil {
					sseResp.Body.Close()
					t.Fatalf("Failed to send initialize: %v", err)
				}
				initResp.Body.Close()

				// Give time for initialization to complete
				// 初期化が完了するまで待機
				time.Sleep(50 * time.Millisecond)
			}

			// Close the SSE connection to trigger disconnect logging
			// 切断ログをトリガーするためにSSE接続を閉じる
			sseResp.Body.Close()

			// Give time for disconnect logging to happen
			// 切断ログが発生するまで待機
			time.Sleep(100 * time.Millisecond)

			// Check results
			// 結果を確認
			logOutput := capture.String()
			hasDisconnectLog := strings.Contains(logOutput, "Client disconnected")

			if tt.expectLog && !hasDisconnectLog {
				t.Errorf("Expected disconnect log but got none.\nLog output:\n%s", logOutput)
			}
			if !tt.expectLog && hasDisconnectLog {
				t.Errorf("Expected no disconnect log but got one.\nLog output:\n%s", logOutput)
			}
			if tt.expectLog && tt.expectLogSubstr != "" {
				if !strings.Contains(logOutput, tt.expectLogSubstr) {
					t.Errorf("Expected %q in log but not found.\nLog output:\n%s", tt.expectLogSubstr, logOutput)
				}
			}
		})
	}
}

// TestInitializationLoggingIntegration tests the actual initialization logging
// by sending initialize requests through the real handler.
// For dkmcp-go-client, we verify there are NO INFO level logs (only DEBUG).
// For other clients, we verify there IS an INFO level log.
//
// TestInitializationLoggingIntegrationは、実際のハンドラを通じて
// initializeリクエストを送信することで実際の初期化ログをテストします。
// dkmcp-go-clientの場合、INFOレベルのログがないこと（DEBUGのみ）を検証します。
// 他のクライアントの場合、INFOレベルのログがあることを検証します。
func TestInitializationLoggingIntegration(t *testing.T) {
	tests := []struct {
		name           string
		clientName     string
		expectLogLevel string
		expectNoInfo   bool // For dkmcp-go-client, expect NO INFO level logs
	}{
		{
			name:           "dkmcp_client_debug",
			clientName:     "dkmcp-go-client",
			expectLogLevel: "DEBUG",
			expectNoInfo:   true, // dkmcp-go-client should have NO INFO logs
		},
		{
			name:           "claude_code_info",
			clientName:     "claude-code",
			expectLogLevel: "INFO",
			expectNoInfo:   false,
		},
		{
			name:           "other_client_info",
			clientName:     "some-other-client",
			expectLogLevel: "INFO",
			expectNoInfo:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture logs at DEBUG level
			// DEBUGレベルでログをキャプチャ
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			// Create server
			// サーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0)

			// Create test HTTP server
			// テストHTTPサーバーを作成
			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			mux.HandleFunc("POST /message", server.handleMessage)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			// Connect to SSE and get session ID
			// SSEに接続してセッションIDを取得
			sseReq, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
			sseReq.Header.Set("Accept", "text/event-stream")

			sseResp, err := http.DefaultClient.Do(sseReq)
			if err != nil {
				t.Fatalf("Failed to connect to SSE: %v", err)
			}
			defer sseResp.Body.Close()

			scanner := bufio.NewScanner(sseResp.Body)
			var sessionID string
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: /message?sessionId=") {
					sessionID = strings.TrimPrefix(line, "data: /message?sessionId=")
					break
				}
			}

			if sessionID == "" {
				t.Fatal("Failed to get session ID")
			}

			// Clear the log buffer before initialize
			// initialize前にログバッファをクリア
			capture.buf.Reset()

			// Send initialize request
			// initializeリクエストを送信
			initReq := JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      0,
				Method:  "initialize",
				Params: map[string]any{
					"clientInfo": map[string]string{
						"name":    tt.clientName,
						"version": "1.0.0",
					},
				},
			}
			initBody, _ := json.Marshal(initReq)

			initResp, err := http.Post(
				ts.URL+"/message?sessionId="+sessionID,
				"application/json",
				bytes.NewReader(initBody),
			)
			if err != nil {
				t.Fatalf("Failed to send initialize: %v", err)
			}
			initResp.Body.Close()

			// Give time for logging
			// ログ出力を待機
			time.Sleep(50 * time.Millisecond)

			// Check results
			// 結果を確認
			logOutput := capture.String()

			// Check that initialization log exists with correct client name
			// 初期化ログが正しいクライアント名で存在することを確認
			if !strings.Contains(logOutput, tt.clientName) {
				t.Errorf("Expected client_name %s in log but got:\n%s", tt.clientName, logOutput)
			}

			// For dkmcp-go-client, verify there are NO INFO level logs containing the client name
			// This is the key test: dkmcp-go-client initialization should only log at DEBUG level
			// dkmcp-go-clientの場合、クライアント名を含むINFOレベルのログがないことを検証
			// これが重要なテスト: dkmcp-go-clientの初期化はDEBUGレベルでのみログ出力すべき
			if tt.expectNoInfo {
				// Check for INFO logs that contain the client name (initialization-related logs)
				// クライアント名を含むINFOログ（初期化関連ログ）をチェック
				lines := strings.Split(logOutput, "\n")
				for _, line := range lines {
					if strings.Contains(line, "level=INFO") && strings.Contains(line, tt.clientName) {
						t.Errorf("Expected NO INFO level logs containing %s but found:\n%s", tt.clientName, line)
					}
				}
				// Verify DEBUG log exists for client initialization
				// クライアント初期化のDEBUGログが存在することを確認
				if !strings.Contains(logOutput, "level=DEBUG") || !strings.Contains(logOutput, "Client connected") {
					t.Errorf("Expected DEBUG level 'Client connected' log for %s but got:\n%s", tt.clientName, logOutput)
				}
			} else {
				// For other clients, verify there IS an INFO level log with client name
				// 他のクライアントの場合、クライアント名を含むINFOレベルのログがあることを検証
				foundClientInfo := false
				lines := strings.Split(logOutput, "\n")
				for _, line := range lines {
					if strings.Contains(line, "level=INFO") && strings.Contains(line, tt.clientName) {
						foundClientInfo = true
						break
					}
				}
				if !foundClientInfo {
					t.Errorf("Expected INFO level log containing %s but got:\n%s", tt.clientName, logOutput)
				}
			}
		})
	}
}

// TestVerboseLogContainsClientInfo tests that verbose request/response logs
// include the client name. This tests the logVerboseRequest and logVerboseResponse
// functions directly since they are simple output functions.
//
// TestVerboseLogContainsClientInfoは詳細リクエスト/レスポンスログに
// クライアント名が含まれることをテストします。これはlogVerboseRequestと
// logVerboseResponse関数を直接テストします（単純な出力関数のため）。
func TestVerboseLogContainsClientInfo(t *testing.T) {
	tests := []struct {
		name             string
		clientName       string
		userAgent        string
		initialized      bool
		expectedName     string // expected client_name value in log
		expectedAgent    string // expected user_agent value in log
	}{
		{
			name:          "initialized_client",
			clientName:    "claude-code",
			userAgent:     "claude-code/2.1.7",
			initialized:   true,
			expectedName:  "claude-code",
			expectedAgent: "claude-code/2.1.7",
		},
		{
			name:          "uninitialized_client",
			clientName:    "",
			userAgent:     "Go-http-client/1.1",
			initialized:   false,
			expectedName:  "(not initialized)",
			expectedAgent: "Go-http-client/1.1",
		},
		{
			name:          "dkmcp_client",
			clientName:    "dkmcp-go-client",
			userAgent:     "Go-http-client/1.1",
			initialized:   true,
			expectedName:  "dkmcp-go-client",
			expectedAgent: "Go-http-client/1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_request", func(t *testing.T) {
			// Capture logs
			// ログをキャプチャ
			capture := newLogCapture(slog.LevelInfo)
			restore := capture.install()
			defer restore()

			// Create server with verbosity enabled
			// verbosity有効でサーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(1))

			// Create mock client
			// モッククライアントを作成
			mockClient := &client{
				id:          "test-client-789",
				clientName:  tt.clientName,
				userAgent:   tt.userAgent,
				initialized: tt.initialized,
				remoteAddr:  "127.0.0.1:54321",
			}

			// Call logVerboseRequest
			// logVerboseRequestを呼び出す
			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params: map[string]any{
					"name": "list_containers",
				},
			}
			// Marshal request to simulate raw JSON bytes from client
			// クライアントからの生JSONバイトをシミュレートするためにリクエストをmarshal
			rawJSON, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Failed to marshal test request: %v", err)
			}
			server.logVerboseRequest(mockClient, req, rawJSON, 1)

			// Check that client_name and user_agent are in the log as separate fields
			// client_nameとuser_agentが別々のフィールドとしてログに含まれることを確認
			logOutput := capture.String()
			if !containsLogField(logOutput, "client_name", tt.expectedName) {
				t.Errorf("Expected client_name=%s in log but got:\n%s", tt.expectedName, logOutput)
			}
			if !containsLogField(logOutput, "user_agent", tt.expectedAgent) {
				t.Errorf("Expected user_agent=%s in log but got:\n%s", tt.expectedAgent, logOutput)
			}
			if !strings.Contains(logOutput, "REQUEST") {
				t.Errorf("Expected 'REQUEST' in log but got:\n%s", logOutput)
			}
		})

		t.Run(tt.name+"_response", func(t *testing.T) {
			// Capture logs
			// ログをキャプチャ
			capture := newLogCapture(slog.LevelInfo)
			restore := capture.install()
			defer restore()

			// Create server with verbosity enabled
			// verbosity有効でサーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(1))

			// Create mock client
			// モッククライアントを作成
			mockClient := &client{
				id:          "test-client-789",
				clientName:  tt.clientName,
				userAgent:   tt.userAgent,
				initialized: tt.initialized,
				remoteAddr:  "127.0.0.1:54321",
			}

			// Call logVerboseResponse
			// logVerboseResponseを呼び出す
			resp := &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]any{"status": "ok"},
			}
			server.logVerboseResponse(mockClient, resp, 1)

			// Check that client_name and user_agent are in the log as separate fields
			// client_nameとuser_agentが別々のフィールドとしてログに含まれることを確認
			logOutput := capture.String()
			if !containsLogField(logOutput, "client_name", tt.expectedName) {
				t.Errorf("Expected client_name=%s in log but got:\n%s", tt.expectedName, logOutput)
			}
			if !containsLogField(logOutput, "user_agent", tt.expectedAgent) {
				t.Errorf("Expected user_agent=%s in log but got:\n%s", tt.expectedAgent, logOutput)
			}
			if !strings.Contains(logOutput, "RESPONSE") {
				t.Errorf("Expected 'RESPONSE' in log but got:\n%s", logOutput)
			}
		})
	}
}

// TestSSERequestFiltering tests that SSE requests are filtered based on verbosity.
// At verbosity < 3, SSE requests should NOT be logged at INFO level (they're noise).
// At verbosity >= 3, SSE requests should be logged.
//
// TestSSERequestFilteringはSSEリクエストがverbosityに基づいてフィルタリングされることをテストします。
// verbosity < 3の場合、SSEリクエストはINFOレベルでログ出力されるべきではありません（ノイズ）。
// verbosity >= 3の場合、SSEリクエストはログ出力されるべきです。
func TestSSERequestFiltering(t *testing.T) {
	tests := []struct {
		name           string
		verbosity      int
		expectSSEInLog bool // true if SSE "Request received" should appear in logs
	}{
		{
			name:           "verbosity_0_filters_sse",
			verbosity:      0,
			expectSSEInLog: false, // SSE requests should NOT appear at verbosity 0
		},
		{
			name:           "verbosity_1_filters_sse",
			verbosity:      1,
			expectSSEInLog: false, // SSE requests should NOT appear at verbosity 1
		},
		{
			name:           "verbosity_2_filters_sse",
			verbosity:      2,
			expectSSEInLog: false, // SSE requests should NOT appear at verbosity 2
		},
		{
			name:           "verbosity_3_shows_sse",
			verbosity:      3,
			expectSSEInLog: true, // SSE requests should appear at verbosity 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture logs at DEBUG level to see all logs
			// 全ログを見るためにDEBUGレベルでログをキャプチャ
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			// Create server with the test verbosity
			// テストのverbosityでサーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(tt.verbosity))

			// Create test HTTP server with logging middleware
			// ロギングミドルウェアを使用してテストHTTPサーバーを作成
			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			ts := httptest.NewServer(server.loggingMiddleware(mux))
			defer ts.Close()

			// Connect to SSE and immediately close
			// SSEに接続してすぐに閉じる
			sseReq, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
			sseReq.Header.Set("Accept", "text/event-stream")

			client := &http.Client{Timeout: 100 * time.Millisecond}
			resp, err := client.Do(sseReq)
			if err == nil {
				resp.Body.Close()
			}

			// Wait for logs to be written
			// ログが書き込まれるのを待つ
			time.Sleep(50 * time.Millisecond)

			// Check log output
			// ログ出力を確認
			logOutput := capture.String()

			// Look for SSE request in logs
			// ログ内のSSEリクエストを探す
			hasSSERequest := strings.Contains(logOutput, "path=/sse") && strings.Contains(logOutput, "Request received")

			if tt.expectSSEInLog && !hasSSERequest {
				t.Errorf("Expected SSE request in log at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}

			if !tt.expectSSEInLog && hasSSERequest {
				t.Errorf("Expected NO SSE request in log at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}
		})
	}
}

// TestClientLogAttrs tests the clientLogAttrs helper function.
// It verifies that client_name and user_agent are returned as separate fields:
// - Initialized with name: raw client_name + user_agent
// - Initialized without name: "(empty name)" + user_agent
// - Not initialized: "(not initialized)" + user_agent
//
// TestClientLogAttrsはclientLogAttrsヘルパー関数をテストします。
// client_nameとuser_agentが別々のフィールドとして返されることを検証します：
// - 名前付きで初期化済み：生のclient_name + user_agent
// - 名前なしで初期化済み："(empty name)" + user_agent
// - 未初期化："(not initialized)" + user_agent
func TestClientLogAttrs(t *testing.T) {
	tests := []struct {
		name           string
		c              *client
		wantClientName string
		wantUserAgent  string
	}{
		{
			name:           "initialized_with_name",
			c:              &client{clientName: "claude-code", initialized: true, userAgent: "claude-code/2.1.7"},
			wantClientName: "claude-code",
			wantUserAgent:  "claude-code/2.1.7",
		},
		{
			name:           "initialized_empty_name",
			c:              &client{clientName: "", initialized: true, userAgent: "Go-http-client/1.1"},
			wantClientName: "(empty name)",
			wantUserAgent:  "Go-http-client/1.1",
		},
		{
			name:           "not_initialized",
			c:              &client{clientName: "", initialized: false, userAgent: "Go-http-client/1.1"},
			wantClientName: "(not initialized)",
			wantUserAgent:  "Go-http-client/1.1",
		},
		{
			name:           "not_initialized_no_user_agent",
			c:              &client{clientName: "", initialized: false, userAgent: ""},
			wantClientName: "(not initialized)",
			wantUserAgent:  "",
		},
		{
			name:           "initialized_with_suffix",
			c:              &client{clientName: "dkmcp-go-client_logs", initialized: true, userAgent: "Go-http-client/1.1"},
			wantClientName: "dkmcp-go-client_logs",
			wantUserAgent:  "Go-http-client/1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := clientLogAttrs(tt.c)
			if len(attrs) != 4 {
				t.Fatalf("expected 4 elements (2 key-value pairs), got %d: %v", len(attrs), attrs)
			}
			if attrs[0] != "client_name" || attrs[1] != tt.wantClientName {
				t.Errorf("client_name: got %v=%v, want %v", attrs[0], attrs[1], tt.wantClientName)
			}
			if attrs[2] != "user_agent" || attrs[3] != tt.wantUserAgent {
				t.Errorf("user_agent: got %v=%v, want %v", attrs[2], attrs[3], tt.wantUserAgent)
			}
		})
	}
}

// TestHTTPHeaderLogging tests that HTTP headers are logged at verbosity >= 4.
// At lower verbosity levels, headers should not appear in the logs.
//
// TestHTTPHeaderLoggingはverbosity >= 4でHTTPヘッダーがログ出力されることをテストします。
// より低いverbosityレベルではヘッダーはログに出力されるべきではありません。
func TestHTTPHeaderLogging(t *testing.T) {
	tests := []struct {
		name             string
		verbosity        int
		expectHeadersLog bool
	}{
		{
			name:             "verbosity_3_no_headers",
			verbosity:        3,
			expectHeadersLog: false,
		},
		{
			name:             "verbosity_4_shows_headers",
			verbosity:        4,
			expectHeadersLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture logs at DEBUG level
			// DEBUGレベルでログをキャプチャ
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			// Create server with the test verbosity
			// テストのverbosityでサーバーを作成
			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(tt.verbosity))

			// Create a simple handler
			// シンプルなハンドラを作成
			handler := server.loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create a request with some headers
			// ヘッダー付きのリクエストを作成
			req := httptest.NewRequest("POST", "/message", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-Header", "test-value")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Check log output
			// ログ出力を確認
			logOutput := capture.String()
			hasHeaderLog := strings.Contains(logOutput, "HTTP header")

			if tt.expectHeadersLog && !hasHeaderLog {
				t.Errorf("Expected HTTP header log at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}
			if !tt.expectHeadersLog && hasHeaderLog {
				t.Errorf("Expected NO HTTP header log at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}

			// At verbosity 4, verify specific headers are present
			// verbosity 4では特定のヘッダーが存在することを確認
			if tt.expectHeadersLog {
				if !strings.Contains(logOutput, "X-Test-Header") {
					t.Errorf("Expected X-Test-Header in log but got:\n%s", logOutput)
				}
				if !strings.Contains(logOutput, "test-value") {
					t.Errorf("Expected test-value in log but got:\n%s", logOutput)
				}
			}
		})
	}
}

// =============================================================================
// Integration Tests for separate client fields / クライアントフィールド分離 連携テスト
// =============================================================================

// sseTestHelper connects to SSE, extracts session ID, and optionally initializes.
// SSEに接続してセッションIDを取得し、必要に応じて初期化するテストヘルパー。
type sseTestHelper struct {
	t         *testing.T
	ts        *httptest.Server
	sseResp   *http.Response
	sessionID string
}

// connectSSE connects to the SSE endpoint with a custom User-Agent header.
// Returns the helper for chaining. Caller must call close() when done.
//
// カスタムUser-Agentヘッダーを指定してSSEエンドポイントに接続します。
// 呼び出し元は完了時にclose()を呼ぶ必要があります。
func newSSETestHelper(t *testing.T, ts *httptest.Server, userAgent string) *sseTestHelper {
	t.Helper()

	sseReq, err := http.NewRequest("GET", ts.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("Failed to create SSE request: %v", err)
	}
	if userAgent != "" {
		sseReq.Header.Set("User-Agent", userAgent)
	}

	sseResp, err := (&http.Client{}).Do(sseReq)
	if err != nil {
		t.Fatalf("Failed to connect to SSE: %v", err)
	}

	// Read the endpoint event to extract session ID
	// endpointイベントからセッションIDを抽出
	scanner := bufio.NewScanner(sseResp.Body)
	var sessionID string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: /message?sessionId=") {
			sessionID = strings.TrimPrefix(line, "data: /message?sessionId=")
			break
		}
	}
	if sessionID == "" {
		sseResp.Body.Close()
		t.Fatal("Failed to get session ID from SSE endpoint")
	}

	return &sseTestHelper{t: t, ts: ts, sseResp: sseResp, sessionID: sessionID}
}

// initialize sends an initialize request with the specified client name and version.
// 指定されたクライアント名とバージョンでinitializeリクエストを送信します。
func (h *sseTestHelper) initialize(clientName, clientVersion string) {
	h.t.Helper()

	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]any{
			"clientInfo": map[string]string{
				"name":    clientName,
				"version": clientVersion,
			},
		},
	}
	initBody, _ := json.Marshal(initReq)

	resp, err := http.Post(
		h.ts.URL+"/message?sessionId="+h.sessionID,
		"application/json",
		bytes.NewReader(initBody),
	)
	if err != nil {
		h.t.Fatalf("Failed to send initialize request: %v", err)
	}
	resp.Body.Close()

	// Wait for processing
	// 処理完了を待機
	time.Sleep(50 * time.Millisecond)
}

// close releases resources.
// リソースを解放します。
func (h *sseTestHelper) close() {
	if h.sseResp != nil {
		h.sseResp.Body.Close()
	}
}

// TestResolveDisplayNameInitIntegration tests the full SSE + initialize flow
// to verify that client_name in the "[+] Client connected" log shows the
// resolved display name based on User-Agent and client_name relationship.
//
// SSE接続 → initialize の一連のフローを通じて、
// "[+] Client connected" ログの client_name に、User-Agent と client_name の
// 関係に基づく表示名が正しく表示されることを検証する連携テスト。
func TestSeparateClientFieldsInitIntegration(t *testing.T) {
	tests := []struct {
		name           string
		userAgent      string // User-Agent header on SSE connection
		clientName     string // client_name from MCP initialize
		wantClientName string // expected client_name field value
		wantUserAgent  string // expected user_agent field value
	}{
		{
			name:           "user_agent_contains_client_name",
			userAgent:      "claude-code/2.1.7",
			clientName:     "claude-code",
			wantClientName: "claude-code",
			wantUserAgent:  "claude-code/2.1.7",
		},
		{
			name:           "user_agent_does_not_contain_client_name",
			userAgent:      "Go-http-client/1.1",
			clientName:     "dkmcp-go-client",
			wantClientName: "dkmcp-go-client",
			wantUserAgent:  "Go-http-client/1.1",
		},
		{
			name:           "client_with_suffix_not_in_user_agent",
			userAgent:      "Go-http-client/1.1",
			clientName:     "dkmcp-go-client_logs",
			wantClientName: "dkmcp-go-client_logs",
			wantUserAgent:  "Go-http-client/1.1",
		},
		{
			name:           "exact_match_user_agent_equals_client_name",
			userAgent:      "my-custom-tool",
			clientName:     "my-custom-tool",
			wantClientName: "my-custom-tool",
			wantUserAgent:  "my-custom-tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture logs at DEBUG level to see all logs including dkmcp-go-client
			// dkmcp-go-client のDEBUGログも含め、全ログをキャプチャ
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0)

			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			mux.HandleFunc("POST /message", server.handleMessage)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			// Clear log buffer before the test action
			// テスト実行前にログバッファをクリア
			capture.buf.Reset()

			// Connect SSE with custom User-Agent, then initialize
			// カスタムUser-AgentでSSE接続後、initializeを送信
			h := newSSETestHelper(t, ts, tt.userAgent)
			defer h.close()
			capture.buf.Reset() // Clear SSE connection noise / SSE接続ノイズをクリア
			h.initialize(tt.clientName, "1.0.0")

			logOutput := capture.String()

			// Verify "[+] Client connected" log contains separate client_name and user_agent
			// "[+] Client connected" ログに client_name と user_agent が別々に含まれることを検証
			connectedLines := filterLogLines(logOutput, "Client connected")
			if len(connectedLines) == 0 {
				t.Fatalf("Expected '[+] Client connected' log but found none.\nFull log:\n%s", logOutput)
			}

			connLine := connectedLines[0]
			if !strings.Contains(connLine, "client_name="+tt.wantClientName) {
				t.Errorf("Expected client_name=%s in connection log but got:\n%s", tt.wantClientName, connLine)
			}
			if !strings.Contains(connLine, "user_agent="+tt.wantUserAgent) {
				t.Errorf("Expected user_agent=%s in connection log but got:\n%s", tt.wantUserAgent, connLine)
			}
		})
	}
}

// TestSeparateClientFieldsDisconnectIntegration tests that the disconnect log
// shows client_name and user_agent as separate fields after a client initializes
// and disconnects.
//
// クライアントが初期化後に切断した場合、切断ログに client_name と user_agent が
// 別々のフィールドとして正しく表示されることを検証する連携テスト。
func TestSeparateClientFieldsDisconnectIntegration(t *testing.T) {
	tests := []struct {
		name           string
		userAgent      string
		clientName     string
		wantClientName string
		wantUserAgent  string
	}{
		{
			name:           "claude_code_disconnect",
			userAgent:      "claude-code/2.1.7",
			clientName:     "claude-code",
			wantClientName: "claude-code",
			wantUserAgent:  "claude-code/2.1.7",
		},
		{
			name:           "different_agent_disconnect",
			userAgent:      "custom-agent/3.0",
			clientName:     "my-tool",
			wantClientName: "my-tool",
			wantUserAgent:  "custom-agent/3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0)

			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			mux.HandleFunc("POST /message", server.handleMessage)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			// Connect, initialize, then disconnect
			// 接続 → 初期化 → 切断
			h := newSSETestHelper(t, ts, tt.userAgent)
			h.initialize(tt.clientName, "1.0.0")

			// Clear log buffer to isolate disconnect log
			// 切断ログだけを確認するためにバッファをクリア
			capture.buf.Reset()

			// Close SSE connection to trigger disconnect
			// SSE接続を閉じて切断をトリガー
			h.close()

			// Wait for disconnect log to be written
			// 切断ログが出力されるのを待機
			time.Sleep(150 * time.Millisecond)

			logOutput := capture.String()

			// Verify disconnect log contains separate client_name and user_agent
			// 切断ログに client_name と user_agent が別々に含まれることを検証
			disconnectLines := filterLogLines(logOutput, "Client disconnected")
			if len(disconnectLines) == 0 {
				t.Fatalf("Expected '[-] Client disconnected' log but found none.\nFull log:\n%s", logOutput)
			}

			discLine := disconnectLines[0]
			if !strings.Contains(discLine, "client_name="+tt.wantClientName) {
				t.Errorf("Expected client_name=%s in disconnect log but got:\n%s", tt.wantClientName, discLine)
			}
			if !strings.Contains(discLine, "user_agent="+tt.wantUserAgent) {
				t.Errorf("Expected user_agent=%s in disconnect log but got:\n%s", tt.wantUserAgent, discLine)
			}
		})
	}
}

// TestSeparateClientFieldsProcessRequestIntegration tests that client_name and
// user_agent appear as separate fields in subsequent JSON-RPC request logs
// (e.g., tools/list) after initialization.
//
// 初期化後のJSON-RPCリクエストログ（tools/listなど）に client_name と user_agent が
// 別々のフィールドとして表示されることを検証する連携テスト。
func TestSeparateClientFieldsProcessRequestIntegration(t *testing.T) {
	capture := newLogCapture(slog.LevelDebug)
	restore := capture.install()
	defer restore()

	dockerClient := &docker.Client{}
	server := NewServer(dockerClient, 0)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /sse", server.handleSSE)
	mux.HandleFunc("POST /message", server.handleMessage)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Connect with a User-Agent that contains the client name
	// client_name を含む User-Agent で接続
	h := newSSETestHelper(t, ts, "claude-code/2.1.7")
	defer h.close()
	h.initialize("claude-code", "2.1.7")

	// Clear log buffer before the tools/list call
	// tools/list 呼び出し前にバッファをクリア
	capture.buf.Reset()

	// Send tools/list request to trigger "Processing JSON-RPC request" log
	// "Processing JSON-RPC request" ログを出すために tools/list を送信
	toolsReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	toolsBody, _ := json.Marshal(toolsReq)
	resp, err := http.Post(
		ts.URL+"/message?sessionId="+h.sessionID,
		"application/json",
		bytes.NewReader(toolsBody),
	)
	if err != nil {
		t.Fatalf("Failed to send tools/list request: %v", err)
	}
	resp.Body.Close()
	time.Sleep(50 * time.Millisecond)

	logOutput := capture.String()

	// Verify "Processing JSON-RPC request" log has separate client_name and user_agent
	// "Processing JSON-RPC request" ログに client_name と user_agent が別々に含まれることを検証
	processingLines := filterLogLines(logOutput, "Processing JSON-RPC request")
	if len(processingLines) == 0 {
		t.Fatalf("Expected 'Processing JSON-RPC request' log but found none.\nFull log:\n%s", logOutput)
	}

	procLine := processingLines[0]
	if !strings.Contains(procLine, "client_name=claude-code") {
		t.Errorf("Expected client_name=claude-code in processing log but got:\n%s", procLine)
	}
	if !strings.Contains(procLine, "user_agent=claude-code/2.1.7") {
		t.Errorf("Expected user_agent=claude-code/2.1.7 in processing log but got:\n%s", procLine)
	}
}

// TestHTTPHeaderLoggingIntegration tests the full HTTP flow for header logging
// at verbosity >= 4. This connects to the actual server endpoints with custom
// headers and verifies they appear in the logs.
//
// verbosity >= 4 で実際のHTTPフロー全体を通してヘッダーログが出力されることを
// 検証する連携テスト。カスタムヘッダー付きでサーバーエンドポイントに接続し、
// ログに出力されることを確認します。
func TestHTTPHeaderLoggingIntegration(t *testing.T) {
	tests := []struct {
		name             string
		verbosity        int
		expectHeadersLog bool
	}{
		{
			name:             "verbosity_3_no_headers_in_real_flow",
			verbosity:        3,
			expectHeadersLog: false,
		},
		{
			name:             "verbosity_4_headers_in_real_flow",
			verbosity:        4,
			expectHeadersLog: true,
		},
		{
			name:             "verbosity_5_headers_in_real_flow",
			verbosity:        5,
			expectHeadersLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newLogCapture(slog.LevelDebug)
			restore := capture.install()
			defer restore()

			dockerClient := &docker.Client{}
			server := NewServer(dockerClient, 0, WithVerbosity(tt.verbosity))

			// Set up server with full middleware chain (including loggingMiddleware)
			// ロギングミドルウェアを含む完全なミドルウェアチェーンでサーバーを構成
			mux := http.NewServeMux()
			mux.HandleFunc("GET /sse", server.handleSSE)
			mux.HandleFunc("POST /message", server.handleMessage)
			mux.HandleFunc("GET /health", server.handleHealth)
			ts := httptest.NewServer(server.loggingMiddleware(mux))
			defer ts.Close()

			// Send a health check request with custom headers
			// カスタムヘッダー付きでヘルスチェックリクエストを送信
			req, _ := http.NewRequest("GET", ts.URL+"/health", nil)
			req.Header.Set("X-Custom-Debug", "integration-test-value")
			req.Header.Set("X-Request-ID", "req-12345")

			capture.buf.Reset()

			resp, err := (&http.Client{}).Do(req)
			if err != nil {
				t.Fatalf("Failed to send health request: %v", err)
			}
			resp.Body.Close()

			logOutput := capture.String()
			hasHeaderLog := strings.Contains(logOutput, "HTTP header")

			if tt.expectHeadersLog && !hasHeaderLog {
				t.Errorf("Expected HTTP header logs at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}
			if !tt.expectHeadersLog && hasHeaderLog {
				t.Errorf("Expected NO HTTP header logs at verbosity %d but got:\n%s", tt.verbosity, logOutput)
			}

			if tt.expectHeadersLog {
				// Verify custom headers appear in the log
				// カスタムヘッダーがログに出力されていることを確認
				if !strings.Contains(logOutput, "X-Custom-Debug") {
					t.Errorf("Expected X-Custom-Debug header in log but got:\n%s", logOutput)
				}
				if !strings.Contains(logOutput, "integration-test-value") {
					t.Errorf("Expected 'integration-test-value' in log but got:\n%s", logOutput)
				}
				// Note: Go's http.Header normalizes names to canonical form
				// (e.g., "X-Request-ID" becomes "X-Request-Id")
				// Go の http.Header はヘッダー名を正規化する
				// （例: "X-Request-ID" → "X-Request-Id"）
				if !strings.Contains(logOutput, "X-Request-Id") {
					t.Errorf("Expected X-Request-Id header in log but got:\n%s", logOutput)
				}
				if !strings.Contains(logOutput, "req-12345") {
					t.Errorf("Expected 'req-12345' in log but got:\n%s", logOutput)
				}

				// Verify headers are sorted alphabetically
				// ヘッダーがアルファベット順にソートされていることを確認
				xCustomIdx := strings.Index(logOutput, "X-Custom-Debug")
				xRequestIdx := strings.Index(logOutput, "X-Request-Id")
				if xCustomIdx >= 0 && xRequestIdx >= 0 && xCustomIdx > xRequestIdx {
					t.Errorf("Expected headers to be sorted alphabetically (X-Custom-Debug before X-Request-ID)")
				}
			}
		})
	}
}

// filterLogLines returns log lines that contain the given substring.
// 指定された部分文字列を含むログ行を返すヘルパー関数。
func filterLogLines(logOutput, substr string) []string {
	var result []string
	for _, line := range strings.Split(logOutput, "\n") {
		if strings.Contains(line, substr) {
			result = append(result, line)
		}
	}
	return result
}

// containsLogField checks if logOutput contains a slog field key=value.
// The slog text handler may quote values with special characters (e.g., spaces, parens),
// so this checks for both key=value and key="value".
//
// containsLogFieldはlogOutput内にslogフィールド key=value が含まれるかを確認します。
// slogのテキストハンドラは特殊文字（スペース、括弧等）を含む値をクォートするため、
// key=value と key="value" の両方をチェックします。
func containsLogField(logOutput, key, value string) bool {
	return strings.Contains(logOutput, key+"="+value) ||
		strings.Contains(logOutput, key+`="`+value+`"`)
}

