package request

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestThirdPartyPriceItemCodeContract(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
		wantErr bool
	}{
		{name: "textual", payload: `{"item_code":"TEA452-0"}`, want: "TEA452-0"},
		{name: "temporary numeric compatibility", payload: `{"item_code":452}`, want: "452"},
		{name: "invalid decimal", payload: `{"item_code":45.2}`, wantErr: true},
		{name: "invalid empty", payload: `{"item_code":""}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dto ThirdPartyPriceDTO
			err := json.Unmarshal([]byte(tt.payload), &dto)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected invalid item_code to be rejected")
				}
				return
			}
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if dto.ItemCode.String() != tt.want {
				t.Fatalf("item_code=%q, want %q", dto.ItemCode, tt.want)
			}
		})
	}
}

func TestSupplyIdentityFieldsAreNotAcceptedFromJSON(t *testing.T) {
	var order CreatePurchaseOrderDTO
	if err := json.Unmarshal([]byte(`{"enterprise_code":999,"created_by":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`), &order); err != nil {
		t.Fatal(err)
	}
	if order.EnterpriseCode != 0 || order.CreatedBy != uuid.Nil {
		t.Fatalf("purchase order accepted caller-controlled identity: %+v", order)
	}

	var params UpsertSupplierParametersDTO
	if err := json.Unmarshal([]byte(`{"enterprise_code":999}`), &params); err != nil {
		t.Fatal(err)
	}
	if params.EnterpriseCode != 0 {
		t.Fatalf("supplier parameters accepted caller-controlled tenant: %d", params.EnterpriseCode)
	}
}
