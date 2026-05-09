// Package logger provides a structured, JSON-first logger built on top of
// the standard library log/slog. It adds:
//   - Fatal level (logs ERROR then os.Exit(1))
//   - Context-aware helpers (automatically attaches request_id, user_id)
//   - ENV-driven level selection (LOG_LEVEL=debug|info|warn|error)
//   - AddSource in DEBUG so every message shows file:line
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger is a thin wrapper around *slog.Logger that adds Fatal and
// context-aware helpers while remaining fully compatible with the stdlib.
type Logger struct {
	inner *slog.Logger
}

// New builds a Logger. It always emits JSON so that production stacks
// (ELK, Grafana Loki, Datadog) can ingest logs without extra config.
//
// In development, pipe the output through humanlog for pretty printing:
//
//	go run ./api/... | humanlog
//	# install once: go install github.com/humanlogio/humanlog/cmd/humanlog@latest
//
// Or with jq (no install needed if jq is available):
//
//	go run ./api/... | jq .
func New(levelStr string) *Logger {
	level := parseLevel(levelStr)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // show file:line only in debug
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Normalise level names to lowercase for consistency.
			if a.Key == slog.LevelKey {
				lv := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(strings.ToLower(lv.String()))
			}
			return a
		},
	}

	return &Logger{inner: slog.New(slog.NewJSONHandler(os.Stdout, opts))}
}

// --- Core log methods -------------------------------------------------------

func (l *Logger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

// Fatal logs at ERROR level and terminates the process.
// Use only for unrecoverable startup failures (config missing, DB unreachable).
func (l *Logger) Fatal(msg string, args ...any) {
	l.inner.Error(msg, args...)
	os.Exit(1)
}

// --- Builder helpers --------------------------------------------------------

// With returns a new Logger pre-populated with the given key-value pairs,
// following the same slog.Any convention.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{inner: l.inner.With(args...)}
}

// WithContext returns a new Logger enriched with values extracted from ctx:
//   - request_id  (set by CorrelationMiddleware)
//   - user_id     (set by JWT middleware, when present)
func (l *Logger) WithContext(ctx context.Context) *Logger {
	enriched := l.inner

	if id := CorrelationIDFromContext(ctx); id != "" {
		enriched = enriched.With(slog.String("request_id", id))
	}
	if uid := UserIDFromContext(ctx); uid != "" {
		enriched = enriched.With(slog.String("user_id", uid))
	}

	return &Logger{inner: enriched}
}

// Slog returns the underlying *slog.Logger for integration with packages
// that accept the standard interface.
func (l *Logger) Slog() *slog.Logger { return l.inner }

// --- stdlib slog.Handler interface passthrough (used by chi Logger) ---------

// FromContext retrieves the per-request Logger stored by RequestLoggerMiddleware.
// Falls back to a default INFO logger if none is present (safe for use in tests).
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*Logger); ok && l != nil {
		return l
	}
	return New("info")
}

// --- Internal helpers --------------------------------------------------------

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
