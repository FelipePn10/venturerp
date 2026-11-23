package financial_uc

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

func TestDREGerencialConsolidaNasSinteticas(t *testing.T) {
	d := decimal.NewFromInt
	contas := []entity.ContaDoResultado{
		{ID: 1, Numero: "3", Nome: "Receitas"},
		{ID: 2, ParentID: 1, Numero: "3.1", Nome: "Vendas", Analitica: true, Creditos: d(1000)},
		{ID: 3, Numero: "4", Nome: "Custos e despesas"},
		{ID: 4, ParentID: 3, Numero: "4.1", Nome: "CMV", Analitica: true, Debitos: d(600)},
		{ID: 5, ParentID: 3, Numero: "4.2", Nome: "Despesas", Analitica: true, Debitos: d(150)},
		{ID: 6, Numero: "1.1", Nome: "Caixa", Analitica: true, Debitos: d(1000)}, // patrimonial: fora
	}
	dre := MontarDRE(contas, []string{"3", "4"})
	if !dre.Receita.Equal(d(1000)) || !dre.Resultado.Equal(d(250)) || len(dre.Linhas) != 5 {
		t.Fatalf("%+v", dre)
	}
	for _, l := range dre.Linhas {
		if l.Numero == "4" && (!l.Valor.Equal(d(-750)) || !l.Percentual.Equal(d(-75)) || l.Nivel != 0) {
			t.Fatalf("grupo 4: %+v", l)
		}
		if l.Numero == "4.1" && l.Nivel != 1 {
			t.Fatalf("nível: %+v", l)
		}
	}
}
