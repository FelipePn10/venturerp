package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPendenciasDoCliente(t *testing.T) {
	hoje := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	pol := DefaultCreditPolicy()
	s := func(v string) *string { return &v }
	d := func(dias int) *time.Time { v := hoje.AddDate(0, 0, dias); return &v }
	ok := SaneamentoCliente{Code: 1, DocumentType: DocumentCNPJ, DocumentNumber: "11.222.333/0001-81", UF: "PR",
		NFeEmail: s("nfe@x.com"), RegistrationStatus: s("ATIVA"), RegistrationCheckedAt: d(-5)}
	if p := PendenciasDoCliente(ok, pol, hoje); len(p) != 0 {
		t.Fatalf("cadastro em ordem com pendências: %+v", p)
	}
	ruim := ok
	ruim.DocumentNumber = "11.222.333/0001-82"
	ruim.UF = ""
	ruim.SuframaCode = s("200100123")
	ruim.RegistrationStatus = s("BAIXADA")
	ruim.CreditLimit = decimal.NewFromInt(5000)
	ruim.LastReviewOrigin = "LEGADO"
	ruim.NFeEmail = nil
	cats := map[string]string{}
	for _, p := range PendenciasDoCliente(ruim, pol, hoje) {
		cats[p.Categoria] = p.Gravidade
		if p.ComoResolver == "" {
			t.Errorf("%s sem como resolver", p.Categoria)
		}
	}
	for cat, grav := range map[string]string{"DOCUMENTO_INVALIDO": SaneamentoImpede, "SEM_ENDERECO": SaneamentoImpede,
		"SUFRAMA_INVALIDA": SaneamentoImpede, "RECEITA_IRREGULAR": SaneamentoImpede, "LIMITE_SEM_ANALISE": SaneamentoAtencao,
		"SEM_EMAIL_NFE": SaneamentoAtencao} {
		if cats[cat] != grav {
			t.Errorf("%s: gravidade %q, quer %q (todas: %v)", cat, cats[cat], grav, cats)
		}
	}
	// SUFRAMA válida mas vencida; limite vencido; CNPJ nunca conferido.
	venc := ok
	venc.SuframaCode, venc.SuframaExpiry = s("200100122"), d(-1)
	venc.CreditLimit, venc.CreditLimitExpiresAt, venc.LastReviewOrigin = decimal.NewFromInt(1000), d(-3), "REVISAO"
	venc.RegistrationCheckedAt = nil
	cats = map[string]string{}
	for _, p := range PendenciasDoCliente(venc, pol, hoje) {
		cats[p.Categoria] = p.Gravidade
	}
	if cats["SUFRAMA_VENCIDA"] == "" || cats["LIMITE_VENCIDO"] == "" || cats["RECEITA_NAO_CONFERIDA"] == "" {
		t.Fatalf("faltaram pendências: %v", cats)
	}
	// Limite legado vencido trava o pedido: impede, não só atenção.
	venc.LastReviewOrigin = "LEGADO"
	for _, p := range PendenciasDoCliente(venc, pol, hoje) {
		if p.Categoria == "LIMITE_SEM_ANALISE" || (p.Categoria == "LIMITE_VENCIDO" && p.Gravidade != SaneamentoImpede) {
			t.Fatalf("legado vencido: %+v", p)
		}
	}
}
