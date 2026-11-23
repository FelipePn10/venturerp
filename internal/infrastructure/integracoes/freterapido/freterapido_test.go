package freterapido

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

func TestCotacaoERastreio(t *testing.T) {
	var cot map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/quote/simulate":
			_ = json.NewDecoder(r.Body).Decode(&cot)
			_, _ = w.Write([]byte(`{"dispatchers":[{"offers":[
				{"carrier":{"name":"Rodonaves","registered_number":"1"},"service":"Normal","delivery_time":{"days":5,"estimated_date":"2026-11-30"},"final_price":420.10},
				{"carrier":{"name":"Braspress","registered_number":"2"},"service":"Normal","delivery_time":{"days":4,"estimated_date":"2026-11-28"},"final_price":390.00}]}]}`))
		case "/api/external/embarcador/v1/quotes/occurrences/invoices":
			if r.URL.Query().Get("token") != "tok" || !strings.Contains(r.URL.Query().Get("invoices"), "41261") {
				w.WriteHeader(401)
				return
			}
			_, _ = w.Write([]byte(`[{"FR11111111111":{"occurrences":[
				{"code":2,"name":"Em trânsito","occurrence_date":"2026-11-22 10:00:00"},
				{"code":1,"name":"Coletado","occurrence_date":"2026-11-21 16:30:00","message":"Coleta feita"}]}}]`))
		}
	}))
	defer srv.Close()
	c := New(Config{Token: "tok", PlatformCode: "PC"})
	c.baseCotacao, c.baseRast = srv.URL, srv.URL
	of, err := c.Cotar(context.Background(), ports.PedidoDeCotacao{CNPJEmbarcador: "11.444.777/0001-61", CEPOrigem: "87000-000", CEPDestino: "01001000", DestinatarioPJ: true,
		ValorNota: decimal.NewFromInt(5000), Volumes: []ports.VolumeDeFrete{{Quantidade: 2, Altura: decimal.RequireFromString("0.2"), Largura: decimal.NewFromInt(1), Comp: decimal.NewFromInt(3), PesoKg: decimal.NewFromInt(85), ValorUnitario: decimal.NewFromInt(2500)}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(of) != 2 || of[0].Transportadora != "Braspress" || !of[0].Preco.Equal(decimal.NewFromInt(390)) {
		t.Fatalf("ofertas (mais barata primeiro): %+v", of)
	}
	sh := cot["shipper"].(map[string]any)
	if sh["registered_number"] != "11444777000161" || sh["token"] != "tok" || cot["recipient"].(map[string]any)["zipcode"].(float64) != 1001000 {
		t.Fatalf("corpo: %+v", cot)
	}
	ev, err := c.Rastrear(context.Background(), "", "41261012345678000190550010000000011000000010")
	if err != nil || len(ev) != 2 || !strings.HasPrefix(ev[0].Situacao, "Coletado") {
		t.Fatalf("rastreio (em ordem): %+v err=%v", ev, err)
	}
}
