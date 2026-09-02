package request

import (
	"encoding/json"
	"testing"
)

func TestManufacturingContractsAcceptCanonicalAndLegacyCodes(t *testing.T) {
	for _, body := range []string{`{"item_code":"TEA452-0"}`, `{"item_code":10001}`} {
		var dto CreateRouteDTO
		if err := json.Unmarshal([]byte(body), &dto); err != nil || dto.ItemCode.String() == "" {
			t.Fatalf("roteiro %s: código=%q erro=%v", body, dto.ItemCode, err)
		}
	}
	for _, body := range []string{`{"parent_code":"PAI-01","child_code":"MP-02"}`, `{"parent_code":10001,"child_code":10002}`} {
		var dto CreateStructureComponentDTO
		if err := json.Unmarshal([]byte(body), &dto); err != nil || dto.ParentCode.String() == "" || dto.ChildCode.String() == "" {
			t.Fatalf("BOM %s: pai=%q filho=%q erro=%v", body, dto.ParentCode, dto.ChildCode, err)
		}
	}
	for _, body := range []string{`{"item_code":"TEA452-0","machine_code":7}`, `{"item_code":10001,"machine_code":7}`} {
		var dto CreateItemMachineTimeDTO
		if err := json.Unmarshal([]byte(body), &dto); err != nil || dto.ItemCode.String() == "" {
			t.Fatalf("tempo por item × máquina %s: código=%q erro=%v", body, dto.ItemCode, err)
		}
	}
}

func TestManufacturingContractsRejectInvalidCodes(t *testing.T) {
	for _, body := range []string{`{"item_code":0}`, `{"item_code":1.2}`, `{"item_code":false}`, `{"item_code":""}`} {
		var dto CreateRouteDTO
		if err := json.Unmarshal([]byte(body), &dto); err == nil {
			t.Fatalf("esperava erro para %s", body)
		}
		var machineDTO CreateItemMachineTimeDTO
		if err := json.Unmarshal([]byte(body), &machineDTO); err == nil {
			t.Fatalf("tempo por item × máquina deveria rejeitar %s", body)
		}
	}
}
