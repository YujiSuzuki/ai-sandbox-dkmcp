// Package client provides an HTTP client for interacting with the DockMCP server.
// It implements the MCP (Model Context Protocol) over SSE (Server-Sent Events) and HTTP POST,
// allowing AI assistants to call tools on the DockMCP server such as listing containers,
// getting logs, executing commands, etc.
//
// clientパッケージはDockMCPサーバーと通信するためのHTTPクライアントを提供します。
// MCP（Model Context Protocol）をSSE（Server-Sent Events）とHTTP POSTで実装し、
// AIアシスタントがDockMCPサーバー上のツール（コンテナ一覧取得、ログ取得、
// コマンド実行など）を呼び出すことを可能にします。
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// clientVersion holds the version string for the client, used during MCP initialization.
// This is typically set at build time using ldflags.
//
// clientVersionはクライアントのバージョン文字列を保持し、MCP初期化時に使用されます。
// これは通常、ビルド時にldflagsを使用して設定されます。
var clientVersion string

// Client is an HTTP client for DockMCP server that manages SSE connections
// and JSON-RPC communication with the MCP server.
//
// The client maintains:
// - baseURL: The base URL of the DockMCP server (e.g., "http://localhost:8080")
// - httpClient: Standard HTTP client with timeout for regular requests
// - sseHTTPClient: HTTP client without timeout for long-lived SSE connections
// - sessionID: Unique identifier for the MCP session, obtained during SSE connection
// - sseConn: The active SSE connection response
// - messages: Channel for receiving parsed SSE messages
// - errors: Channel for receiving SSE connection errors
// - ctx/cancel: Context for managing client lifecycle and cancellation
// - mu: Mutex for thread-safe access to shared state
//
// ClientはDockMCPサーバー用のHTTPクライアントで、SSE接続と
// MCPサーバーとのJSON-RPC通信を管理します。
//
// クライアントは以下を維持します：
// - baseURL: DockMCPサーバーのベースURL（例："http://localhost:8080"）
// - httpClient: 通常リクエスト用のタイムアウト付き標準HTTPクライアント
// - sseHTTPClient: 長期接続のSSE用タイムアウトなしHTTPクライアント
// - sessionID: SSE接続時に取得するMCPセッションの一意識別子
// - sseConn: アクティブなSSE接続レスポンス
// - messages: 解析済みSSEメッセージを受信するチャネル
// - errors: SSE接続エラーを受信するチャネル
// - ctx/cancel: クライアントのライフサイクルとキャンセル管理用コンテキスト
// - mu: 共有状態へのスレッドセーフなアクセス用ミューテックス
type Client struct {
	baseURL       string
	httpClient    *http.Client
	sseHTTPClient *http.Client
	sessionID     string
	sseConn       *http.Response
	messages      chan []byte
	errors        chan error
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	clientSuffix  string // Suffix appended to client name / クライアント名に追加されるサフィックス
}

