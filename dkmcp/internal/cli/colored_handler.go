// colored_handler.go implements a colored slog handler for terminal output.
// It adds ANSI color codes to log levels for better visibility in terminals.
//
// colored_handler.goはターミナル出力用のカラーslogハンドラーを実装します。
// ターミナルでの視認性向上のためにログレベルにANSIカラーコードを追加します。
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// ANSI color codes for terminal output.
// ターミナル出力用のANSIカラーコード。
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// ColoredHandler is a slog.Handler that outputs colored log messages to terminals.
// Colors are only applied when the output is a TTY.
//
// ColoredHandlerはターミナルにカラーログメッセージを出力するslog.Handlerです。
// 出力がTTYの場合のみ色が適用されます。
type ColoredHandler struct {
	out     io.Writer
	level   slog.Level
	colored bool
	mu      sync.Mutex
}

// NewColoredHandler creates a new ColoredHandler.
// If out is a terminal, colors will be enabled.
//
// NewColoredHandlerは新しいColoredHandlerを作成します。
// outがターミナルの場合、色が有効になります。
func NewColoredHandler(out io.Writer, level slog.Level) *ColoredHandler {
	colored := false
	if f, ok := out.(*os.File); ok {
		// Check if the output is a terminal
		// 出力がターミナルかどうかをチェック
		fi, err := f.Stat()
		if err == nil {
			colored = (fi.Mode() & os.ModeCharDevice) != 0
		}
	}

	return &ColoredHandler{
		out:     out,
		level:   level,
		colored: colored,
	}
}

// Enabled implements slog.Handler.Enabled.
// EnabledはslogHandler.Enabledを実装します。
func (h *ColoredHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle implements slog.Handler.Handle.
// It formats and writes the log record with colors.
//
// HandleはslogHandler.Handleを実装します。
// ログレコードをフォーマットして色付きで書き込みます。
func (h *ColoredHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Format timestamp
	// タイムスタンプをフォーマット
	timeStr := r.Time.Format(time.DateTime)

	// Get level string and color
	// レベル文字列と色を取得
	levelStr, levelColor := h.levelInfo(r.Level)

	// Build the log line
	// ログ行を構築
	var line string
	if h.colored {
		// Colored output: time in gray, level in color, message in default
		// カラー出力: 時間はグレー、レベルは色付き、メッセージはデフォルト
		line = fmt.Sprintf("%s%s%s %s%-5s%s %s",
			colorGray, timeStr, colorReset,
			levelColor, levelStr, colorReset,
			r.Message,
		)
	} else {
		// Plain output
		// プレーン出力
		line = fmt.Sprintf("%s %-5s %s", timeStr, levelStr, r.Message)
	}

	// Append attributes
	// 属性を追加
	r.Attrs(func(a slog.Attr) bool {
		if h.colored {
			line += fmt.Sprintf(" %s%s%s=%v", colorCyan, a.Key, colorReset, a.Value)
		} else {
			line += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		}
		return true
	})

	line += "\n"

	_, err := h.out.Write([]byte(line))
	return err
}

// WithAttrs implements slog.Handler.WithAttrs.
// WithAttrsはslog.Handler.WithAttrsを実装します。
func (h *ColoredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, we don't support pre-set attrs in this handler
	// シンプルさのため、このハンドラーでは事前設定された属性をサポートしません
	return h
}

// WithGroup implements slog.Handler.WithGroup.
// WithGroupはslog.Handler.WithGroupを実装します。
func (h *ColoredHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we don't support groups in this handler
	// シンプルさのため、このハンドラーではグループをサポートしません
	return h
}

// levelInfo returns the level string and color for a given slog.Level.
// levelInfoは指定されたslog.Levelのレベル文字列と色を返します。
func (h *ColoredHandler) levelInfo(level slog.Level) (string, string) {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG", colorBlue
	case level < slog.LevelWarn:
		return "INFO", colorGreen
	case level < slog.LevelError:
		return "WARN", colorYellow
	default:
		return "ERROR", colorRed
	}
}

// multiHandler is a slog.Handler that writes to multiple handlers.
// It allows logging to both stdout (colored) and file (plain) simultaneously.
//
// multiHandlerは複数のハンドラーに書き込むslog.Handlerです。
// stdout（カラー）とファイル（プレーン）の両方に同時にログを出力できます。
type multiHandler struct {
	handlers []slog.Handler
}

// Enabled implements slog.Handler.Enabled.
// Returns true if any handler is enabled for the level.
//
// EnabledはslogHandler.Enabledを実装します。
// いずれかのハンドラーがレベルに対して有効な場合にtrueを返します。
func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements slog.Handler.Handle.
// Writes to all handlers.
//
// HandleはslogHandler.Handleを実装します。
// すべてのハンドラーに書き込みます。
func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs implements slog.Handler.WithAttrs.
// WithAttrsはslog.Handler.WithAttrsを実装します。
func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

// WithGroup implements slog.Handler.WithGroup.
// WithGroupはslog.Handler.WithGroupを実装します。
func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
