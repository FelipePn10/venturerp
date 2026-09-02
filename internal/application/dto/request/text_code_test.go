package request

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommercialPricingItemCodeAcceptsCanonicalTextAndLegacyNumber(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{name: "texto canônico", body: `{"item_code":"SKU-A01","price":10}`, want: "SKU-A01"},
		{name: "número legado", body: `{"item_code":123,"price":10}`, want: "123"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var dto CreateSalesTablePriceDTO
			if err := json.Unmarshal([]byte(test.body), &dto); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if dto.ItemCode.String() != test.want {
				t.Fatalf("item_code = %q, want %q", dto.ItemCode, test.want)
			}
		})
	}
}

func TestTextCodeCountsLegacyNumericCompatibility(t *testing.T) {
	before := LegacyNumericTextCodeCount()
	var code TextCode
	if err := json.Unmarshal([]byte(`12345`), &code); err != nil {
		t.Fatal(err)
	}
	if got := LegacyNumericTextCodeCount(); got != before+1 {
		t.Fatalf("legacy numeric telemetry = %d, want %d", got, before+1)
	}
}

func TestCommercialPricingItemCodesAcceptMixedCompatibilityList(t *testing.T) {
	var dto GenerateSalesTablePricesDTO
	if err := json.Unmarshal([]byte(`{"item_codes":["SKU-A01",123]}`), &dto); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(dto.ItemCodes) != 2 || dto.ItemCodes[0].String() != "SKU-A01" || dto.ItemCodes[1].String() != "123" {
		t.Fatalf("item_codes = %#v", dto.ItemCodes)
	}
}

func TestCommercialPricingItemCodeRejectsInvalidValuesInPortuguese(t *testing.T) {
	for _, body := range []string{
		`{"item_code":1.5}`,
		`{"item_code":false}`,
		`{"item_code":""}`,
		`{"item_code":null}`,
	} {
		var dto CreateSalesTablePriceDTO
		err := json.Unmarshal([]byte(body), &dto)
		if err == nil || (!strings.Contains(err.Error(), "código do item") && !strings.Contains(err.Error(), "item")) {
			t.Fatalf("Unmarshal(%s) error = %v", body, err)
		}
	}
}

func TestCommercialPolicySpecificItemAcceptsCanonicalAndLegacyCode(t *testing.T) {
	for _, body := range []string{`{"item_code":"SKU-A01"}`, `{"item_code":123}`} {
		var dto CommercialPolicySpecificItemDTO
		if err := json.Unmarshal([]byte(body), &dto); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", body, err)
		}
		if dto.ItemCode == nil || dto.ItemCode.String() == "" {
			t.Fatalf("item_code não normalizado para %s", body)
		}
	}
}
