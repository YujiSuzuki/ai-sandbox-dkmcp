// Package mcp provides the MCP (Model Context Protocol) server implementation for DockMCP.
// This package handles JSON-RPC communication over Server-Sent Events (SSE) to enable
// AI assistants to interact with Docker containers in a controlled manner.
//
// mcpパッケージはDockMCPのMCP（Model Context Protocol）サーバー実装を提供します。
// このパッケージはServer-Sent Events（SSE）を介したJSON-RPC通信を処理し、
// AIアシスタントが制御された方法でDockerコンテナと対話できるようにします。
package mcp

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/docker"
)

// Server represents the MCP server that handles communication between AI assistants
// and Docker containers. It manages HTTP endpoints for SSE connections and JSON-RPC
// message handling, maintaining a list of connected clients.
//
// ServerはAIアシスタントとDockerコンテナ間の通信を処理するMCPサーバーを表します。
// SSE接続とJSON-RPCメッセージ処理のためのHTTPエンドポイントを管理し、
// 接続されたクライアントのリストを維持します。
type Server struct {
	// docker is the Docker client used to interact with containers.
	// Uses DockerClientInterface for dependency injection and testing.
	// dockerはコンテナと対話するために使用されるDockerクライアントです。
	// 依存性注入とテストのためにDockerClientInterfaceを使用します。
	docker docker.DockerClientInterface

	// port is the HTTP port the server listens on
	// portはサーバーがリッスンするHTTPポートです
	port int

	// httpServer is the underlying HTTP server instance
	// httpServerは基盤となるHTTPサーバーインスタンスです
	httpServer *http.Server

	// clients holds all connected MCP clients indexed by their session ID
	// clientsはセッションIDでインデックスされた全ての接続済みMCPクライアントを保持します
	clients map[string]*client

	// clientsMu protects concurrent access to the clients map
	// clientsMuはクライアントマップへの並行アクセスを保護します
	clientsMu sync.RWMutex

	// verbosity controls the logging verbosity level
	// Level 0: Normal (INFO level, minimal output)
	// Level 1 (-v): JSON output for initialized clients, filter noise
	// Level 2 (-vv): DEBUG level, JSON output, filter noise
	// Level 3 (-vvv): Full debug, all JSON, show noise
	// Level 4 (-vvvv): Full debug + HTTP headers
	//
	// verbosityはログの詳細レベルを制御します
	// レベル0: 通常（INFOレベル、最小出力）
	// レベル1 (-v): 初期化済みクライアントのJSON出力、ノイズをフィルタ
	// レベル2 (-vv): DEBUGレベル、JSON出力、ノイズをフィルタ
	// レベル3 (-vvv): フルデバッグ、全JSON、ノイズも表示
	// レベル4 (-vvvv): フルデバッグ + HTTPヘッダー表示
	verbosity int

	// requestCounter is an atomic counter for generating unique request IDs for logging.
	// This helps identify which log lines belong to the same request when logs are interleaved.
	//
	// requestCounterはログ用の一意のリクエストIDを生成するためのアトミックカウンターです。
	// ログが混在した際に、どのログ行が同じリクエストに属するかを識別するのに役立ちます。
	requestCounter uint64
}

// client represents a connected MCP client session. Each client maintains its own
// SSE channel for receiving responses and a context for lifecycle management.
//
// clientは接続されたMCPクライアントセッションを表します。各クライアントは
// レスポンスを受信するための独自のSSEチャネルとライフサイクル管理のための
// コンテキストを維持します。
type client struct {
	// id is the unique identifier for this client session
	// idはこのクライアントセッションの一意の識別子です
	id string

	// messages is the channel for sending SSE messages to this client
	// messagesはこのクライアントにSSEメッセージを送信するためのチャネルです
	messages chan []byte

	// ctx is the context for this client's lifecycle
	// ctxはこのクライアントのライフサイクル用のコンテキストです
	ctx context.Context

	// cancel is the function to cancel this client's context
	// cancelはこのクライアントのコンテキストをキャンセルする関数です
	cancel context.CancelFunc

	// initialized indicates whether this client has completed MCP initialization
	// initializedはこのクライアントがMCP初期化を完了したかどうかを示します
	initialized bool

	// clientName is the name of the connected client (e.g., "claude-code", "dkmcp-go-client")
	// clientNameは接続されたクライアントの名前です（例："claude-code"、"dkmcp-go-client"）
	clientName string

	// remoteAddr is the remote address of the client connection (e.g., "127.0.0.1:65182")
	// remoteAddrはクライアント接続のリモートアドレスです（例："127.0.0.1:65182"）
	remoteAddr string

	// userAgent is the User-Agent header from the HTTP request (helps identify the client application)
	// userAgentはHTTPリクエストのUser-Agentヘッダーです（クライアントアプリケーションの特定に役立ちます）
	userAgent string

	// connectedAt is the time when this client connected (for calculating session duration)
	// connectedAtはこのクライアントが接続した時刻です（セッション時間の計算用）
	connectedAt time.Time
}

// ServerOption is a functional option for configuring the MCP server.
// ServerOptionはMCPサーバーを設定するための関数オプションです。
type ServerOption func(*Server)

