package logger

import (
	"net/http"
	"strings"
)

// sensitiveHeaders lists HTTP header names whose values must never appear in logs.
// Keys must be in canonical (http.CanonicalHeaderKey) format.
var sensitiveHeaders = map[string]struct{}{
	"Cookie":              {},
	"Set-Cookie":          {},
	"X-Api-Key":           {},
	"X-Auth-Token":        {},
	"Proxy-Authorization": {},
}

// SanitizeHeaders returns a copy of the headers safe for logging:
//   - Authorization: retains the scheme ("Bearer", "Basic") but redacts the credential.
//   - All entries in sensitiveHeaders are replaced with "[REDACTED]".
func SanitizeHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for key, values := range h {
		canonical := http.CanonicalHeaderKey(key)
		joined := strings.Join(values, ", ")

		if _, sensitive := sensitiveHeaders[canonical]; sensitive {
			out[canonical] = "[REDACTED]"
			continue
		}

		if canonical == "Authorization" {
			out[canonical] = redactAuthHeader(joined)
			continue
		}

		out[canonical] = joined
	}
	return out
}

// redactAuthHeader keeps the authentication scheme but hides the credential.
//
//	"Bearer eyJhbGc..." → "Bearer [REDACTED]"
//	"Basic dXNlcjpw..."  → "Basic [REDACTED]"
//	"Token abc123"       → "Token [REDACTED]"
func redactAuthHeader(value string) string {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 2 {
		return parts[0] + " [REDACTED]"
	}
	return "[REDACTED]"
}

// sensitiveBodyFields contains JSON field names whose values should never be logged.
// Comparison is case-insensitive.
var sensitiveBodyFields = map[string]struct{}{
	"password":      {},
	"senha":         {},
	"token":         {},
	"secret":        {},
	"access_token":  {},
	"refresh_token": {},
	"card_number":   {},
	"numero_cartao": {},
	"cvv":           {},
	"pin":           {},
	"private_key":   {},
	"chave_privada": {},
	"api_key":       {},
	"authorization": {},
}

// IsSensitiveField returns true if the field name matches a known sensitive key.
func IsSensitiveField(name string) bool {
	_, ok := sensitiveBodyFields[strings.ToLower(strings.TrimSpace(name))]
	return ok
}
