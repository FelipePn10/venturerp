//go:build integration

package financial_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Parcelamento pela metade faria parte da dívida sumir: ou entram todas, ou nenhuma.
func TestIntegration_ContasPagarParceladasTudoOuNada(t *testing.T) {
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	dona := testutil.TenantContext(t, pool)
	user := testutil.Actor(t, pool)
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM contas_pagar WHERE numero_documento LIKE 'PARC-%'") })

	titulo := func(numero string, parcela int32) *entity.ContaPagar {
		return &entity.ContaPagar{
			NumeroDocumento: numero, TipoDocumento: "OUTROS", DataLancamento: time.Now(), DataEmissao: time.Now(), DataVencimento: time.Now().AddDate(0, 0, int(parcela)*30),
			ValorBruto: decimal.NewFromInt(100), Desconto: decimal.Zero, Juros: decimal.Zero, Multa: decimal.Zero, ValorPago: decimal.Zero,
			ParcelaNumero: parcela, ParcelaTotal: 2, Status: "pendente", StatusAprovacao: "pendente", IsActive: true, CriadoPor: user,
		}
	}
	doc := "PARC-" + time.Now().Format("150405.000")
	contar := func() int {
		rows, err := repo.ListContasPagar(dona, domrepo.CPFilter{Documento: &doc})
		if err != nil {
			t.Fatal(err)
		}
		return len(rows)
	}

	// A segunda parcela aponta para um plano de contas inexistente: a primeira não pode ficar.
	ruim := titulo(doc+" 2/2", 2)
	ruim.Rateios = []entity.RateioContaPagar{{PlanoContasID: -1, Valor: decimal.NewFromInt(100)}}
	if _, err := repo.CreateContasPagar(dona, []*entity.ContaPagar{titulo(doc+" 1/2", 1), ruim}); err == nil {
		t.Fatal("esperava erro do rateio inválido")
	}
	if n := contar(); n != 0 {
		t.Fatalf("parcelamento pela metade: %d título(s) ficaram gravados", n)
	}

	out, err := repo.CreateContasPagar(dona, []*entity.ContaPagar{titulo(doc+" 1/2", 1), titulo(doc+" 2/2", 2)})
	if err != nil {
		t.Fatal(err)
	}
	if contar() != 2 || out[1].ParcelaPaiID == nil || *out[1].ParcelaPaiID != out[0].ID {
		t.Fatalf("parcelas gravadas erradas: pai=%v", out[1].ParcelaPaiID)
	}

	vizinha := empresaPropria(t, pool)
	rows, err := repo.ListContasPagar(vizinha, domrepo.CPFilter{Documento: &doc})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("outra empresa enxergou %d parcela(s)", len(rows))
	}
}
