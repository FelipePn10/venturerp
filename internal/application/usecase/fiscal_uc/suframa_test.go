package fiscal_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestSuframaValida(t *testing.T) {
	hoje := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	dia := func(d int) *time.Time { v := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC); return &v }
	cod, errado := "20.0100.12-2", "200100123"
	casos := []struct {
		nome     string
		codigo   *string
		validade *time.Time
		quer     string
	}{
		{"sem inscrição", nil, nil, ""},
		{"vale hoje", &cod, dia(16), "200100122"},
		{"vence amanhã", &cod, dia(17), "200100122"},
		{"venceu ontem", &cod, dia(15), ""},
		{"sem validade", &cod, nil, "200100122"},
		{"dígito errado (cadastro antigo)", &errado, nil, ""},
	}
	for _, c := range casos {
		if got := SuframaValida(c.codigo, c.validade, hoje); got != c.quer {
			t.Errorf("%s: %q, quer %q", c.nome, got, c.quer)
		}
	}
}

// SUFRAMA digitada na nota avulsa: dígito errado recusa (desoneraria o ICMS e a
// SEFAZ rejeitaria com 235); a válida vai só com os dígitos.
func TestSuframaDigitadaNaNotaEConferida(t *testing.T) {
	nota := func(isuf string) (*repoNumeracao, error) {
		repo := &repoNumeracao{proximo: 1}
		uc := &CreateFiscalExitUseCase{Repo: repo, Auth: authLiberado{}}
		_, err := uc.Execute(context.Background(), request.CreateFiscalExitDTO{
			DataEmissao: "2026-09-27", Cfop: "5101", NaturezaOperacao: "VENDA", SuframaDestinatario: &isuf,
			Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, Cfop: "5101", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(10), TotalPrice: decimal.NewFromInt(10)}},
		})
		return repo, err
	}
	if repo, err := nota("200100123"); err == nil || repo.criada != nil {
		t.Fatalf("SUFRAMA com dígito errado foi aceita: %v", err)
	}
	repo, err := nota("20.0100.122")
	if err != nil {
		t.Fatalf("SUFRAMA válida recusada: %v", err)
	}
	if repo.criada.SuframaDestinatario == nil || *repo.criada.SuframaDestinatario != "200100122" {
		t.Fatalf("SUFRAMA gravada %v, quer só os dígitos", repo.criada.SuframaDestinatario)
	}
}
