package request

import (
	"encoding/json"
	"testing"
)

// O `<input type="date">` manda só a data; time.Time puro só lê RFC3339 e
// devolvia o erro cru do parser do Go para o usuário.
func TestDataFlexivelAceitaDataSimplesERFC3339(t *testing.T) {
	var corpo struct {
		ValidFrom *DataFlexivel `json:"valid_from"`
	}
	for _, entrada := range []string{`{"valid_from":"2026-09-20"}`, `{"valid_from":"2026-09-20T00:00:00Z"}`} {
		if err := json.Unmarshal([]byte(entrada), &corpo); err != nil {
			t.Fatalf("%s recusado: %v", entrada, err)
		}
		if corpo.ValidFrom.Ponteiro() == nil || corpo.ValidFrom.Year() != 2026 || corpo.ValidFrom.Day() != 20 {
			t.Fatalf("%s: data lida errada: %v", entrada, corpo.ValidFrom)
		}
	}
	for _, vazio := range []string{`{"valid_from":""}`, `{"valid_from":null}`} {
		corpo.ValidFrom = nil
		if err := json.Unmarshal([]byte(vazio), &corpo); err != nil {
			t.Fatalf("%s recusado: %v", vazio, err)
		}
		if corpo.ValidFrom.Ponteiro() != nil {
			t.Fatalf("%s deveria virar nil", vazio)
		}
	}
	corpo.ValidFrom = nil
	if err := json.Unmarshal([]byte(`{"valid_from":"20/09/2026"}`), &corpo); err == nil {
		t.Fatal("data em formato brasileiro deveria ser recusada com mensagem própria")
	}
}
