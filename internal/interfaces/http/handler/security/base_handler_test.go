package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondErrorRedactsServerErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondError(rec, http.StatusInternalServerError, "postgres://user:secret@db/internal SQL failure")
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "SQL") {
		t.Fatalf("internal detail leaked: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Fatalf("generic error missing: %s", rec.Body.String())
	}
}

func TestRespondErrorPreservesClientError(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondError(rec, http.StatusBadRequest, "invalid field")
	if !strings.Contains(rec.Body.String(), "invalid field") {
		t.Fatalf("client-safe error changed: %s", rec.Body.String())
	}
}
