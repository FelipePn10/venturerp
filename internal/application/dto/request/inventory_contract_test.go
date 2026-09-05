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
	if err == nil {
		t.Fatal("valor fora da lista foi aceito")
	}
	// A mensagem precisa nomear o campo, repetir o valor recusado e listar o
	// que é aceito — sem isso o usuário não sabe o que corrigir.
	for _, trecho := range []string{"Tipo de localização", `"INVALIDO"`, "não é aceito", "INTERNO"} {
		if !strings.Contains(err.Error(), trecho) {
			t.Fatalf("mensagem sem %q: %v", trecho, err)
		}
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
