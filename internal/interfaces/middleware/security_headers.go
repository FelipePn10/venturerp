package middleware

import "net/http"

// SecurityHeaders sets conservative, framework-agnostic response headers. This
// is a JSON API consumed by a desktop client, so the set is intentionally
// minimal — no CSP (there is no HTML surface) but the headers that defend
// against MIME sniffing, clickjacking and referrer leakage.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// MaxBodyBytes caps the size of request bodies to guard against memory
// exhaustion from oversized or malicious payloads. A value <= 0 disables the
// cap. Handlers reading past the limit receive an error from the body reader,
// which they surface as 400/413.
func MaxBodyBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if n > 0 && r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}
