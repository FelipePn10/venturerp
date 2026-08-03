package types

import (
	"encoding/json"
	"testing"
)

func TestIntegerEnumsAcceptStringAndIntegerAndRejectInvalid(t *testing.T) {
	tests := []struct {
		name       string
		stringJSON string
		intJSON    string
		newValue   func() any
	}{
		{"mrp", `"PROJETO"`, `1`, func() any { return new(TypeMRPItem) }},
		{"situation", `"PROMOCAO"`, `1`, func() any { return new(TypeSituationItem) }},
		{"struct", `"COMERCIAL"`, `1`, func() any { return new(TypeStructItem) }},
		{"item type", `"SERVICO"`, `3`, func() any { return new(TypeItem) }},
		{"type of use", `"IMOBILIZADO"`, `2`, func() any { return new(TypeOfUseItem) }},
		{"location", `"ESPECIAL"`, `7`, func() any { return new(TypeLocation) }},
		{"warehouse", `"NORMAL"`, `1`, func() any { return new(TypeWarehouse) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, input := range []string{test.stringJSON, test.intJSON} {
				if err := json.Unmarshal([]byte(input), test.newValue()); err != nil {
					t.Fatalf("unmarshal %s: %v", input, err)
				}
			}
			for _, input := range []string{`99`, `"INVALID"`} {
				if err := json.Unmarshal([]byte(input), test.newValue()); err == nil {
					t.Fatalf("expected %s to be rejected", input)
				}
			}
		})
	}
}

func TestIntegerEnumsMarshalRoundTrip(t *testing.T) {
	values := []any{TypeMRPItem(PROJETO), TypeSituationItem(PROMOCAO), TypeStructItem(COMERCIAL), TypeItem(SERVICO), TypeOfUseItem(IMOBILIZADO), TypeLocation(ESPECIAL), TypeWarehouse(NORMAL)}
	for _, original := range values {
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		var target any
		switch original.(type) {
		case TypeMRPItem:
			target = new(TypeMRPItem)
		case TypeSituationItem:
			target = new(TypeSituationItem)
		case TypeStructItem:
			target = new(TypeStructItem)
		case TypeItem:
			target = new(TypeItem)
		case TypeOfUseItem:
			target = new(TypeOfUseItem)
		case TypeLocation:
			target = new(TypeLocation)
		case TypeWarehouse:
			target = new(TypeWarehouse)
		}
		if err := json.Unmarshal(data, target); err != nil {
			t.Fatalf("round trip %T: %v", original, err)
		}
	}
}

func TestAdditionalItemUnitsOfMeasurement(t *testing.T) {
	for _, unit := range []TypeUnitOfMeasurementItem{L, CX, PC, GL, PAR} {
		if !unit.IsValid() {
			t.Fatalf("expected %s to be valid", unit)
		}
		data, err := json.Marshal(unit)
		if err != nil {
			t.Fatal(err)
		}
		var decoded TypeUnitOfMeasurementItem
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded != unit {
			t.Fatalf("got %s, want %s", decoded, unit)
		}
	}
}
