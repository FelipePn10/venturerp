package focusnfe

import (
	"encoding/json"
	"testing"
)

func TestPayloadLevaSuframaDoDestinatario(t *testing.T) {
	p := NFEPayload{Destinatario: NFEDestinatario{CNPJCPF: "11222333000181", Nome: "CLIENTE ZFM", UF: "AM", IndicadorIE: 9, Suframa: "20.0100.12-3"}}
	raw, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["inscricao_suframa_destinatario"] != "200100123" {
		t.Fatalf("ISUF = %v", m["inscricao_suframa_destinatario"])
	}
	p.Destinatario.Suframa = ""
	raw, _ = json.Marshal(p)
	m = map[string]any{}
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["inscricao_suframa_destinatario"]; ok {
		t.Fatal("sem inscrição, o campo não pode ir")
	}
}
