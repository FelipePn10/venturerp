package cnpj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

func TestFallbackUsesPrimaryWhenAvailable(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cnpjwsFixture))
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("secondary provider must not be called when primary succeeds")
	}))
	defer secondary.Close()

	p := New(Config{BaseURL: primary.URL, FallbackBaseURL: secondary.URL, Timeout: time.Second})
	company, err := p.Lookup(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if company.Source != "cnpj.ws" {
		t.Fatalf("Source = %q, want cnpj.ws", company.Source)
	}
	if company.PrimaryStateRegistration() != "9103144679" {
		t.Fatalf("PrimaryStateRegistration() = %q", company.PrimaryStateRegistration())
	}
}

func TestFallbackSwitchesToSecondaryWhenPrimaryUnavailable(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/office/52454668000102" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taxId":"52454668000102","alias":"TECNOFER","company":{"name":"TECNOFER LTDA"},"status":{"text":"Ativa"},"address":{"city":"Mandaguari","state":"PR"}}`))
	}))
	defer secondary.Close()

	p := New(Config{BaseURL: primary.URL, FallbackBaseURL: secondary.URL, Timeout: time.Second})
	company, err := p.Lookup(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if company.Source != "cnpja" {
		t.Fatalf("Source = %q, want cnpja", company.Source)
	}
	if company.LegalName != "TECNOFER LTDA" {
		t.Fatalf("LegalName = %q", company.LegalName)
	}
}

func TestFallbackDoesNotSwitchOnNotFound(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("secondary provider must not be called on ErrNotFound")
	}))
	defer secondary.Close()

	p := New(Config{BaseURL: primary.URL, FallbackBaseURL: secondary.URL, Timeout: time.Second})
	if _, err := p.Lookup(context.Background(), "52.454.668/0001-02"); err != service.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestFallbackReturnsSecondaryErrorWhenBothFail(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "also down", http.StatusBadGateway)
	}))
	defer secondary.Close()

	p := New(Config{BaseURL: primary.URL, FallbackBaseURL: secondary.URL, Timeout: time.Second})
	if _, err := p.Lookup(context.Background(), "52.454.668/0001-02"); err != service.ErrUnavailable {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

var _ service.Provider = (*fallbackProvider)(nil)
var _ service.Provider = (*cnpjwsProvider)(nil)
var _ service.Provider = (*cnpjaProvider)(nil)
