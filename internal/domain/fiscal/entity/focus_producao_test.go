package entity

import "testing"

func TestConferirTrocaDoFocus(t *testing.T) {
	s := func(v string) *string { return &v }
	homolog := &FiscalConfig{FocusNfeAmbiente: "homologacao"}
	prod := &FiscalConfig{FocusNfeAmbiente: "producao"}
	casos := []struct {
		nome     string
		atual    *FiscalConfig
		ambiente string
		token    *string
		ok       bool
	}{
		{"homologação troca o token de homologação", homolog, "homologacao", s("tok-h"), true},
		{"sem configuração grava homologação", nil, "homologacao", s("tok-h"), true},
		{"homologação não entra em produção pela tela", homolog, "producao", nil, false},
		{"sem configuração não nasce em produção", nil, "producao", s("tok-p"), false},
		{"produção salva os outros parâmetros", prod, "producao", nil, true},
		{"produção com token vazio é manter", prod, "Producao", s("  "), true},
		{"produção não troca o token", prod, "producao", s("outro"), false},
		{"produção não volta para homologação", prod, "homologacao", nil, false},
	}
	for _, c := range casos {
		err := ConferirTrocaDoFocus(c.atual, c.ambiente, c.token)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.nome, err)
		}
	}
}
