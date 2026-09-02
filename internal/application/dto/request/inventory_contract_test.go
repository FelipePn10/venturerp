package request

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWarehouseCodeAndIdentityContract(t *testing.T) {
	for _, test := range []struct {
		name, payload, want string
		invalid             bool
	}{
		{"texto", `{"code":"ALMOX01","location":"INTERNO","type":"NORMAL"}`, "ALMOX01", false},
		{"numérico temporário", `{"code":101,"location":"EXPEDICAO","type":"LINHA DE PRODUÇÃO"}`, "101", false},
		{"inválido", `{"code":1.5,"location":"INTERNO","type":"NORMAL"}`, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var dto CreateWarehouseRequestDTO
			err := json.Unmarshal([]byte(test.payload), &dto)
			if test.invalid {
				if err == nil {
					t.Fatal("código inválido deveria ser rejeitado")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if dto.Code.String() != test.want {
				t.Fatalf("code=%q", dto.Code)
			}
		})
	}
	var identity CreateWarehouseRequestDTO
	if err := json.Unmarshal([]byte(`{"code":"A","location":"INTERNO","type":"NORMAL","created_by":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.CreatedBy != uuid.Nil {
		t.Fatal("created_by do cliente foi aceito")
	}
}

func TestWarehouseEnumErrorsArePortuguese(t *testing.T) {
	var dto CreateWarehouseRequestDTO
	err := json.Unmarshal([]byte(`{"code":"A","location":"INVALIDO","type":"NORMAL"}`), &dto)
	if err == nil || !strings.Contains(err.Error(), "inválido") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestLotMaskItemCodeContract(t *testing.T) {
	for _, payload := range []string{`{"item_code":"TEA452-0"}`, `{"item_code":452}`} {
		var dto LotMaskDTO
		if err := json.Unmarshal([]byte(payload), &dto); err != nil {
			t.Fatal(err)
		}
		if dto.ItemCode == nil || dto.ItemCode.String() == "" {
			t.Fatal("item_code não decodificado")
		}
	}
	var invalid LotMaskDTO
	if err := json.Unmarshal([]byte(`{"item_code":4.2}`), &invalid); err == nil {
		t.Fatal("item_code inválido deveria ser rejeitado")
	}
}
