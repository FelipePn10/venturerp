package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Binary/download responses must keep streaming and optional HTTP interfaces.
func TestItemCompatibilityBypassesBinaryResponses(t *testing.T) {
	for _, path := range []string{"/api/reports/export", "/api/orders/1/attachments/2/download"} {
		if !bypassItemResponseTranslation(httptest.NewRequest("GET", path, nil)) {
			t.Fatalf("nao ignorou resposta binaria: %s", path)
		}
	}
}

func TestRotaDeItemEhReconhecidaPelaPosicaoDoSegmento(t *testing.T) {
	// Cada caso é um caminho concreto e a posição do segmento que carrega o
	// código do item. -1 significa "esta rota não referencia o cadastro de item".
	casos := []struct {
		caminho  string
		esperado int
	}{
		{"/api/sales-order/items/77", -1},
		{"/api/consumer-service/calls/checklist/77", -1},
		{"/api/items/classifications/masks/9", -1},
		{"/api/items/search/MP-CH-3MM", 3},
		{"/api/stock/movements/item/MP-CH-3MM", 4},
		{"/api/stock/balances/atp/MP-CH-3MM", 4},
		{"/api/items/structure/resolve/PA-CHASSI", 4},
		{"/api/quality/plans/by-item/MP-CH-3MM", 4},
		{"/api/standard-cost/items/MP-CH-3MM", 3},
		{"/api/mrp-calculation/profile/MP-CH-3MM/PLAN1", 3},
		{"/api/mrp-calculation/configured-rules/MP-CH-3MM", 3},
		{"/api/item-calendar-promise/MP-CH-3MM/_/2026/9", 2},
		{"/api/financial/relatorios/ficha-tecnica/MP-CH-3MM", 4},
		{"/api/customers/support/sales-tables/5/prices/MP-CH-3MM", 6},
	}
	for _, caso := range casos {
		partes := strings.Split(strings.Trim(caso.caminho, "/"), "/")
		if got := itemPathSegmentIndex(partes); got != caso.esperado {
			t.Errorf("%s: esperava posição %d, veio %d", caso.caminho, caso.esperado, got)
		}
	}
}

func TestItemClassificationsIsAStaticPath(t *testing.T) {
	if !isStaticItemsPath("classifications") {
		t.Fatal("/api/items/classifications foi tratado como código comercial de item")
	}
	if isStaticItemsPath("4853") || isStaticItemsPath("TEA452-0") {
		t.Fatal("código comercial foi tratado como segmento estático")
	}
}


func TestVENT0210PathsPreservePublicItemCode(t *testing.T) {
	for _, path := range []string{
		"/api/items/search/RN-01001",
		"/api/items/structure/resolve/RN-01001",
	} {
		req := httptest.NewRequest("GET", path, nil)
		if !nativeItemBusinessCodePath(req) {
			t.Errorf("%s deveria preservar o código comercial para o handler", path)
		}
	}
}

func TestLegacyItemPathStillRequiresTranslation(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/stock/balances/atp/RN-01001", nil)
	if nativeItemBusinessCodePath(req) {
		t.Fatal("rota legada de estoque não pode ignorar a tradução para a chave interna")
	}
}
