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

func TestItemReferencePathDoesNotConfuseOrderLineWithItem(t *testing.T) {
	for _, pattern := range []string{
		"/api/sales-orders/items/{itemCode}",
		"/api/sales-quotations/items/{itemCode}/cancel",
		"/api/consumer-service/calls/checklist/{itemCode}",
	} {
		if isItemReferencePath(pattern, "itemCode") {
			t.Fatalf("identificador de linha confundido com item: %s", pattern)
		}
	}
	for _, pattern := range []string{
		"/api/items/search/{code}",
		"/api/stock/movements/item/{itemCode}",
		"/api/configurator/items/{itemCode}/rules",
		"/api/items/structure/resolve/{itemCode}",
		"/api/quality/plans/by-item/{itemCode}",
		"/api/standard-cost/items/{itemCode}",
		"/api/mrp-calculation/profile/{item_code}/{plan_code}",
		"/api/item-calendar-promise/{item_code}/{mask}/{year}/{month}",
		"/api/financial/relatorios/ficha-tecnica/{item_code}",
	} {
		key := "itemCode"
		if pattern == "/api/items/search/{code}" {
			key = "code"
		} else if strings.Contains(pattern, "{item_code}") {
			key = "item_code"
		}
		if !isItemReferencePath(pattern, key) {
			t.Fatalf("rota de item nao reconhecida: %s", pattern)
		}
	}
}
