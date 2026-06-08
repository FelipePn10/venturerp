//go:build integration

package financial_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Exercises the advance-payment ledger end-to-end against a real Postgres:
// creating a PAGAR advance moves cash, applying it settles a conta a pagar, and
// the advance balance / status track correctly.
func TestIntegration_Adiantamento_CreateAndApplyToContaPagar(t *testing.T) {
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	ctx := context.Background()
	user := uuid.New()
	now := time.Now()

	// Bank account with a known starting balance.
	cb, err := repo.CreateContaBancaria(ctx, &entity.ContaBancaria{
		Banco:        "341",
		Agencia:      "0001",
		Conta:        "ADV",
		Descricao:    "Adiantamento Test",
		SaldoInicial: decimal.NewFromInt(10000),
		IsActive:     true,
		CreatedBy:    user,
	})
	if err != nil {
		t.Fatalf("CreateContaBancaria: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM contas_bancarias WHERE id=$1", cb.ID)

	// A conta a pagar of 1000 to be settled by the advance.
	cp, err := repo.CreateContaPagar(ctx, &entity.ContaPagar{
		NumeroDocumento: "ADV-CP",
		TipoDocumento:   "NF-e",
		DataLancamento:  now,
		DataEmissao:     now,
		DataVencimento:  now.AddDate(0, 0, 30),
		ValorBruto:      decimal.NewFromInt(1000),
		Desconto:        decimal.Zero,
		Juros:           decimal.Zero,
		Multa:           decimal.Zero,
		ValorPago:       decimal.Zero,
		ParcelaNumero:   1,
		ParcelaTotal:    1,
		StatusAprovacao: entity.AprovacaoAprovado,
		Status:          entity.ContaPagarStatusPendente,
		IsActive:        true,
		CriadoPor:       user,
	})
	if err != nil {
		t.Fatalf("CreateContaPagar: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM contas_pagar WHERE id=$1", cp.ID)

	// Create a PAGAR advance of 1500 (cash out).
	desc := "Sinal de pedido"
	contaID := cb.ID
	adv, err := repo.CreateAdiantamentoAtomico(ctx,
		&entity.Adiantamento{
			Tipo:             entity.AdiantamentoTipoPagar,
			ContaBancariaID:  cb.ID,
			DataAdiantamento: now,
			ValorOriginal:    decimal.NewFromInt(1500),
			Descricao:        &desc,
			CreatedBy:        user,
		},
		entity.FluxoCaixa{
			Data:            now,
			Tipo:            entity.FluxoCaixaTipoSaida,
			Valor:           decimal.NewFromInt(1500),
			ContaBancariaID: &contaID,
			Descricao:       &desc,
		},
	)
	if err != nil {
		t.Fatalf("CreateAdiantamentoAtomico: %v", err)
	}
	defer func() {
		testutil.Exec(t, pool, "DELETE FROM adiantamento_aplicacoes WHERE adiantamento_id=$1", adv.ID)
		testutil.Exec(t, pool, "DELETE FROM adiantamentos WHERE id=$1", adv.ID)
		testutil.Exec(t, pool, "DELETE FROM fluxo_caixa WHERE conta_bancaria_id=$1", cb.ID)
	}()

	if adv.Status != entity.AdiantamentoStatusAberto {
		t.Errorf("new advance status = %s, want ABERTO", adv.Status)
	}

	// Cash out: 10000 - 1500 = 8500.
	cbAfter, err := repo.GetContaBancaria(ctx, cb.ID)
	if err != nil {
		t.Fatalf("GetContaBancaria: %v", err)
	}
	if !cbAfter.SaldoInicial.Equal(decimal.NewFromInt(8500)) {
		t.Errorf("saldo after advance = %s, want 8500", cbAfter.SaldoInicial)
	}

	// Apply 1000 of the advance onto the conta a pagar — fully settles it.
	ap, err := repo.AplicarAdiantamentoAtomico(ctx, adv.ID, "PAGAR", cp.ID, decimal.NewFromInt(1000), user, now)
	if err != nil {
		t.Fatalf("AplicarAdiantamentoAtomico: %v", err)
	}
	if !ap.ValorAplicado.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("valor aplicado = %s, want 1000", ap.ValorAplicado)
	}

	cpAfter, err := repo.GetContaPagar(ctx, cp.ID)
	if err != nil {
		t.Fatalf("GetContaPagar: %v", err)
	}
	if cpAfter.Status != entity.ContaPagarStatusPago {
		t.Errorf("conta pagar status = %s, want PAGO", cpAfter.Status)
	}
	if !cpAfter.ValorAdiantamentoAbatido.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("abatido = %s, want 1000", cpAfter.ValorAdiantamentoAbatido)
	}

	advAfter, err := repo.GetAdiantamento(ctx, adv.ID)
	if err != nil {
		t.Fatalf("GetAdiantamento: %v", err)
	}
	if advAfter.Status != entity.AdiantamentoStatusParcial {
		t.Errorf("advance status = %s, want PARCIAL", advAfter.Status)
	}
	if !advAfter.Saldo().Equal(decimal.NewFromInt(500)) {
		t.Errorf("advance saldo = %s, want 500", advAfter.Saldo())
	}

	// Over-applying beyond the remaining balances must fail.
	if _, err := repo.AplicarAdiantamentoAtomico(ctx, adv.ID, "PAGAR", cp.ID, decimal.NewFromInt(9999), user, now); err == nil {
		t.Error("expected error applying more than the advance balance")
	}

	// Mismatched tipo must fail (RECEBER application onto a PAGAR advance).
	if _, err := repo.AplicarAdiantamentoAtomico(ctx, adv.ID, "RECEBER", cp.ID, decimal.NewFromInt(10), user, now); err == nil {
		t.Error("expected error applying with mismatched tipo")
	}

	// List by tipo includes our advance.
	tipo := "PAGAR"
	list, err := repo.ListAdiantamentos(ctx, &tipo, nil)
	if err != nil {
		t.Fatalf("ListAdiantamentos: %v", err)
	}
	found := false
	for _, a := range list {
		if a.ID == adv.ID {
			found = true
		}
	}
	if !found {
		t.Error("created advance not present in ListAdiantamentos(PAGAR)")
	}

	// One application recorded.
	aps, err := repo.ListAplicacoesByAdiantamento(ctx, adv.ID)
	if err != nil {
		t.Fatalf("ListAplicacoesByAdiantamento: %v", err)
	}
	if len(aps) != 1 {
		t.Errorf("aplicacoes = %d, want 1", len(aps))
	}
}