// NewClient creates a new DockMCP HTTP client configured to connect to the specified server.
// It initializes two HTTP clients: one with a 30-second timeout for regular requests,
// and another without timeout for SSE connections that need to stay open indefinitely.
//
// Parameters:
//   - baseURL: The base URL of the DockMCP server (e.g., "http://localhost:8080")
//
// Returns:
//   - A pointer to the newly created Client instance
//
// NewClientは指定されたサーバーに接続するための新しいDockMCP HTTPクライアントを作成します。
// 2つのHTTPクライアントを初期化します：通常リクエスト用の30秒タイムアウト付きクライアントと、
// 無期限に開いたままにする必要があるSSE接続用のタイムアウトなしクライアントです。
//
// パラメータ：
//   - baseURL: DockMCPサーバーのベースURL（例："http://localhost:8080"）
//
// 戻り値：
//   - 新しく作成されたClientインスタンスへのポインタ
func NewClient(baseURL string) *Client {
	// Create a cancellable context for managing the client's lifecycle
	// クライアントのライフサイクル管理用のキャンセル可能なコンテキストを作成
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		baseURL: baseURL,
		// Standard HTTP client with 30-second timeout for regular API calls
		// 通常のAPIコール用の30秒タイムアウト付き標準HTTPクライアント
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		// SSE client with no timeout to maintain long-lived connections
		// 長期接続を維持するためのタイムアウトなしSSEクライアント
		sseHTTPClient: &http.Client{
			Timeout: 0, // No timeout for SSE connections
		},
		// Buffered channel for SSE messages (capacity 10 to prevent blocking)
		// SSEメッセージ用のバッファ付きチャネル（ブロッキング防止のため容量10）
		messages: make(chan []byte, 10),
		// Error channel with capacity 1 for SSE connection errors
		// SSE接続エラー用の容量1のエラーチャネル
		errors: make(chan error, 1),
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetClientSuffix sets a suffix that will be appended to the client name.
// The resulting client name will be "dkmcp-go-client_<suffix>".
// This helps distinguish different callers (e.g., AI vs manual user).
//
// SetClientSuffixはクライアント名に追加されるサフィックスを設定します。
// 結果のクライアント名は"dkmcp-go-client_<suffix>"になります。
// これにより異なる呼び出し元（例：AIと手動ユーザー）を区別できます。
func (c *Client) SetClientSuffix(suffix string) {
	c.clientSuffix = suffix
}

// Connect establishes an SSE connection to the DockMCP server, retrieves the session ID,
// and performs the MCP initialization handshake.
//
// The connection process:
// 1. Sends GET request to /sse endpoint with appropriate headers
// 2. Reads the "endpoint" event from the SSE stream to get the session ID
// 3. Starts a background goroutine to continuously read SSE messages
// 4. Performs MCP initialize handshake with the server
//
// This method is idempotent - if already connected, it returns immediately without error.
//
// Returns:
//   - nil on successful connection and initialization
//   - error if connection or initialization fails
//
// ConnectはDockMCPサーバーへのSSE接続を確立し、セッションIDを取得し、
// MCP初期化ハンドシェイクを実行します。
//
// 接続プロセス：
// 1. 適切なヘッダー付きで/sseエンドポイントにGETリクエストを送信
// 2. SSEストリームから「endpoint」イベントを読み取ってセッションIDを取得
// 3. SSEメッセージを継続的に読み取るバックグラウンドgoroutineを開始
// 4. サーバーとMCP初期化ハンドシェイクを実行
//
// このメソッドは冪等です - 既に接続済みの場合、エラーなしで即座に戻ります。
//
// 戻り値：
//   - 接続と初期化が成功した場合はnil
//   - 接続または初期化が失敗した場合はerror
func (c *Client) Connect() error {
	c.mu.Lock()
	// Unlock is deferred to the end of the function
	// Unlockは関数の終わりまで遅延される

	// Check if already connected to avoid duplicate connections
	// 重複接続を避けるため、既に接続済みかチェック
	if c.sessionID != "" {
		c.mu.Unlock()
		return nil // Already connected / 既に接続済み
	}

	// Construct the SSE endpoint URL
	// SSEエンドポイントURLを構築
	url := fmt.Sprintf("%s/sse", c.baseURL)
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set SSE-specific headers for proper event stream handling
	// 適切なイベントストリーム処理のためのSSE固有ヘッダーを設定
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Initiate the SSE connection
	// SSE接続を開始
	resp, err := c.sseHTTPClient.Do(req)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}

	// Verify successful connection
	// 接続成功を確認
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		c.mu.Unlock()
		return fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
	}

	c.sseConn = resp

	// Read the endpoint event to get session ID from the SSE stream
	// SSEストリームからendpointイベントを読み取ってセッションIDを取得
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for "event: endpoint" line which signals session info
		// セッション情報を示す「event: endpoint」行を探す
		if strings.HasPrefix(line, "event:") {
			eventType := strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			if eventType == "endpoint" {
				// Read the next line which should be "data: /message?sessionId=..."
				// 次の行を読み取る（「data: /message?sessionId=...」のはず）
				if scanner.Scan() {
					dataLine := scanner.Text()
					if strings.HasPrefix(dataLine, "data:") {
						endpoint := strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))
						// Extract sessionId from endpoint URL (e.g., "/message?sessionId=client-123")
						// エンドポイントURLからsessionIdを抽出（例：「/message?sessionId=client-123」）
						if idx := strings.Index(endpoint, "sessionId="); idx != -1 {
							c.sessionID = endpoint[idx+len("sessionId="):]
							// Start background goroutine to read SSE messages
							// SSEメッセージを読み取るバックグラウンドgoroutineを開始
							go c.readSSEMessages()

							// Unlock before calling initialize, as it will lock again
							// initializeを呼ぶ前にアンロック（initializeは再度ロックするため）
							c.mu.Unlock()

							// Perform MCP initialization handshake
							// MCP初期化ハンドシェイクを実行
							return c.initialize()
						}
					}
				}
			}
		}
	}

	// Ensure unlock if loop fails without finding session ID
	// セッションIDが見つからずにループが失敗した場合のアンロック確保
	c.mu.Unlock()
	resp.Body.Close()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read SSE stream: %w", err)
	}

	return fmt.Errorf("failed to get session ID from SSE stream")
}

