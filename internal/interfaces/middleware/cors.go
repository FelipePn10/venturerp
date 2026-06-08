package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a cross-origin middleware. The desktop client (Electron/Tauri,
// Chromium-based) sends an Origin header, so even a "native" app needs CORS to
// be answered correctly.
//
//   - allowedOrigins: exact origins permitted (e.g. "http://localhost:5173",
//     "app://."). Compared case-sensitively against the request Origin.
//   - allowAll: when true, the request Origin is reflected back, which permits
//     any caller. Intended for development only.
//
// Credentials are allowed, so the literal "*" is never sent (the spec forbids
// "*" together with Allow-Credentials); the concrete Origin is echoed instead.
func CORS(allowedOrigins []string, allowAll bool) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || allowed[origin]) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
				h.Set("Access-Control-Expose-Headers", "X-Request-ID")
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Max-Age", "300")
			}

			// Short-circuit the CORS preflight before it reaches business routes.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
