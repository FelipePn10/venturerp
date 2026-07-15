package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appversion "github.com/FelipePn10/panossoerp/internal/version"
)

func TestVersionHandler(t *testing.T) {
	originalVersion, originalMinClient := appversion.Version, appversion.MinClient
	t.Cleanup(func() { appversion.Version, appversion.MinClient = originalVersion, originalMinClient })
	appversion.Version, appversion.MinClient = "v1.4.0", "1.2.0"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	versionHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var got appversion.Info
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Version != "1.4.0" || got.MinClient != "1.2.0" {
		t.Fatalf("response = %#v", got)
	}
}
