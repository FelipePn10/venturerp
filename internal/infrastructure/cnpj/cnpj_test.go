package cnpj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderParsesCompanyAndStateRegistration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/office/52454668000102" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taxId":"52454668000102","alias":"TECNOFER","company":{"name":"TECNOFER LTDA"},"status":{"text":"Ativa"},"address":{"zip":"86975-000","city":"Mandaguari","state":"PR"},"registrations":[{"state":"PR","number":"9103144679","enabled":true}]}`))
	}))
	defer srv.Close()
	company, err := New(Config{BaseURL: srv.URL, Timeout: time.Second}).Lookup(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if company.LegalName != "TECNOFER LTDA" || company.PrimaryStateRegistration() != "9103144679" {
		t.Fatalf("unexpected company: %+v", company)
	}
}