// WithVerbosity sets the verbosity level for detailed logging.
// Level 0: Normal, Level 1: JSON output, Level 2: DEBUG + JSON, Level 3: Full debug, Level 4: + HTTP headers
// WithVerbosityは詳細ログのverbosityレベルを設定します。
// レベル0: 通常、レベル1: JSON出力、レベル2: DEBUG + JSON、レベル3: フルデバッグ、レベル4: + HTTPヘッダー
func WithVerbosity(level int) ServerOption {
	return func(s *Server) {
		s.verbosity = level
	}
}

// NewServer creates a new MCP server with the given Docker client and port.
// The Docker client is used to execute container operations, while the port
// specifies which HTTP port the server will listen on.
// Optional ServerOption functions can be passed to configure additional settings.
//
// The dockerClient parameter accepts any implementation of DockerClientInterface,
// allowing for both real Docker clients and mock clients for testing.
//
// NewServerは指定されたDockerクライアントとポートで新しいMCPサーバーを作成します。
// Dockerクライアントはコンテナ操作の実行に使用され、ポートは
// サーバーがリッスンするHTTPポートを指定します。
// オプションのServerOption関数を渡して追加の設定を行えます。
//
// dockerClientパラメータはDockerClientInterfaceの任意の実装を受け入れ、
// 実際のDockerクライアントとテスト用のモッククライアントの両方を許可します。
func NewServer(dockerClient docker.DockerClientInterface, port int, opts ...ServerOption) *Server {
	s := &Server{
		docker:  dockerClient,
		port:    port,
		clients: make(map[string]*client),
	}

	// Apply optional configurations
	// オプションの設定を適用
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start starts the MCP server and begins listening for connections.
// It sets up three HTTP endpoints:
// - GET /sse: SSE endpoint for establishing client connections
// - POST /message: JSON-RPC endpoint for receiving client requests
// - GET /health: Health check endpoint for monitoring
//
// The server runs with logging and CORS middleware applied.
// This method blocks until the server is stopped.
//
// StartはMCPサーバーを起動し、接続のリッスンを開始します。
// 3つのHTTPエンドポイントを設定します：
// - GET /sse: クライアント接続確立用のSSEエンドポイント
// - POST /message: クライアントリクエスト受信用のJSON-RPCエンドポイント
// - GET /health: 監視用のヘルスチェックエンドポイント
//
// サーバーはロギングとCORSミドルウェアを適用して実行されます。
// このメソッドはサーバーが停止するまでブロックします。
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// SSE endpoint for MCP - clients connect here to establish SSE connection
	// MCP用のSSEエンドポイント - クライアントはここに接続してSSE接続を確立します
	mux.HandleFunc("GET /sse", s.handleSSE)

	// JSON-RPC endpoint for MCP - clients send requests here
	// MCP用のJSON-RPCエンドポイント - クライアントはここにリクエストを送信します
	mux.HandleFunc("POST /message", s.handleMessage)

	// Health check endpoint for monitoring and diagnostics
	// 監視と診断のためのヘルスチェックエンドポイント
	mux.HandleFunc("GET /health", s.handleHealth)

	// Apply middleware chain: logging -> origin validation -> CORS -> handlers
	// Per MCP specification, Origin header validation is required to prevent DNS rebinding attacks.
	// ミドルウェアチェーンを適用: ロギング -> Origin検証 -> CORS -> ハンドラー
	// MCP仕様に従い、DNSリバインディング攻撃を防ぐためにOriginヘッダー検証が必要です。
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.loggingMiddleware(s.originValidationMiddleware(s.corsMiddleware(mux))),
	}

	slog.Info("Starting MCP server", "port", s.port)
	if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Stop gracefully stops the MCP server within the given context deadline.
// It first cancels all client SSE connections, then shuts down the HTTP server.
//
// Stopは指定されたコンテキストのデッドライン内でMCPサーバーを正常に停止します。
// まず全クライアントのSSE接続をキャンセルし、その後HTTPサーバーをシャットダウンします。
func (s *Server) Stop(ctx context.Context) error {
	// Cancel all client contexts to close SSE connections
	// 全クライアントのコンテキストをキャンセルしてSSE接続を閉じる
	s.clientsMu.Lock()
	clientCount := len(s.clients)

	// Collect User-Agent statistics for uninitialized connections
	// 未初期化接続のUser-Agent統計を収集
	userAgentCounts := make(map[string]int)
	initializedCount := 0
	for _, c := range s.clients {
		if c.clientName == "" {
			ua := c.userAgent
			if ua == "" {
				ua = "(empty)"
			}
			userAgentCounts[ua]++
		} else {
			initializedCount++
		}
		if c.cancel != nil {
			c.cancel()
		}
	}
	s.clientsMu.Unlock()

	if clientCount > 0 {
		slog.Debug("Cancelled client contexts", "count", clientCount, "initialized", initializedCount, "uninitialized", clientCount-initializedCount)

		// Log User-Agent breakdown for uninitialized connections (helps identify noise source)
		// 未初期化接続のUser-Agent内訳をログ出力（ノイズ元の特定に役立つ）
		if len(userAgentCounts) > 0 {
			for ua, count := range userAgentCounts {
				slog.Debug("Uninitialized connections by User-Agent", "user_agent", ua, "count", count)
			}
		}
	}

	// Now shutdown the HTTP server
	// HTTPサーバーをシャットダウン
	if s.httpServer != nil {
		// Try graceful shutdown first with a short timeout
		// まず短いタイムアウトでグレースフルシャットダウンを試行
		shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := s.httpServer.Shutdown(shutdownCtx)
		if err != nil {
			// If graceful shutdown times out, force close
			// グレースフルシャットダウンがタイムアウトしたら強制終了
			slog.Debug("Graceful shutdown timed out, forcing close")
			return s.httpServer.Close()
		}
		return nil
	}
	return nil
}

