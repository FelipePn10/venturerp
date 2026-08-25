package request

import (
	"encoding/json"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

func TestCreateDeliveryRescheduleDTOItemCodeUsesNumericJSON(t *testing.T) {
	var dto CreateDeliveryRescheduleDTO
	if err := json.Unmarshal([]byte(`{"item_code":4853}`), &dto); err != nil {
		t.Fatalf("item_code numérico deveria ser aceito: %v", err)
	}
	if dto.ItemCode != valueobject.ItemCode(4853) {
		t.Fatalf("item_code = %d, esperado 4853", dto.ItemCode)
	}

	if err := json.Unmarshal([]byte(`{"item_code":"4853"}`), &dto); err == nil {
		t.Fatal("item_code textual não deveria ser aceito pelo contrato")
	}
}

func TestCreateDeliveryRescheduleDTOIgnoresClientSuppliedCode(t *testing.T) {
	var dto CreateDeliveryRescheduleDTO
	if err := json.Unmarshal([]byte(`{"code":0,"sales_order_code":10,"item_code":4853}`), &dto); err != nil {
		t.Fatalf("payload deveria ser decodificado sem aceitar code no contrato: %v", err)
	}

	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("não foi possível serializar o DTO: %v", err)
	}
	if string(encoded) == "" || json.Valid(encoded) == false {
		t.Fatalf("DTO serializado inválido: %s", encoded)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["code"]; exists {
		t.Fatal("code não deve fazer parte do contrato de criação")
	}
}
