package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YujiSuzuki/ai-sandbox-dkmcp/dkmcp/internal/config"
)

func TestNewLogger(t *testing.T) {
	t.Run("stdout logger", func(t *testing.T) {
		cfg := config.AuditConfig{
			Enabled: true,
			File:    "",
			Events: config.AuditEvents{
				ToolCalls:    true,
				AccessDenied: true,
			},
		}

		logger, err := newLogger(cfg)
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		defer logger.Close()

		if logger.file != nil {
			t.Error("expected file to be nil for stdout logger")
		}
	})

	t.Run("file logger", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "audit.log")

		cfg := config.AuditConfig{
			Enabled: true,
			File:    logFile,
			Events: config.AuditEvents{
				ToolCalls: true,
			},
		}

		logger, err := newLogger(cfg)
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		defer logger.Close()

		if logger.file == nil {
			t.Error("expected file to be non-nil for file logger")
		}

		// Verify file was created
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("expected log file to be created")
		}
	})
}

func TestLoggerLog(t *testing.T) {
	t.Run("logs tool call event", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "audit.log")

		cfg := config.AuditConfig{
			Enabled: true,
			File:    logFile,
			Events: config.AuditEvents{
				ToolCalls: true,
			},
		}

		logger, err := newLogger(cfg)
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		defer logger.Close()

		ctx := context.Background()
		logger.Log(ctx, Event{
			Type:       EventToolCall,
			Tool:       "exec_command",
			Container:  "test-api",
			Result:     ResultSuccess,
			DurationMs: 123,
			Details: map[string]any{
				"command": "npm test",
			},
		})

		// Close to flush
		logger.Close()

		// Read and verify log
		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		var logEntry map[string]any
		if err := json.Unmarshal(data, &logEntry); err != nil {
			t.Fatalf("failed to parse log entry: %v", err)
		}

		if logEntry["event_type"] != "tool_call" {
			t.Errorf("expected event_type=tool_call, got %v", logEntry["event_type"])
		}
		if logEntry["tool"] != "exec_command" {
			t.Errorf("expected tool=exec_command, got %v", logEntry["tool"])
		}
		if logEntry["container"] != "test-api" {
			t.Errorf("expected container=test-api, got %v", logEntry["container"])
		}
	})

	t.Run("respects event type config", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "audit.log")

		cfg := config.AuditConfig{
			Enabled: true,
			File:    logFile,
			Events: config.AuditEvents{
				ToolCalls:    false, // Disabled
				AccessDenied: true,
			},
		}

		logger, err := newLogger(cfg)
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		defer logger.Close()

		ctx := context.Background()

		// This should NOT be logged
		logger.Log(ctx, Event{
			Type: EventToolCall,
			Tool: "get_logs",
		})

		// This SHOULD be logged
		logger.Log(ctx, Event{
			Type:         EventAccessDenied,
			Tool:         "read_file",
			ErrorMessage: "blocked path",
		})

		logger.Close()

		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		// Should only contain access_denied event
		if !bytes.Contains(data, []byte("access_denied")) {
			t.Error("expected access_denied event in log")
		}
		if bytes.Contains(data, []byte("tool_call")) {
			t.Error("did not expect tool_call event in log")
		}
	})

	t.Run("disabled logger does not log", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "audit.log")

		cfg := config.AuditConfig{
			Enabled: false, // Disabled
			File:    logFile,
		}

		logger, err := newLogger(cfg)
		if err != nil {
			t.Fatalf("newLogger() error = %v", err)
		}
		defer logger.Close()

		ctx := context.Background()
		logger.Log(ctx, Event{
			Type: EventToolCall,
			Tool: "test",
		})

		logger.Close()

		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		if len(data) > 0 {
			t.Error("expected empty log file when audit is disabled")
		}
	})
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		events    config.AuditEvents
		want      bool
	}{
		{
			name:      "tool_calls enabled",
			eventType: EventToolCall,
			events:    config.AuditEvents{ToolCalls: true},
			want:      true,
		},
		{
			name:      "tool_calls disabled",
			eventType: EventToolCall,
			events:    config.AuditEvents{ToolCalls: false},
			want:      false,
		},
		{
			name:      "access_denied enabled",
			eventType: EventAccessDenied,
			events:    config.AuditEvents{AccessDenied: true},
			want:      true,
		},
		{
			name:      "client_connect enabled",
			eventType: EventClientConnect,
			events:    config.AuditEvents{ClientConnections: true},
			want:      true,
		},
		{
			name:      "client_disconnect enabled",
			eventType: EventClientDisconnect,
			events:    config.AuditEvents{ClientConnections: true},
			want:      true,
		},
		{
			name:      "security_policy enabled",
			eventType: EventSecurityPolicy,
			events:    config.AuditEvents{SecurityPolicy: true},
			want:      true,
		},
		{
			name:      "security_policy disabled",
			eventType: EventSecurityPolicy,
			events:    config.AuditEvents{SecurityPolicy: false},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &Logger{
				cfg: config.AuditConfig{
					Enabled: true,
					Events:  tt.events,
				},
			}

			got := logger.shouldLog(tt.eventType)
			if got != tt.want {
				t.Errorf("shouldLog(%v) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Reset global state
	ResetLogger()
	defer ResetLogger()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	cfg := config.AuditConfig{
		Enabled: true,
		File:    logFile,
		Events: config.AuditEvents{
			ToolCalls:         true,
			AccessDenied:      true,
			ClientConnections: true,
			SecurityPolicy:    true,
		},
	}

	err := Initialize(cfg)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	ctx := context.Background()

	// Test LogToolCall
	LogToolCall(ctx, "exec_command", "api", ResultSuccess, 100, map[string]any{"cmd": "test"})

	// Test LogAccessDenied
	LogAccessDenied(ctx, "read_file", "api", "blocked path", map[string]any{"path": "/secrets"})

	// Test LogClientConnect
	LogClientConnect(ctx, "Claude Code", "session-123")

	// Test LogClientDisconnect
	LogClientDisconnect(ctx, "Claude Code", "session-123", 5000)

	// Test LogSecurityPolicy
	LogSecurityPolicy(ctx, "get_security_policy", nil)

	// Verify logger was set
	if GetLogger() == nil {
		t.Error("expected global logger to be set")
	}

	// Verify log file contains all expected event types
	// ログファイルに期待される全イベントタイプが含まれていることを確認
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Parse each line as JSON and collect event types
	// 各行をJSONとしてパースしてイベントタイプを収集
	lines := bytes.Split(data, []byte("\n"))
	eventTypes := make(map[string]bool)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if et, ok := entry["event_type"].(string); ok {
			eventTypes[et] = true
		}
	}

	// Verify all 5 event types were logged
	// 5つのイベントタイプ全てがログされたことを確認
	expectedEvents := []string{"tool_call", "access_denied", "client_connect", "client_disconnect", "security_policy"}
	for _, expected := range expectedEvents {
		if !eventTypes[expected] {
			t.Errorf("expected event type %q not found in log file", expected)
		}
	}
}

func TestMeasureDuration(t *testing.T) {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := MeasureDuration(start)

	if duration < 10 {
		t.Errorf("expected duration >= 10ms, got %dms", duration)
	}
}

func TestNilLoggerSafety(t *testing.T) {
	// Reset to ensure nil logger
	ResetLogger()

	ctx := context.Background()

	// These should not panic
	LogToolCall(ctx, "test", "container", ResultSuccess, 0, nil)
	LogAccessDenied(ctx, "test", "container", "reason", nil)
	LogClientConnect(ctx, "client", "session")
	LogClientDisconnect(ctx, "client", "session", 0)
	LogSecurityPolicy(ctx, "tool", nil)

	var nilLogger *Logger
	nilLogger.Log(ctx, Event{Type: EventToolCall})
}

// BenchmarkLogger measures logging performance
func BenchmarkLogger(b *testing.B) {
	tmpDir := b.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	cfg := config.AuditConfig{
		Enabled: true,
		File:    logFile,
		Events: config.AuditEvents{
			ToolCalls: true,
		},
	}

	logger, _ := newLogger(cfg)
	defer logger.Close()

	ctx := context.Background()
	event := Event{
		Type:       EventToolCall,
		Tool:       "exec_command",
		Container:  "test-api",
		Result:     ResultSuccess,
		DurationMs: 123,
		Details:    map[string]any{"command": "npm test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(ctx, event)
	}
}

// TestLoggerCustomHandler tests custom slog handler usage
func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer

	logger := &Logger{
		cfg: config.AuditConfig{
			Enabled: true,
			Events: config.AuditEvents{
				ToolCalls: true,
			},
		},
		logger: slog.New(slog.NewJSONHandler(&buf, nil)),
	}

	ctx := context.Background()
	logger.Log(ctx, Event{
		Type:      EventToolCall,
		Tool:      "test_tool",
		Container: "test_container",
		Result:    ResultSuccess,
	})

	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	if entry["msg"] != "audit_event" {
		t.Errorf("expected msg=audit_event, got %v", entry["msg"])
	}
}