// handleHealth handles health check requests by returning a simple JSON response
// indicating the server is operational. This endpoint is used by load balancers,
// monitoring systems, and diagnostic tools.
//
// handleHealthはサーバーが稼働中であることを示すシンプルなJSONレスポンスを返すことで
// ヘルスチェックリクエストを処理します。このエンドポイントはロードバランサー、
// 監視システム、診断ツールによって使用されます。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleSSE handles Server-Sent Events connections for MCP clients.
// This is the entry point for AI assistants to connect to the MCP server.
//
// The flow is:
// 1. Set appropriate SSE headers (Content-Type, Cache-Control, Connection)
// 2. Create a new client session with unique ID
// 3. Send the endpoint event with the message URL (as per MCP SSE spec)
// 4. Enter a loop to stream messages to the client until disconnection
//
// handleSSEはMCPクライアント用のServer-Sent Events接続を処理します。
// これはAIアシスタントがMCPサーバーに接続するためのエントリーポイントです。
//
// フローは以下の通りです：
// 1. 適切なSSEヘッダーを設定（Content-Type、Cache-Control、Connection）
// 2. 一意のIDを持つ新しいクライアントセッションを作成
// 3. メッセージURLを含むendpointイベントを送信（MCP SSE仕様に従って）
// 4. 切断されるまでクライアントにメッセージをストリーミングするループに入る
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers to establish proper event stream connection
	// 適切なイベントストリーム接続を確立するためのSSEヘッダーを設定
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new client session with cancellable context
	// キャンセル可能なコンテキストを持つ新しいクライアントセッションを作成
	ctx, cancel := context.WithCancel(r.Context())
	clientID := generateClientID()

	c := &client{
		id:          clientID,
		messages:    make(chan []byte, 10),
		ctx:         ctx,
		cancel:      cancel,
		initialized: false,
		remoteAddr:  r.RemoteAddr,
		userAgent:   r.UserAgent(),
		connectedAt: time.Now(),
	}

	// Register the client in the server's client map
	// サーバーのクライアントマップにクライアントを登録
	s.clientsMu.Lock()
	s.clients[clientID] = c
	s.clientsMu.Unlock()

	// Cleanup: Remove client from map and cancel context when connection closes
	// クリーンアップ：接続が閉じられたときにマップからクライアントを削除しコンテキストをキャンセル
	defer func() {
		// Calculate session duration
		// セッション時間を計算
		duration := time.Since(c.connectedAt)

		// Noise filtering based on verbosity level:
		// - Uninitialized connections (noise) are only shown at verbosity >= 3
		// - dkmcp-go-client is always Debug level (frequent short-lived connections)
		// - Other initialized clients are Info level
		//
		// verbosityレベルに基づくノイズフィルタリング：
		// - 未初期化接続（ノイズ）はverbosity >= 3でのみ表示
		// - dkmcp-go-clientは常にDebugレベル（頻繁な短期接続）
		// - その他の初期化済みクライアントはInfoレベル
		if !c.initialized {
			// Noise: uninitialized connection - only show at verbosity >= 3
			// ノイズ：未初期化接続 - verbosity >= 3でのみ表示
			if s.verbosity >= 3 {
				slog.Debug("[-] Client disconnected",
					append([]any{"clientID", clientID, "duration", duration.String(), "remote", c.remoteAddr}, clientLogAttrs(c)...)...,
				)
			}
		} else if strings.HasPrefix(c.clientName, "dkmcp-go-client") {
			// CLI client (including with suffix) - always Debug level
			// CLIクライアント（サフィックス付き含む）- 常にDebugレベル
			slog.Debug("[-] Client disconnected",
				append([]any{"clientID", clientID, "duration", duration.String(), "remote", c.remoteAddr}, clientLogAttrs(c)...)...,
			)
		} else {
			// Initialized client (Claude Code, etc.) - Info level
			// 初期化済みクライアント（Claude Code等）- Infoレベル
			slog.Info("[-] Client disconnected",
				append([]any{"clientID", clientID, "duration", duration.String(), "remote", c.remoteAddr}, clientLogAttrs(c)...)...,
			)
		}
		s.clientsMu.Lock()
		delete(s.clients, clientID)
		s.clientsMu.Unlock()
		cancel()
	}()

	// Send endpoint event as per MCP SSE specification
	// This tells the client where to send JSON-RPC requests
	// MCP SSE仕様に従ってendpointイベントを送信
	// これはクライアントにJSON-RPCリクエストの送信先を伝えます
	endpointURL := fmt.Sprintf("/message?sessionId=%s", clientID)
	// Only log SSE endpoint events at verbosity >= 3 (noise filtering)
	// Include user_agent to identify the source of connections
	// SSE endpointイベントはverbosity >= 3の場合のみログ出力（ノイズフィルタリング）
	// 接続元を特定するためにuser_agentを含める
	if s.verbosity >= 3 {
		slog.Debug("SSE client connected", "clientID", clientID, "remote", c.remoteAddr, "user_agent", c.userAgent)
	}
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream messages to the client until the context is cancelled
	// コンテキストがキャンセルされるまでクライアントにメッセージをストリーミング
	for {
		select {
		case <-ctx.Done():
			// Client disconnected or context cancelled
			// クライアントが切断されたかコンテキストがキャンセルされた
			return
		case msg := <-c.messages:
			// Send message as SSE "message" event per MCP 2024-11-05 specification
			// MCP 2024-11-05仕様に従って、SSE "message"イベントとしてメッセージを送信
			// Format: "event: message\ndata: <json>\n\n"
			// フォーマット: "event: message\ndata: <json>\n\n"
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// handleMessage handles JSON-RPC messages from MCP clients.
// Messages are received via POST request and responses are sent via the SSE channel.
//
// The flow is:
// 1. Extract session ID from query parameter
// 2. Validate the session exists
// 3. Decode the JSON-RPC request
// 4. Enforce initialization (non-initialize methods require prior initialization)
// 5. Process the request and send response via SSE
// 6. Return HTTP 202 Accepted to acknowledge receipt
//
// handleMessageはMCPクライアントからのJSON-RPCメッセージを処理します。
// メッセージはPOSTリクエストで受信され、レスポンスはSSEチャネルを通じて送信されます。
//
// フローは以下の通りです：
// 1. クエリパラメータからセッションIDを抽出
// 2. セッションが存在することを検証
// 3. JSON-RPCリクエストをデコード
// 4. 初期化を強制（initialize以外のメソッドは事前の初期化が必要）
// 5. リクエストを処理しSSE経由でレスポンスを送信
// 6. 受信確認としてHTTP 202 Acceptedを返す
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	// Get session ID from query parameter to identify the client
	// クライアントを識別するためにクエリパラメータからセッションIDを取得
	sessionID := r.URL.Query().Get("sessionId")
	slog.Debug("Received message request", "sessionID", sessionID, "url", r.URL.String())
	if sessionID == "" {
		slog.Warn("Missing sessionId parameter in message request")
		sendError(w, nil, -32600, "Missing sessionId parameter")
		return
	}

	// Verify session exists by looking up the client in the map
	// マップ内のクライアントを検索してセッションの存在を確認
	s.clientsMu.RLock()
	client, exists := s.clients[sessionID]
	if !exists {
		s.clientsMu.RUnlock()
		sendError(w, nil, -32600, "Invalid session ID")
		return
	}
	// Copy the initialized state while holding the lock to avoid race condition
	// 競合状態を避けるためにロックを保持したまま初期化状態をコピー
	clientInitialized := client.initialized
	s.clientsMu.RUnlock()

	// Read the raw request body for logging and decoding
	// ログ出力とデコードのために生のリクエストボディを読み取る
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		sendError(w, nil, -32700, "Parse error")
		return
	}

	// Decode the JSON-RPC request from the raw bytes
	// 生バイトからJSON-RPCリクエストをデコード
	var req JSONRPCRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		slog.Error("Failed to decode JSON-RPC request", "error", err)
		sendError(w, nil, -32700, "Parse error")
		return
	}
	slog.Debug("Decoded JSON-RPC request", "method", req.Method, "id", req.ID)

	// Generate unique request number for log correlation
	// ログの相関付けのために一意のリクエスト番号を生成
	reqNum := atomic.AddUint64(&s.requestCounter, 1)

	// Verbose mode: Log raw request JSON as received from client
	// 詳細モード: クライアントから受信した生のリクエストJSONをログ出力
	// verbosity >= 1 でJSON出力が有効
	if s.verbosity >= 1 {
		s.logVerboseRequest(client, &req, bodyBytes, reqNum)
	}

	// Enforce initialization: Only "initialize" method is allowed before initialization
	// 初期化の強制：初期化前は "initialize" メソッドのみが許可される
	// Use the copied value to avoid race condition with concurrent initialization
	// 並行初期化との競合状態を避けるためにコピーした値を使用
	if !clientInitialized && req.Method != "initialize" {
		sendErrorViaSSE(w, client, req.ID, -32000, "Client not initialized")
		return
	}

	// Process the request and get the result
	// リクエストを処理して結果を取得
	result, err := s.processRequest(client, &req)
	if err != nil {
		sendErrorViaSSE(w, client, req.ID, -32603, err.Error())
		return
	}

	// Send response via SSE channel (not directly in HTTP response)
	// SSEチャネル経由でレスポンスを送信（HTTPレスポンスに直接ではなく）
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		sendErrorViaSSE(w, client, req.ID, -32603, "Failed to marshal response")
		return
	}

	// Debug: Log response for debugging communication
	// デバッグ: 通信デバッグ用にレスポンスをログ出力
	slog.Debug("JSON-RPC response", "id", req.ID, "response_size", len(respBytes))

	// Verbose mode: Log full response details with pretty-printed JSON
	// JSON output is enabled for verbosity >= 1
	// 詳細モード: 整形されたJSONで完全なレスポンス詳細をログ出力
	// verbosity >= 1 でJSON出力が有効
	if s.verbosity >= 1 {
		s.logVerboseResponse(client, &resp, reqNum)
	}

	// Send to client's SSE channel with timeout handling
	// タイムアウト処理付きでクライアントのSSEチャネルに送信
	select {
	case client.messages <- respBytes:
		// ACK to the POST request - indicate message was accepted
		// POSTリクエストへのACK - メッセージが受け入れられたことを示す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	case <-client.ctx.Done():
		// Client has disconnected
		// クライアントが切断された
		sendError(w, req.ID, -32603, "Client disconnected")
	case <-time.After(5 * time.Second):
		// Timeout waiting to send message
		// メッセージ送信待ちのタイムアウト
		sendError(w, req.ID, -32603, "Timeout sending response")
	}
}

