package financial_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

type parceladaRepo struct {
	crudRepo
	gravados []*entity.ContaPagar
}

func (r *parceladaRepo) CreateContasPagar(_ context.Context, t []*entity.ContaPagar) ([]*entity.ContaPagar, error) {
	r.gravados = t
	return t, nil
}

func dinheiro(s string) decimal.Decimal { d, _ := decimal.NewFromString(s); return d }

func dtoParcelado(parcelas ...string) request.CreateContaPagarParceladaDTO {
	dto := request.CreateContaPagarParceladaDTO{CreateContaPagarDTO: request.CreateContaPagarDTO{
		NumeroDocumento: "FAT-77", TipoDocumento: "OUTROS", DataEmissao: "2026-11-01", ValorBruto: 1000,
	}}
	for i, v := range parcelas {
		dto.Parcelas = append(dto.Parcelas, request.ParcelaContaPagarDTO{DataVencimento: "2026-12-" + []string{"01", "02", "03", "04"}[i], Valor: dinheiro(v)})
	}
	return dto
}

func TestParcelada_EntradaMaisParcelasFechaNoTotal(t *testing.T) {
	repo := &parceladaRepo{}
	uc := CreateContaPagarParceladaUseCase{Repo: repo, Auth: crudAuth{ok: true}}
	out, err := uc.Execute(context.Background(), dtoParcelado("300", "350", "350"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].NumeroDocumento != "FAT-77 1/3" || out[2].ParcelaNumero != 3 || out[2].ParcelaTotal != 3 {
		t.Fatalf("parcelas erradas: %+v", out[0])
	}
	if !out[0].ValorBruto.Equal(dinheiro("300")) {
		t.Fatalf("entrada = %s", out[0].ValorBruto)
	}
}

func TestParcelada_DescontoERateioNaoPerdemCentavo(t *testing.T) {
	dto := dtoParcelado("333.33", "333.33", "333.34")
	dto.ValorBruto, dto.Desconto = 1010.10, 10.10
	dto.Rateios = []request.RateioContaPagarDTO{{PlanoContasID: 1, Valor: dinheiro("333.37")}, {PlanoContasID: 2, Valor: dinheiro("676.73")}}
	dia := func(s string) time.Time { d, _ := time.Parse("2006-01-02", s); return d }
	titulos, err := desdobrarTitulo(dto, uuid.New(), dia("2026-11-01"), []time.Time{dia("2026-12-01"), dia("2026-12-02"), dia("2026-12-03")},
		dinheiro("1010.10"), dinheiro("10.10"), dinheiro("1000"))
	if err != nil {
		t.Fatal(err)
	}
	bruto, desconto, p1, p2 := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	for _, c := range titulos {
		bruto, desconto = bruto.Add(c.ValorBruto), desconto.Add(c.Desconto)
		linha := decimal.Zero
		for _, r := range c.Rateios {
			linha = linha.Add(r.Valor)
			if r.PlanoContasID == 1 {
				p1 = p1.Add(r.Valor)
			} else {
				p2 = p2.Add(r.Valor)
			}
		}
		if !linha.Equal(c.ValorBruto) {
			t.Fatalf("rateio %s não fecha o título %s", linha, c.ValorBruto)
		}
	}
	if !bruto.Equal(dinheiro("1010.10")) || !desconto.Equal(dinheiro("10.10")) || !p1.Equal(dinheiro("333.37")) || !p2.Equal(dinheiro("676.73")) {
		t.Fatalf("bruto=%s desconto=%s p1=%s p2=%s", bruto, desconto, p1, p2)
	}
}

func TestParcelada_RecusaOQueNaoFecha(t *testing.T) {
	uc := CreateContaPagarParceladaUseCase{Repo: &parceladaRepo{}, Auth: crudAuth{ok: true}}
	casos := map[string]request.CreateContaPagarParceladaDTO{
		"somam":               dtoParcelado("300", "300"),
		"maior que zero":      dtoParcelado("1000", "0"),
		"informe as parcelas": dtoParcelado(),
	}
	antes := dtoParcelado("1000")
	antes.Parcelas[0].DataVencimento = "2026-10-01"
	casos["antes da emissão"] = antes
	for esperado, dto := range casos {
		if _, err := uc.Execute(context.Background(), dto); err == nil || !strings.Contains(err.Error(), esperado) {
			t.Errorf("esperava %q, veio %v", esperado, err)
		}
	}
}