// initialize performs the MCP initialize handshake with the server.
// This is part of MCP and must be completed before any tool calls.
// It sends client information to the server and waits for acknowledgment.
//
// The initialize request includes:
// - Client name: "dkmcp-go-client"
// - Client version: Set via clientVersion variable
//
// Returns:
//   - nil on successful initialization
//   - error if the handshake fails or times out
//
// initializeはサーバーとのMCP初期化ハンドシェイクを実行します。
// これはMCPプロトコルの一部であり、ツール呼び出しの前に完了する必要があります。
// クライアント情報をサーバーに送信し、確認を待ちます。
//
// 初期化リクエストには以下が含まれます：
// - クライアント名："dkmcp-go-client"
// - クライアントバージョン：clientVersion変数で設定
//
// 戻り値：
//   - 初期化が成功した場合はnil
//   - ハンドシェイクが失敗またはタイムアウトした場合はerror
func (c *Client) initialize() error {
	// Build client name with optional suffix
	// オプションのサフィックスを含むクライアント名を構築
	clientName := "dkmcp-go-client"
	if c.clientSuffix != "" {
		clientName = clientName + "_" + c.clientSuffix
	}

	// Create the JSON-RPC initialize request following MCP
	// MCPプロトコルに従ったJSON-RPC初期化リクエストを作成
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      0, // ID for initialize is often 0 / initializeのIDは通常0
		Method:  "initialize",
		Params: map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    clientName,
				"version": clientVersion,
			},
		},
	}

	// Marshal the request to JSON
	// リクエストをJSONにマーシャル
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %w", err)
	}

	// Get session ID in a thread-safe manner
	// スレッドセーフな方法でセッションIDを取得
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()

	// Send the initialize request to the message endpoint
	// messageエンドポイントに初期化リクエストを送信
	url := fmt.Sprintf("%s/message?sessionId=%s", c.baseURL, sessionID)
	httpReq, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create initialize request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful acceptance (200 OK or 202 Accepted)
	// 正常な受付を確認（200 OKまたは202 Accepted）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error on initialize: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Wait for response from SSE channel with timeout handling
	// タイムアウト処理付きでSSEチャネルからのレスポンスを待機
	select {
	case msg := <-c.messages:
		// Parse the initialize response
		// 初期化レスポンスを解析
		var jsonrpcResp InitializeResponse
		if err := json.Unmarshal(msg, &jsonrpcResp); err != nil {
			return fmt.Errorf("failed to decode initialize response: %w", err)
		}
		// Check for JSON-RPC level errors
		// JSON-RPCレベルのエラーをチェック
		if jsonrpcResp.Error != nil {
			return fmt.Errorf("JSON-RPC error on initialize: %s (code %d)", jsonrpcResp.Error.Message, jsonrpcResp.Error.Code)
		}
		// Initialization successful
		// 初期化成功
		return nil

	case err := <-c.errors:
		return fmt.Errorf("SSE connection error during initialize: %w", err)
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for initialize response")
	case <-c.ctx.Done():
		return fmt.Errorf("client closed during initialize")
	}
}

