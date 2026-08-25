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
