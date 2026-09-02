package request

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommercialItemDTOsAcceptCanonicalAndLegacyCodes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "pedido textual", raw: `{"item_code":"TEA452-0"}`, want: "TEA452-0"},
		{name: "pedido numérico temporário", raw: `{"item_code":452}`, want: "452"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var dto CreateSalesOrderItemDTO
			if err := json.Unmarshal([]byte(tc.raw), &dto); err != nil {
				t.Fatalf("decodificar pedido: %v", err)
			}
			if dto.ItemCode.String() != tc.want {
				t.Fatalf("item_code = %q; esperado %q", dto.ItemCode, tc.want)
			}
			var quotation CreateSalesQuotationItemDTO
			if err := json.Unmarshal([]byte(tc.raw), &quotation); err != nil {
				t.Fatalf("decodificar orçamento: %v", err)
			}
			if quotation.ItemCode.String() != tc.want {
				t.Fatalf("item_code do orçamento = %q; esperado %q", quotation.ItemCode, tc.want)
			}
		})
	}
}

func TestCommercialItemDTORejectsInvalidCodeInPortuguese(t *testing.T) {
	var dto CreateSalesOrderItemDTO
	err := json.Unmarshal([]byte(`{"item_code":1.5}`), &dto)
	if err == nil || !strings.Contains(err.Error(), "código do item") {
		t.Fatalf("erro inválido: %v", err)
	}
}