// readSSEMessages continuously reads messages from the SSE stream in a background goroutine.
// It parses SSE "data:" lines and sends the parsed content to the messages channel.
// Any read errors are sent to the errors channel.
//
// This function runs until:
// - The SSE connection is closed
// - The client context is cancelled
// - A read error occurs
//
// readSSEMessagesはバックグラウンドgoroutineでSSEストリームからメッセージを継続的に読み取ります。
// SSEの「data:」行を解析し、解析されたコンテンツをmessagesチャネルに送信します。
// 読み取りエラーはerrorsチャネルに送信されます。
//
// この関数は以下の場合まで実行されます：
// - SSE接続が閉じられた
// - クライアントコンテキストがキャンセルされた
// - 読み取りエラーが発生した
func (c *Client) readSSEMessages() {
	// Ensure the SSE connection body is closed when this goroutine exits
	// このgoroutineが終了するときにSSE接続ボディを確実に閉じる
	defer c.sseConn.Body.Close()

	// Create a scanner to read the SSE stream line by line
	// SSEストリームを行ごとに読み取るスキャナーを作成
	scanner := bufio.NewScanner(c.sseConn.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE data lines start with "data:" prefix
		// SSEデータ行は「data:」プレフィックスで始まる
		if strings.HasPrefix(line, "data:") {
			// Extract the data content after the prefix
			// プレフィックスの後のデータ内容を抽出
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data != "" {
				// Send the message to the channel, respecting context cancellation
				// コンテキストキャンセルを尊重してメッセージをチャネルに送信
				select {
				case c.messages <- []byte(data):
				case <-c.ctx.Done():
					return
				}
			}
		}
	}

	// If there was a scanner error, send it to the errors channel
	// スキャナーエラーがあった場合、errorsチャネルに送信
	if err := scanner.Err(); err != nil {
		select {
		case c.errors <- err:
		case <-c.ctx.Done():
		}
	}
}

// Close closes the client connection and releases all resources.
// It cancels the client context (which stops the SSE reader goroutine)
// and closes the SSE connection if one exists.
//
// This method is safe to call multiple times.
//
// Returns:
//   - nil if closed successfully or already closed
//   - error if closing the SSE connection fails
//
// Closeはクライアント接続を閉じ、すべてのリソースを解放します。
// クライアントコンテキストをキャンセルし（SSEリーダーgoroutineを停止）、
// SSE接続が存在する場合は閉じます。
//
// このメソッドは複数回呼び出しても安全です。
//
// 戻り値：
//   - 正常に閉じた場合または既に閉じている場合はnil
//   - SSE接続のクローズに失敗した場合はerror
func (c *Client) Close() error {
	// Cancel the context to signal all goroutines to stop
	// すべてのgoroutineに停止を通知するためにコンテキストをキャンセル
	c.cancel()
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close the SSE connection if it exists
	// SSE接続が存在する場合は閉じる
	if c.sseConn != nil {
		return c.sseConn.Body.Close()
	}
	return nil
}

