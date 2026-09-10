package structure

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

func TestToItemStructureDTOUsesPublicJSONContract(t *testing.T) {
	warehouse := int64(7)
	component := &entity.ItemStructure{
		ID:             42,
		ParentCode:     100,
		ChildCode:      200,
		Quantity:       1.5,
		Sequence:       3,
		WarehouseCode:  &warehouse,
		CostLossType:   "PERCENTUAL",
		IsActive:       true,
	}

	body, err := json.Marshal(ToItemStructureDTO(component))
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	jsonBody := string(body)

	for _, field := range []string{`"id":42`, `"parent_code":100`, `"child_code":200`, `"sequence":3`, `"warehouse_code":7`} {
		if !strings.Contains(jsonBody, field) {
			t.Fatalf("public response is missing %s: %s", field, jsonBody)
		}
	}
	for _, internalField := range []string{`"ID"`, `"ParentCode"`, `"ChildCode"`} {
		if strings.Contains(jsonBody, internalField) {
			t.Fatalf("response leaked internal field %s: %s", internalField, jsonBody)
		}
	}
}
