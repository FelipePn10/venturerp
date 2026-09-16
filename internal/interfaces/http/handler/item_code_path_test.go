package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestStructurePublicCodesThroughMountedRouter(t *testing.T) {
	for _, code := range []string{"MP/01-2", "/MP/01/", "RN-01001", "123", "MP%2F01", "MP%/01", "MP+01", "MP 01"} {
		for _, route := range []string{"/search/{code}", "/structure/resolve/{code}", "/structure/{code}/children", "/structure/{code}/history", "/structure/{code}/configuration-check", "/structure/{code}/configurator", "/structure/{code}/configurator/apply", "/structure/{code}/{childCode}"} {
			t.Run(code+route, func(t *testing.T) {
				router := chi.NewRouter()
				router.Route("/api/items", func(r chi.Router) {
					r.Get(route, func(w http.ResponseWriter, req *http.Request) {
						if got := itemCodePathParam(req, "code"); got != code {
							t.Errorf("code = %q, want %q", got, code)
						}
						if chi.URLParam(req, "childCode") != "" {
							if got := itemCodePathParam(req, "childCode"); got != "CH/02-3" {
								t.Errorf("child = %q", got)
							}
						}
						w.WriteHeader(http.StatusNoContent)
					})
				})
				path := strings.ReplaceAll(route, "{code}", url.PathEscape(code))
				path = strings.ReplaceAll(path, "{childCode}", url.PathEscape("CH/02-3"))
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/items"+path, nil))
				if rec.Code != http.StatusNoContent {
					t.Fatalf("status = %d", rec.Code)
				}
			})
		}
	}
}
