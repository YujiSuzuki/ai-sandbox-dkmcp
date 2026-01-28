// Package audit provides audit logging for DockMCP.
// It records security-relevant events for monitoring and compliance.
//
// auditパッケージはDockMCPの監査ログを提供します。
// セキュリティ関連イベントを監視とコンプライアンスのために記録します。
package audit

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

// EventType represents the type of audit event.
// EventTypeは監査イベントのタイプを表します。
type EventType string

const (
	// EventToolCall is logged when an MCP tool is invoked.
	// EventToolCallはMCPツールが呼び出された時にログ記録されます。
	EventToolCall EventType = "tool_call"

	// EventAccessDenied is logged when access is denied.
	// EventAccessDeniedはアクセスが拒否された時にログ記録されます。
	EventAccessDenied EventType = "access_denied"

	// EventClientConnect is logged when a client connects.
	// EventClientConnectはクライアントが接続した時にログ記録されます。
	EventClientConnect EventType = "client_connect"

	// EventClientDisconnect is logged when a client disconnects.
	// EventClientDisconnectはクライアントが切断した時にログ記録されます。
	EventClientDisconnect EventType = "client_disconnect"

	// EventSecurityPolicy is logged when security policy is queried.
	// EventSecurityPolicyはセキュリティポリシーが照会された時にログ記録されます。
	EventSecurityPolicy EventType = "security_policy"
)

// Result represents the outcome of an operation.
// Resultは操作の結果を表します。
type Result string

const (
	ResultSuccess Result = "success"
	ResultDenied  Result = "denied"
	ResultError   Result = "error"
)

// Event represents an audit event.
// Eventは監査イベントを表します。
type Event struct {
	// Type is the type of audit event.
	// Typeは監査イベントのタイプです。
	Type EventType

	// Tool is the name of the MCP tool (for tool_call events).
	// ToolはMCPツールの名前です（tool_callイベント用）。
	Tool string

	// Container is the target container name.
	// Containerは対象コンテナ名です。
	Container string

	// Result is the outcome of the operation.
	// Resultは操作の結果です。
	Result Result

	// ClientName is the name of the MCP client.
	// ClientNameはMCPクライアントの名前です。
	ClientName string

	// SessionID is the unique session identifier.
	// SessionIDはユニークなセッション識別子です。
	SessionID string

	// Details contains additional event-specific information.
	// Detailsは追加のイベント固有情報を含みます。
	Details map[string]any

	// DurationMs is the operation duration in milliseconds.
	// DurationMsは操作の所要時間（ミリ秒）です。
	DurationMs int64

	// ErrorMessage contains the error message if Result is error/denied.
	// ErrorMessageはResultがerror/deniedの場合のエラーメッセージです。
	ErrorMessage string
}

// Logger is the audit logger.
// Loggerは監査ロガーです。
type Logger struct {
	cfg    config.AuditConfig
	logger *slog.Logger
	mu     sync.Mutex
	file   *os.File
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Initialize sets up the global audit logger.
// InitializeはグローバルAuditロガーを設定します。
func Initialize(cfg config.AuditConfig) error {
	var initErr error
	once.Do(func() {
		logger, err := newLogger(cfg)
		if err != nil {
			initErr = err
			return
		}
		globalLogger = logger
	})
	return initErr
}

// newLogger creates a new audit logger.
// newLoggerは新しい監査ロガーを作成します。
func newLogger(cfg config.AuditConfig) (*Logger, error) {
	l := &Logger{cfg: cfg}

	var output io.Writer = os.Stdout
	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		l.file = f
		output = f
	}

	l.logger = slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return l, nil
}

// Close closes the audit logger.
// Closeは監査ロガーをクローズします。
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Log records an audit event.
// Logは監査イベントを記録します。
func (l *Logger) Log(ctx context.Context, event Event) {
	if l == nil || !l.cfg.Enabled {
		return
	}

	// Check if this event type should be logged
	// このイベントタイプをログ記録すべきかチェック
	if !l.shouldLog(event.Type) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	attrs := []any{
		slog.String("event_type", string(event.Type)),
	}

	if event.Tool != "" {
		attrs = append(attrs, slog.String("tool", event.Tool))
	}
	if event.Container != "" {
		attrs = append(attrs, slog.String("container", event.Container))
	}
	if event.Result != "" {
		attrs = append(attrs, slog.String("result", string(event.Result)))
	}
	if event.ClientName != "" {
		attrs = append(attrs, slog.String("client_name", event.ClientName))
	}
	if event.SessionID != "" {
		attrs = append(attrs, slog.String("session_id", event.SessionID))
	}
	if event.DurationMs > 0 {
		attrs = append(attrs, slog.Int64("duration_ms", event.DurationMs))
	}
	if event.ErrorMessage != "" {
		attrs = append(attrs, slog.String("error", event.ErrorMessage))
	}
	if len(event.Details) > 0 {
		attrs = append(attrs, slog.Any("details", event.Details))
	}

	l.logger.InfoContext(ctx, "audit_event", attrs...)
}

