package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLoggerMiddleware is the access log middleware. For every HTTP request
// it records the following fields in a single structured JSON line:
//
//	request_id   – correlation ID (set by CorrelationMiddleware)
//	method       – HTTP verb
//	path         – URL path (without query string)
//	query        – URL query string, if any
//	status       – HTTP status code written by the handler
//	latency_ms   – wall-clock time from first byte received to response flushed
//	ip           – client IP (respects X-Real-IP / X-Forwarded-For via chi RealIP)
//	user_agent   – User-Agent header
//	bytes        – response body size in bytes
//
// The Authorization header is sanitized automatically (scheme kept, token
// redacted), so it is safe to log even on shared infrastructure.
//
// It also stores the per-request logger (enriched with request_id) in the
// context so handlers can retrieve it via logger.FromContext(ctx).
func RequestLoggerMiddleware(base *applogger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the ResponseWriter so we can capture status + bytes.
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			// Build a per-request logger pre-populated with the correlation ID.
			reqLogger := base.WithContext(r.Context())

			// Store the enriched logger in the context for downstream use.
			ctx := applogger.WithLogger(r.Context(), reqLogger)

			next.ServeHTTP(ww, r.WithContext(ctx))

			// --- Access log -------------------------------------------------
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK // chi default when no WriteHeader call
			}

			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}

			reqLogger.Slog().LogAttrs(
				r.Context(),
				level,
				"request",
				slog.String("request_id", applogger.CorrelationIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", status),
				slog.Int64("latency_ms", time.Since(start).Milliseconds()),
				slog.String("ip", realIP(r)),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("bytes", ww.BytesWritten()),
			)
		})
	}
}

// realIP extracts the real client IP, honouring the X-Real-IP / X-Forwarded-For
// headers that chi's RealIP middleware sets on RemoteAddr.
func realIP(r *http.Request) string {
	// chi's RealIP middleware already rewrites RemoteAddr.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
