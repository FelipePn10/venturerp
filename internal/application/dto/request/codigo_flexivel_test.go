package request

import (
	"encoding/json"
	"testing"
)

func TestCodigoFlexivelAceitaTextoENumero(t *testing.T) {
	var dto struct {
		A *CodigoFlexivel `json:"a,omitempty"`
		B *CodigoFlexivel `json:"b,omitempty"`
		C *CodigoFlexivel `json:"c,omitempty"`
		D *CodigoFlexivel `json:"d,omitempty"`
	}
	if err := json.Unmarshal([]byte(`{"a":"ALM-PA","b":900100,"c":""}`), &dto); err != nil {
		t.Fatal(err)
	}
	if got := dto.A.Ptr(); got == nil || *got != "ALM-PA" {
		t.Errorf("texto alfanumérico: obtido %v", got)
	}
	if got := dto.B.Ptr(); got == nil || *got != "900100" {
		t.Errorf("número de cliente antigo: obtido %v", got)
	}
	if dto.C.Ptr() != nil {
		t.Error("vazio deveria significar não informado")
	}
	if dto.D != nil || dto.D.Ptr() != nil {
		t.Error("ausente deveria ficar nil")
	}
}