// processRequest processes a JSON-RPC request and routes it to the appropriate handler.
// It supports the following MCP methods:
// - tools/list: Returns the list of available tools
// - tools/call: Executes a specific tool
// - initialize: Initializes the MCP session
//
// processRequestはJSON-RPCリクエストを処理し、適切なハンドラにルーティングします。
// 以下のMCPメソッドをサポートしています：
// - tools/list: 利用可能なツールのリストを返す
// - tools/call: 特定のツールを実行する
// - initialize: MCPセッションを初期化する
func (s *Server) processRequest(c *client, req *JSONRPCRequest) (any, error) {
	logger := slog.Default()
	logger.Info("Processing JSON-RPC request",
		append([]any{"method", req.Method, "clientID", c.id}, clientLogAttrs(c)...)...,
	)

	switch req.Method {
	case "tools/list":
		// Return the list of available tools
		// 利用可能なツールのリストを返す
		return s.listTools()
	case "tools/call":
		// Extract tool name for logging purposes
		// ログ記録のためにツール名を抽出
		if params, ok := req.Params.(map[string]any); ok {
			if toolName, ok := params["name"].(string); ok {
				logger.Info("Tool called",
					append([]any{"tool", toolName, "clientID", c.id}, clientLogAttrs(c)...)...,
				)
			}
		}
		return s.callTool(c.ctx, req.Params)
	case "initialize":
		// Handle MCP initialization and update client context with client name
		// MCP初期化を処理し、クライアント名でクライアントコンテキストを更新
		result, clientName, clientVersion, err := s.initialize(req.Params)
		if err != nil {
			return nil, err
		}
		if clientName != "" {
			c.clientName = clientName
		}
		// NOTE: These fields are modified without holding clientsMu lock.
		// This is intentional: MCP protocol requires "initialize" to be called exactly once
		// per session before any other methods. Well-behaved clients (Claude Code, etc.)
		// follow this spec. The theoretical race condition (concurrent initialize requests
		// from the same session) only affects logging consistency, not security or data integrity.
		// If stricter protection is needed in the future, wrap these writes with clientsMu.Lock().
		//
		// 注: これらのフィールドはclientsMuロックを保持せずに変更されます。
		// これは意図的です：MCPプロトコルでは"initialize"は各セッションで他のメソッドの前に
		// 1回だけ呼び出されることが要求されています。正常なクライアント（Claude Code等）は
		// この仕様に従います。理論上のレース条件（同一セッションからの同時initializeリクエスト）
		// はログの一貫性にのみ影響し、セキュリティやデータ整合性には影響しません。
		// 将来より厳密な保護が必要な場合は、これらの書き込みをclientsMu.Lock()で囲んでください。
		c.initialized = true

		// Log client initialization at appropriate level
		// dkmcp-go-client (CLI, including with suffix) logs at Debug level, others at Info level
		// 適切なレベルでクライアント初期化をログ出力
		// dkmcp-go-client（CLI、サフィックス付き含む）はDebugレベル、その他はInfoレベル
		if strings.HasPrefix(clientName, "dkmcp-go-client") {
			slog.Debug("[+] Client connected (initialized)",
				append([]any{"clientID", c.id, "client_version", clientVersion}, clientLogAttrs(c)...)...,
			)
		} else {
			slog.Info("[+] Client connected (initialized)",
				append([]any{"clientID", c.id, "client_version", clientVersion}, clientLogAttrs(c)...)...,
			)
		}

		return result, nil
	default:
		return nil, fmt.Errorf("method not found: %s", req.Method)
	}
}

