package handler

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// itemCodePathParam decodes a public code after chi has matched the route.
// Chi uses RawPath when present; otherwise its parameters are already decoded.
// Unescaping only once preserves literal percent sequences such as %2F.
func itemCodePathParam(r *http.Request, name string) string {
	code := chi.URLParam(r, name)
	if r.URL.RawPath != "" {
		if decoded, err := url.PathUnescape(code); err == nil {
			return decoded
		}
	}
	return code
}