// shouldLog checks if an event type should be logged based on config.
// shouldLogは設定に基づいてイベントタイプをログ記録すべきかチェックします。
func (l *Logger) shouldLog(eventType EventType) bool {
	switch eventType {
	case EventToolCall:
		return l.cfg.Events.ToolCalls
	case EventAccessDenied:
		return l.cfg.Events.AccessDenied
	case EventClientConnect, EventClientDisconnect:
		return l.cfg.Events.ClientConnections
	case EventSecurityPolicy:
		return l.cfg.Events.SecurityPolicy
	default:
		return true
	}
}

// LogToolCall logs a tool invocation.
// LogToolCallはツール呼び出しをログ記録します。
func LogToolCall(ctx context.Context, tool, container string, result Result, durationMs int64, details map[string]any) {
	if globalLogger == nil {
		return
	}
	globalLogger.Log(ctx, Event{
		Type:       EventToolCall,
		Tool:       tool,
		Container:  container,
		Result:     result,
		DurationMs: durationMs,
		Details:    details,
	})
}

// LogAccessDenied logs an access denial.
// LogAccessDeniedはアクセス拒否をログ記録します。
func LogAccessDenied(ctx context.Context, tool, container, reason string, details map[string]any) {
	if globalLogger == nil {
		return
	}
	globalLogger.Log(ctx, Event{
		Type:         EventAccessDenied,
		Tool:         tool,
		Container:    container,
		Result:       ResultDenied,
		ErrorMessage: reason,
		Details:      details,
	})
}

// LogClientConnect logs a client connection.
// LogClientConnectはクライアント接続をログ記録します。
func LogClientConnect(ctx context.Context, clientName, sessionID string) {
	if globalLogger == nil {
		return
	}
	globalLogger.Log(ctx, Event{
		Type:       EventClientConnect,
		ClientName: clientName,
		SessionID:  sessionID,
		Result:     ResultSuccess,
	})
}

// LogClientDisconnect logs a client disconnection.
// LogClientDisconnectはクライアント切断をログ記録します。
func LogClientDisconnect(ctx context.Context, clientName, sessionID string, durationMs int64) {
	if globalLogger == nil {
		return
	}
	globalLogger.Log(ctx, Event{
		Type:       EventClientDisconnect,
		ClientName: clientName,
		SessionID:  sessionID,
		Result:     ResultSuccess,
		DurationMs: durationMs,
	})
}

// LogSecurityPolicy logs a security policy query.
// LogSecurityPolicyはセキュリティポリシー照会をログ記録します。
func LogSecurityPolicy(ctx context.Context, tool string, details map[string]any) {
	if globalLogger == nil {
		return
	}
	globalLogger.Log(ctx, Event{
		Type:    EventSecurityPolicy,
		Tool:    tool,
		Result:  ResultSuccess,
		Details: details,
	})
}

// MeasureDuration is a helper to measure operation duration.
// MeasureDurationは操作の所要時間を計測するヘルパーです。
func MeasureDuration(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// GetLogger returns the global audit logger for testing.
// GetLoggerはテスト用にグローバルAuditロガーを返します。
func GetLogger() *Logger {
	return globalLogger
}

// SetLogger sets the global audit logger (for testing).
// SetLoggerはグローバルAuditロガーを設定します（テスト用）。
func SetLogger(l *Logger) {
	globalLogger = l
}

// ResetLogger resets the global audit logger (for testing).
// ResetLoggerはグローバルAuditロガーをリセットします（テスト用）。
func ResetLogger() {
	if globalLogger != nil {
		globalLogger.Close()
	}
	globalLogger = nil
	once = sync.Once{}
}