// loggingMiddleware logs HTTP requests and responses with timing information.
// It wraps the response writer to capture the status code for logging.
//
// loggingMiddlewareはHTTPリクエストとレスポンスをタイミング情報と共にログに記録します。
// ログ記録のためにステータスコードを取得するためにレスポンスライターをラップします。
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		// ステータスコードを取得するためのレスポンスライターラッパーを作成
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Check if this is a "noise" request that should be filtered
		// SSE connections without initialization are considered noise
		// フィルタリングすべき「ノイズ」リクエストかどうかを確認
		// 初期化なしのSSE接続はノイズと見なされます
		isNoise := r.URL.Path == "/sse"

		// Log incoming request (unless it's noise and verbosity < 3)
		// 受信リクエストをログに記録（ノイズかつverbosity < 3の場合を除く）
		if s.verbosity >= 3 || !isNoise {
			slog.Info("Request received",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
		}

		// Log HTTP headers at verbosity >= 4 (-vvvv)
		// verbosity >= 4 (-vvvv) の場合、HTTPヘッダーをログ出力
		if s.verbosity >= 4 {
			keys := make([]string, 0, len(r.Header))
			for k := range r.Header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				slog.Debug("HTTP header", "key", k, "value", strings.Join(r.Header[k], ", "))
			}
		}

		// Process the request
		// リクエストを処理
		next.ServeHTTP(ww, r)

		// Log response with duration (unless it's noise and verbosity < 3)
		// 処理時間と共にレスポンスをログに記録（ノイズかつverbosity < 3の場合を除く）
		duration := time.Since(start)
		if s.verbosity >= 3 || !isNoise {
			slog.Info("Response sent",
				"status", ww.statusCode,
				"duration", duration.String(),
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
// This is needed for logging middleware to record the response status.
//
// responseWriterはステータスコードを取得するためにhttp.ResponseWriterをラップします。
// これはログミドルウェアがレスポンスステータスを記録するために必要です。
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and forwards to the underlying ResponseWriter.
// WriteHeaderはステータスコードを取得し、基盤となるResponseWriterに転送します。
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher interface by delegating to the underlying ResponseWriter.
// This is essential for SSE to work properly as events need to be flushed immediately.
//
// Flushは基盤となるResponseWriterに委譲することでhttp.Flusherインターフェースを実装します。
// イベントは即座にフラッシュされる必要があるため、これはSSEが正しく動作するために不可欠です。
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// clientLogAttrs returns slog key-value pairs for client identification in log messages.
// It always returns both "client_name" and "user_agent" as separate fields.
//
// The client_name field reflects the client's initialization state:
//   - Initialized with name:    the raw MCP clientInfo.name (e.g., "dkmcp-go-client")
//   - Initialized without name: "(empty name)"
//   - Not initialized:          "(not initialized)"
//
// clientLogAttrsはログメッセージでのクライアント識別用のslogキーバリューペアを返します。
// 常に "client_name" と "user_agent" を別々のフィールドとして返します。
//
// client_nameフィールドはクライアントの初期化状態を反映します：
//   - 名前付きで初期化済み：    生のMCP clientInfo.name（例："dkmcp-go-client"）
//   - 名前なしで初期化済み：    "(empty name)"
//   - 未初期化：                "(not initialized)"
func clientLogAttrs(c *client) []any {
	var name string
	switch {
	case c.clientName != "":
		name = c.clientName
	case c.initialized:
		name = "(empty name)"
	default:
		name = "(not initialized)"
	}
	return []any{"client_name", name, "user_agent", c.userAgent}
}

// corsMiddleware adds CORS headers to allow cross-origin requests.
// This is necessary for web-based MCP clients to connect to the server.
// It handles preflight OPTIONS requests and adds appropriate headers to all responses.
// Only localhost origins are allowed, consistent with originValidationMiddleware.
//
// corsMiddlewareはクロスオリジンリクエストを許可するためにCORSヘッダーを追加します。
// これはWebベースのMCPクライアントがサーバーに接続するために必要です。
// プリフライトOPTIONSリクエストを処理し、全てのレスポンスに適切なヘッダーを追加します。
// originValidationMiddlewareと一貫して、localhostオリジンのみが許可されます。
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only set CORS header for allowed origins (localhost)
		// 許可されたオリジン（localhost）に対してのみCORSヘッダーを設定
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		// プリフライトOPTIONSリクエストを処理
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin checks if the given origin is a localhost origin.
// This is used by both corsMiddleware and originValidationMiddleware.
//
// isAllowedOriginは指定されたオリジンがlocalhostオリジンかどうかをチェックします。
// これはcorsMiddlewareとoriginValidationMiddlewareの両方で使用されます。
func isAllowedOrigin(origin string) bool {
	allowedOrigins := []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
		"http://[::1]",
		"https://[::1]",
	}

	for _, allowedOrigin := range allowedOrigins {
		// Check exact match or prefix match with port (e.g., localhost:3000)
		// 完全一致またはポート付きプレフィックス一致をチェック（例：localhost:3000）
		if origin == allowedOrigin || (len(origin) > len(allowedOrigin) && origin[:len(allowedOrigin)+1] == allowedOrigin+":") {
			return true
		}
	}
	return false
}

// originValidationMiddleware validates the Origin header to prevent DNS rebinding attacks.
// Per MCP specification, servers MUST validate the Origin header on all incoming connections.
// This middleware allows requests from localhost origins or requests without an Origin header
// (which are typically from server-side clients like curl or CLI tools).
//
// originValidationMiddlewareはDNSリバインディング攻撃を防ぐためにOriginヘッダーを検証します。
// MCP仕様に従い、サーバーは全ての受信接続でOriginヘッダーを検証しなければなりません。
// このミドルウェアはlocalhostオリジンからのリクエストまたはOriginヘッダーのないリクエスト
// （通常、curlやCLIツールなどのサーバーサイドクライアントから）を許可します。
func (s *Server) originValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow requests without Origin header (server-side clients, curl, etc.)
		// Originヘッダーのないリクエストを許可（サーバーサイドクライアント、curlなど）
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Validate Origin header using shared helper function
		// 共通のヘルパー関数を使用してOriginヘッダーを検証
		if !isAllowedOrigin(origin) {
			slog.Warn("Rejected request due to invalid Origin header",
				"origin", origin,
				"remote", r.RemoteAddr,
			)
			http.Error(w, "Forbidden: Invalid Origin header", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// sendError sends a JSON-RPC error response directly to the HTTP response.
// This is used as a fallback when the SSE channel is not available or
// when errors occur before the session is established.
//
// sendErrorはJSON-RPCエラーレスポンスをHTTPレスポンスに直接送信します。
// これはSSEチャネルが利用できない場合や、セッションが確立される前に
// エラーが発生した場合のフォールバックとして使用されます。
func sendError(w http.ResponseWriter, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	// JSON-RPC errors are sent with HTTP 200 OK status
	// JSON-RPCエラーはHTTP 200 OKステータスで送信される
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// sendErrorViaSSE sends a JSON-RPC error response via the client's SSE channel.
// This ensures errors are delivered through the same channel as successful responses,
// maintaining MCP's event-driven communication model.
//
// sendErrorViaSSEはクライアントのSSEチャネル経由でJSON-RPCエラーレスポンスを送信します。
// これによりエラーが成功レスポンスと同じチャネルを通じて配信され、
// MCPプロトコルのイベント駆動型通信モデルが維持されます。
func sendErrorViaSSE(w http.ResponseWriter, c *client, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		// Fallback to direct HTTP response if marshal fails
		// マーシャルに失敗した場合はHTTPレスポンスに直接フォールバック
		sendError(w, id, code, message)
		return
	}

	// Send to client's SSE channel with timeout handling
	// タイムアウト処理付きでクライアントのSSEチャネルに送信
	select {
	case c.messages <- respBytes:
		// ACK to the POST request
		// POSTリクエストへのACK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	case <-c.ctx.Done():
		// Client has disconnected
		// クライアントが切断された
		sendError(w, id, -32603, "Client disconnected")
	case <-time.After(5 * time.Second):
		// Timeout waiting to send error response
		// エラーレスポンス送信待ちのタイムアウト
		sendError(w, id, -32603, "Timeout sending error response")
	}
}

// generateClientID generates a unique client ID using the current timestamp.
// Each client session needs a unique identifier to track its SSE connection
// and associate incoming JSON-RPC requests with the correct session.
//
// generateClientIDは現在のタイムスタンプを使用して一意のクライアントIDを生成します。
// 各クライアントセッションはSSE接続を追跡し、受信したJSON-RPCリクエストを
// 正しいセッションに関連付けるために一意の識別子が必要です。
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

// JSONRPCRequest represents a JSON-RPC 2.0 request message.
// It contains the protocol version, request ID, method name, and optional parameters.
//
// JSONRPCRequestはJSON-RPC 2.0リクエストメッセージを表します。
// プロトコルバージョン、リクエストID、メソッド名、およびオプションのパラメータを含みます。
type JSONRPCRequest struct {
	// JSONRPC is the JSON-RPC protocol version (should be "2.0")
	// JSONRPCはJSON-RPCプロトコルバージョン（"2.0"であるべき）
	JSONRPC string `json:"jsonrpc"`

	// ID is the request identifier used to match responses to requests
	// IDはレスポンスをリクエストに対応させるために使用されるリクエスト識別子
	ID any `json:"id"`

	// Method is the name of the method to invoke
	// Methodは呼び出すメソッドの名前
	Method string `json:"method"`

	// Params contains optional parameters for the method
	// Paramsはメソッドのオプションパラメータを含む
	Params any `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response message.
// It contains either a result (for successful responses) or an error (for failed ones).
//
// JSONRPCResponseはJSON-RPC 2.0レスポンスメッセージを表します。
// 結果（成功時）またはエラー（失敗時）のいずれかを含みます。
type JSONRPCResponse struct {
	// JSONRPC is the JSON-RPC protocol version (should be "2.0")
	// JSONRPCはJSON-RPCプロトコルバージョン（"2.0"であるべき）
	JSONRPC string `json:"jsonrpc"`

	// ID is the request identifier (matches the request's ID)
	// IDはリクエスト識別子（リクエストのIDと一致）
	ID any `json:"id"`

	// Result contains the method result (nil if error)
	// Resultはメソッドの結果を含む（エラー時はnil）
	Result any `json:"result,omitempty"`

	// Error contains error details (nil if successful)
	// Errorはエラーの詳細を含む（成功時はnil）
	Error *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
// It contains a numeric error code, human-readable message, and optional data.
//
// JSONRPCErrorはJSON-RPC 2.0エラーオブジェクトを表します。
// 数値のエラーコード、人が読めるメッセージ、およびオプションのデータを含みます。
type JSONRPCError struct {
	// Code is a numeric error code (standard JSON-RPC codes or custom)
	// Codeは数値のエラーコード（標準JSON-RPCコードまたはカスタム）
	Code int `json:"code"`

	// Message is a human-readable error description
	// Messageは人が読めるエラーの説明
	Message string `json:"message"`

	// Data contains optional additional error information
	// Dataはオプションの追加エラー情報を含む
	Data any `json:"data,omitempty"`
}

// logVerboseRequest logs detailed request information when verbose mode is enabled.
// It outputs the raw JSON as received from the client, pretty-printed for readability.
// The rawJSON parameter contains the original bytes from the HTTP request body,
// preserving the exact content sent by the client (including unknown fields and original structure).
// The reqNum parameter is used to correlate log lines when requests are interleaved.
//
// logVerboseRequestは詳細モードが有効な場合にリクエストの詳細情報をログ出力します。
// クライアントから受信した生のJSONを、可読性のために整形して出力します。
// rawJSONパラメータはHTTPリクエストボディの元のバイト列を保持しており、
// クライアントが送信した正確な内容（未知のフィールドや元の構造を含む）を保存しています。
// reqNumパラメータはリクエストが混在した際にログ行を相関付けるために使用されます。
func (s *Server) logVerboseRequest(c *client, req *JSONRPCRequest, rawJSON []byte, reqNum uint64) {
	// Extract tool name for better readability
	// 可読性向上のためツール名を抽出
	toolName := ""
	if req.Method == "tools/call" {
		if params, ok := req.Params.(map[string]any); ok {
			if name, ok := params["name"].(string); ok {
				toolName = name
			}
		}
	}

	// Pretty-print the raw JSON received from the client
	// クライアントから受信した生のJSONを整形して出力
	var prettyBuf bytes.Buffer
	if err := json.Indent(&prettyBuf, rawJSON, "", "  "); err != nil {
		// If pretty-printing fails, use the raw JSON as-is
		// 整形に失敗した場合は生のJSONをそのまま使用
		slog.Warn("Failed to pretty-print raw request JSON, using raw bytes",
			"function", "logVerboseRequest",
			"error", err,
		)
		prettyBuf.Reset()
		prettyBuf.Write(rawJSON)
	}

	// Log with visual separator for easy identification
	// The request number helps correlate log lines when requests are interleaved
	// 識別しやすいように視覚的な区切りでログ出力
	// リクエスト番号はリクエストが混在した際にログ行を相関付けるのに役立ちます
	slog.Info(fmt.Sprintf("═══ [#%d] ═══════════════════════════════════════════════════════════", reqNum))
	baseAttrs := clientLogAttrs(c)
	if toolName != "" {
		slog.Info("▼ REQUEST", append(baseAttrs, "method", req.Method, "tool", toolName, "id", req.ID)...)
	} else {
		slog.Info("▼ REQUEST", append(baseAttrs, "method", req.Method, "id", req.ID)...)
	}
	slog.Info("Request body:\n" + prettyBuf.String())
	slog.Info(fmt.Sprintf("─── [#%d] ───────────────────────────────────────────────────────────", reqNum))
}

// logVerboseResponse logs detailed response information when verbose mode is enabled.
// It outputs the full JSON-RPC response with pretty-printed formatting.
// The reqNum parameter is used to correlate log lines when requests are interleaved.
//
// logVerboseResponseは詳細モードが有効な場合にレスポンスの詳細情報をログ出力します。
// 整形されたフォーマットで完全なJSON-RPCレスポンスを出力します。
// reqNumパラメータはリクエストが混在した際にログ行を相関付けるために使用されます。
func (s *Server) logVerboseResponse(c *client, resp *JSONRPCResponse, reqNum uint64) {
	// Pretty-print the full response
	// 完全なレスポンスを整形して出力
	// NOTE: This marshal should never fail for a valid JSONRPCResponse struct.
	// If it does, it indicates a bug in the response structure (e.g., channel or func type).
	// NOTE: 有効なJSONRPCResponse構造体でこのmarshalが失敗することはないはずです。
	// 失敗した場合、レスポンス構造にバグがあることを示します（例：channel型やfunc型の混入）。
	prettyJSON, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		slog.Error("Unexpected marshal failure in verbose response logging - possible bug in response structure",
			"function", "logVerboseResponse",
			"error", err,
		)
		return
	}

	// Log with visual separator for easy identification
	// The request number helps correlate log lines when requests are interleaved
	// 識別しやすいように視覚的な区切りでログ出力
	// リクエスト番号はリクエストが混在した際にログ行を相関付けるのに役立ちます
	baseAttrs := clientLogAttrs(c)
	if resp.Error != nil {
		slog.Info("▲ RESPONSE (ERROR)", append(baseAttrs, "id", resp.ID, "error_code", resp.Error.Code)...)
	} else {
		slog.Info("▲ RESPONSE", append(baseAttrs, "id", resp.ID)...)
	}
	slog.Info("Response body:\n" + string(prettyJSON))
	slog.Info(fmt.Sprintf("═══ [#%d] ═══════════════════════════════════════════════════════════", reqNum))
}
