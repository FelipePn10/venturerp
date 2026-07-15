package system_update_uc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appversion "github.com/FelipePn10/panossoerp/internal/version"
)

func TestManagerRequestsLatestReleaseAndLocksQueue(t *testing.T) {
	original := appversion.Version
	appversion.Version = "1.0.0"
	t.Cleanup(func() { appversion.Version = original })

	dir := t.TempDir()
	manager := NewManager(dir, "https://releases.test/latest", releaseClient("v1.2.0"))
	status, err := manager.Request(context.Background(), "")
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if status.State != StateQueued || status.TargetVersion != "1.2.0" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if _, err := os.Stat(filepath.Join(dir, "request.json")); err != nil {
		t.Fatalf("request was not persisted: %v", err)
	}
	if _, err := manager.Request(context.Background(), "1.3.0"); !errors.Is(err, ErrUpdateInProgress) {
		t.Fatalf("second Request() error = %v, want ErrUpdateInProgress", err)
	}
}

func TestManagerStatusReportsAvailableRelease(t *testing.T) {
	original := appversion.Version
	appversion.Version = "1.0.0"
	t.Cleanup(func() { appversion.Version = original })

	status, err := NewManager(t.TempDir(), "https://releases.test/latest", releaseClient("v1.1.0")).Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.UpdateAvailable || status.LatestVersion != "1.1.0" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func releaseClient(tag string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		var body strings.Builder
		_ = json.NewEncoder(&body).Encode(map[string]string{"tag_name": tag})
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String())), Header: make(http.Header)}, nil
	})}
}

func TestManagerRejectsInvalidAndNonIncreasingVersions(t *testing.T) {
	original := appversion.Version
	appversion.Version = "2.0.0"
	t.Cleanup(func() { appversion.Version = original })

	manager := NewManager(t.TempDir(), "unused", nil)
	for _, version := range []string{"latest", "1.9.9", "2.0.0"} {
		if _, err := manager.Request(context.Background(), version); !errors.Is(err, ErrInvalidVersion) {
			t.Errorf("Request(%q) error = %v, want ErrInvalidVersion", version, err)
		}
	}
}