// JSONRPCRequest represents a JSON-RPC 2.0 request structure used for
// communicating with the DockMCP server.
//
// Fields:
// - JSONRPC: Protocol version, always "2.0"
// - ID: Request identifier for matching responses
// - Method: The method to invoke (e.g., "initialize", "tools/call")
// - Params: Optional parameters for the method
//
// JSONRPCRequestはDockMCPサーバーとの通信に使用される
// JSON-RPC 2.0リクエスト構造体を表します。
//
// フィールド：
// - JSONRPC: プロトコルバージョン、常に「2.0」
// - ID: レスポンスとのマッチング用リクエスト識別子
// - Method: 呼び出すメソッド（例：「initialize」、「tools/call」）
// - Params: メソッドのオプションパラメータ
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response for tool calls.
// It contains either a successful result or an error, but not both.
//
// Fields:
// - JSONRPC: Protocol version, always "2.0"
// - ID: Request identifier matching the original request
// - Result: Tool execution result (present on success)
// - Error: Error information (present on failure)
//
// JSONRPCResponseはツール呼び出し用のJSON-RPC 2.0レスポンスを表します。
// 成功した結果またはエラーのいずれかを含みますが、両方は含みません。
//
// フィールド：
// - JSONRPC: プロトコルバージョン、常に「2.0」
// - ID: 元のリクエストに対応するリクエスト識別子
// - Result: ツール実行結果（成功時に存在）
// - Error: エラー情報（失敗時に存在）
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  *ToolResult   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// InitializeResponse represents the response for the MCP initialize method.
// It has a generic Result type since initialization responses vary by server.
//
// Fields:
// - JSONRPC: Protocol version, always "2.0"
// - ID: Request identifier matching the initialize request
// - Result: Server capabilities and information (any type)
// - Error: Error information if initialization failed
//
// InitializeResponseはMCP initializeメソッドのレスポンスを表します。
// 初期化レスポンスはサーバーによって異なるため、汎用的なResult型を持ちます。
//
// フィールド：
// - JSONRPC: プロトコルバージョン、常に「2.0」
// - ID: 初期化リクエストに対応するリクエスト識別子
// - Result: サーバーの機能と情報（任意の型）
// - Error: 初期化が失敗した場合のエラー情報
type InitializeResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents an error in JSON-RPC 2.0 format.
// Standard error codes are defined by the JSON-RPC specification:
// - -32700: Parse error
// - -32600: Invalid request
// - -32601: Method not found
// - -32602: Invalid params
// - -32603: Internal error
//
// Fields:
// - Code: Numeric error code
// - Message: Human-readable error description
// - Data: Additional error data (optional)
//
// JSONRPCErrorはJSON-RPC 2.0形式のエラーを表します。
// 標準エラーコードはJSON-RPC仕様で定義されています：
// - -32700: 解析エラー
// - -32600: 無効なリクエスト
// - -32601: メソッドが見つからない
// - -32602: 無効なパラメータ
// - -32603: 内部エラー
//
// フィールド：
// - Code: 数値エラーコード
// - Message: 人が読めるエラー説明
// - Data: 追加のエラーデータ（オプション）
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ToolResult represents the result of an MCP tool call.
// It follows the MCP content structure for returning tool outputs.
//
// Fields:
// - Content: Array of content blocks (typically text results)
// - IsError: True if the tool execution failed (tool-level error, not JSON-RPC error)
//
// ToolResultはMCPツール呼び出しの結果を表します。
// ツール出力を返すためのMCPコンテンツ構造に従います。
//
// フィールド：
// - Content: コンテンツブロックの配列（通常はテキスト結果）
// - IsError: ツール実行が失敗した場合はtrue（JSON-RPCエラーではなくツールレベルのエラー）
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents a single content block in an MCP response.
// Currently supports text content type, which is the most common.
//
// Fields:
// - Type: Content type (e.g., "text", "image")
// - Text: The actual text content
//
// ContentはMCPレスポンス内の単一コンテンツブロックを表します。
// 現在は最も一般的なテキストコンテンツタイプをサポートしています。
//
// フィールド：
// - Type: コンテンツタイプ（例：「text」、「image」）
// - Text: 実際のテキストコンテンツ
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallTool calls an MCP tool on the DockMCP server using JSON-RPC 2.0 over HTTP POST.
// The response is received asynchronously via the SSE channel.
//
// The tool call process:
// 1. Validates that the client is connected (has session ID)
// 2. Creates a JSON-RPC request with method "tools/call"
// 3. Sends the request to /message endpoint with session ID
// 4. Waits for response on the SSE channel (with 30-second timeout)
// 5. Parses and returns the result or error
//
// Parameters:
//   - name: The name of the tool to call (e.g., "list_containers", "get_logs")
//   - arguments: Tool-specific arguments as key-value pairs
//
// Returns:
//   - *ToolResult: The tool execution result on success
//   - error: Error if not connected, request fails, or tool execution fails
//
// CallToolはJSON-RPC 2.0 over HTTP POSTを使用してDockMCPサーバー上のMCPツールを呼び出します。
// レスポンスはSSEチャネル経由で非同期に受信されます。
//
// ツール呼び出しプロセス：
// 1. クライアントが接続されている（セッションIDを持っている）ことを検証
// 2. メソッド「tools/call」でJSON-RPCリクエストを作成
// 3. セッションID付きで/messageエンドポイントにリクエストを送信
// 4. SSEチャネルでレスポンスを待機（30秒タイムアウト付き）
// 5. 結果またはエラーを解析して返却
//
// パラメータ：
//   - name: 呼び出すツールの名前（例：「list_containers」、「get_logs」）
//   - arguments: ツール固有の引数（キー値ペア）
//
// 戻り値：
//   - *ToolResult: 成功時のツール実行結果
//   - error: 未接続、リクエスト失敗、またはツール実行失敗時のエラー
func (c *Client) CallTool(name string, arguments map[string]interface{}) (*ToolResult, error) {
	// Get session ID in a thread-safe manner
	// スレッドセーフな方法でセッションIDを取得
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()

	// Verify that Connect() has been called successfully
	// Connect()が正常に呼び出されたことを確認
	if sessionID == "" {
		return nil, fmt.Errorf("not connected: call Connect() first")
	}

	// Create JSON-RPC request for tool call
	// ツール呼び出し用のJSON-RPCリクエストを作成
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
	}

	// Marshal the request to JSON
	// リクエストをJSONにマーシャル
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Include sessionId in URL for server to route the response correctly
	// サーバーがレスポンスを正しくルーティングするためにURLにsessionIdを含める
	url := fmt.Sprintf("%s/message?sessionId=%s", c.baseURL, sessionID)
	httpReq, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	// HTTPリクエストを送信
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Server responds with 202 Accepted for SSE mode (async response)
	// サーバーはSSEモードで202 Acceptedを返す（非同期レスポンス）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Wait for response from SSE channel with timeout and cancellation handling
	// タイムアウトとキャンセル処理付きでSSEチャネルからのレスポンスを待機
	select {
	case msg := <-c.messages:
		// Parse the JSON-RPC response
		// JSON-RPCレスポンスを解析
		var jsonrpcResp JSONRPCResponse
		if err := json.Unmarshal(msg, &jsonrpcResp); err != nil {
			return nil, fmt.Errorf("failed to decode SSE response: %w", err)
		}

		// Check for JSON-RPC level error (protocol error)
		// JSON-RPCレベルのエラーをチェック（プロトコルエラー）
		if jsonrpcResp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error: %s (code %d)", jsonrpcResp.Error.Message, jsonrpcResp.Error.Code)
		}

		// Check for tool execution error (tool returned error in result)
		// ツール実行エラーをチェック（ツールが結果でエラーを返した）
		if jsonrpcResp.Result != nil && jsonrpcResp.Result.IsError {
			if len(jsonrpcResp.Result.Content) > 0 {
				return nil, fmt.Errorf("tool call failed: %s", jsonrpcResp.Result.Content[0].Text)
			}
			return nil, fmt.Errorf("tool call failed with unknown error")
		}

		return jsonrpcResp.Result, nil

	case err := <-c.errors:
		// SSE connection error occurred
		// SSE接続エラーが発生
		return nil, fmt.Errorf("SSE connection error: %w", err)

	case <-time.After(30 * time.Second):
		// Response timeout
		// レスポンスタイムアウト
		return nil, fmt.Errorf("timeout waiting for response")

	case <-c.ctx.Done():
		// Client was closed during the operation
		// 操作中にクライアントが閉じられた
		return nil, fmt.Errorf("client closed")
	}
}

// HealthCheck checks if the DockMCP server is running and healthy.
// It sends a GET request to the /health endpoint and verifies a 200 OK response.
//
// This is useful for:
// - Verifying server availability before establishing SSE connection
// - Monitoring server health in production
// - Quick connectivity tests
//
// Returns:
//   - nil if server is healthy
//   - error if server is unreachable or returns non-200 status
//
// HealthCheckはDockMCPサーバーが実行中で正常かどうかをチェックします。
// /healthエンドポイントにGETリクエストを送信し、200 OKレスポンスを確認します。
//
// 以下の用途に有用です：
// - SSE接続確立前のサーバー可用性確認
// - 継続的なサーバーヘルスモニタリング
// - 簡易接続テスト
//
// 戻り値：
//   - サーバーが正常な場合はnil
//   - サーバーに到達できないまたは非200ステータスの場合はerror
func (c *Client) HealthCheck() error {
	// Construct the health check URL
	// ヘルスチェックURLを構築
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Verify 200 OK status
	// 200 OKステータスを確認
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unhealthy status: %d", resp.StatusCode)
	}

	return nil
}
