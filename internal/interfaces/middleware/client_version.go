package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	clientVersionHeader  = "X-ERP-Client-Version"
	lastHeaderlessClient = "1.1.9"
)

type upgradeRequiredResponse struct {
	Error         string `json:"error"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	ClientVersion string `json:"client_version,omitempty"`
	MinClient     string `json:"min_client"`
}

// ClientVersionCompatibility prevents an already-open desktop from changing
// data after min_client is raised. Requests without the desktop header remain
// accepted for server-to-server integrations; legacy Tauri clients are
// identified by their Origin and must upgrade as well.
func ClientVersionCompatibility(minClient string) func(http.Handler) http.Handler {
	minClient = normalizeVersion(minClient)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if minClient == "dev" || compatibilityExempt(r) {
				next.ServeHTTP(w, r)
				return
			}

			clientVersion := strings.TrimSpace(r.Header.Get(clientVersionHeader))
			if clientVersion == "" {
				// Desktop 1.1.9 and earlier predate this header. Keep them working
				// during the first backend rollout; once min_client is raised above
				// that migration boundary, legacy Tauri sessions are blocked too.
				legacyTauriMustUpgrade := isTauriOrigin(r.Header.Get("Origin")) && compareVersions(minClient, lastHeaderlessClient) > 0
				if !legacyTauriMustUpgrade {
					next.ServeHTTP(w, r)
					return
				}
			}

			if compareVersions(clientVersion, minClient) >= 0 {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)
			_ = json.NewEncoder(w).Encode(upgradeRequiredResponse{
				Error:         "atualizacao obrigatoria",
				Code:          "CLIENT_UPGRADE_REQUIRED",
				Message:       "Esta versao do VentureERP nao pode mais realizar operacoes. Atualize para continuar.",
				ClientVersion: clientVersion,
				MinClient:     minClient,
			})
		})
	}
}

func compatibilityExempt(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}
	switch r.URL.Path {
	case "/api/version", "/health", "/health/live", "/health/ready", "/metrics":
		return true
	default:
		return false
	}
}

func isTauriOrigin(origin string) bool {
	switch strings.ToLower(strings.TrimSpace(origin)) {
	case "tauri://localhost", "http://tauri.localhost", "https://tauri.localhost":
		return true
	default:
		return false
	}
}

func compareVersions(left, right string) int {
	a, okA := parseVersion(left)
	b, okB := parseVersion(right)
	if !okA || !okB {
		return -1
	}
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(value string) ([3]int, bool) {
	var result [3]int
	value = strings.SplitN(normalizeVersion(value), "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, false
	}
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return result, false
		}
		result[i] = number
	}
	return result, true
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return "dev"
	}
	return value
}
