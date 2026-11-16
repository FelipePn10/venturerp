package cnpj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCnpjaProviderParsesCompany(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/office/52454668000102" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taxId":"52454668000102","alias":"TECNOFER","company":{"name":"TECNOFER LTDA"},"status":{"text":"Ativa"},"address":{"zip":"86975-000","street":"Rua Jacob Evaldo Stadler","number":"83","district":"Parque Industrial III","city":"Mandaguari","state":"PR"},"emails":[{"address":"a@b.com"}],"phones":[{"area":"44","number":"98904502"}],"mainActivity":{"id":2511000,"text":"Fabricação de estruturas metálicas"}}`))
	}))
	defer srv.Close()

	p := &cnpjaProvider{base: srv.URL, http: &http.Client{Timeout: time.Second}}
	company, err := p.Lookup(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}
	if company.LegalName != "TECNOFER LTDA" {
		t.Fatalf("LegalName = %q", company.LegalName)
	}
	if company.Address.City != "Mandaguari" || company.Address.UF != "PR" {
		t.Fatalf("unexpected address: %+v", company.Address)
	}
	// CNPJá does not expose Inscrição Estadual — it must stay empty here.
	if len(company.StateRegistrations) != 0 || company.PrimaryStateRegistration() != "" {
		t.Fatalf("cnpja must not report state registrations: %+v", company.StateRegistrations)
	}
}

func TestCapitalSocial(t *testing.T) {
	casos := map[string]string{"150000.00": "150000.00", "1.500.000,50": "1500000.50", "1000": "1000.00", "": "", "abc": "", "-5": ""}
	for in, want := range casos {
		if got := capitalSocial(in); got != want {
			t.Errorf("capitalSocial(%q) = %q, quer %q", in, got, want)
		}
	}
	var f flexNumber
	if err := f.UnmarshalJSON([]byte(`12.5`)); err != nil || capitalSocial(string(f)) != "12.50" {
		t.Fatalf("número JSON: %q %v", f, err)
	}
}
