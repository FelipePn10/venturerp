package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientVersionCompatibility(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	tests := []struct {
		name       string
		minClient  string
		path       string
		origin     string
		version    string
		wantStatus int
	}{
		{name: "development does not enforce", minClient: "dev", path: "/api/orders", origin: "http://tauri.localhost", wantStatus: http.StatusNoContent},
		{name: "version endpoint stays public", minClient: "1.2.0", path: "/api/version", origin: "http://tauri.localhost", wantStatus: http.StatusNoContent},
		{name: "current desktop is accepted", minClient: "1.2.0", path: "/api/orders", version: "1.2.0", wantStatus: http.StatusNoContent},
		{name: "newer desktop is accepted", minClient: "1.2.0", path: "/api/orders", version: "1.3.0", wantStatus: http.StatusNoContent},
		{name: "outdated desktop is rejected", minClient: "1.2.0", path: "/api/orders", version: "1.1.9", wantStatus: http.StatusUpgradeRequired},
		{name: "malformed desktop version is rejected", minClient: "1.2.0", path: "/api/orders", version: "unknown", wantStatus: http.StatusUpgradeRequired},
		{name: "legacy tauri desktop is accepted during migration", minClient: "1.1.9", path: "/api/orders", origin: "https://tauri.localhost", wantStatus: http.StatusNoContent},
		{name: "legacy tauri desktop is rejected after migration", minClient: "1.2.0", path: "/api/orders", origin: "https://tauri.localhost", wantStatus: http.StatusUpgradeRequired},
		{name: "server integration without header remains accepted", minClient: "1.2.0", path: "/api/orders", wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ClientVersionCompatibility(tt.minClient)(ok)
			request := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			if tt.version != "" {
				request.Header.Set(clientVersionHeader, tt.version)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUpgradeRequired && !strings.Contains(response.Body.String(), `"min_client":"1.2.0"`) {
				t.Fatalf("response does not expose min_client: %s", response.Body.String())
			}
		})
	}
}
