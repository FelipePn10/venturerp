//go:build integration

package fiscal_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func strptr(s string) *string { return &s }

// Round-trips a fiscal exit with ICMS-ST values through the hand-written SQL,
// confirming the new ST columns (migration 000147) persist and read back.
func TestIntegration_FiscalExit_STPersistence(t *testing.T) {
	pool := testutil.Pool(t)
	repo := fiscalrepo.NewFiscalRepositoryPG(pool)
	ctx := context.Background()
	user := uuid.New()

	exit := &entity.FiscalExit{
		NumeroNF:         testutil.UniqueCode(),
		Serie:            "TST",
		DataEmissao:      time.Now(),
		Cfop:             "5401",
		NaturezaOperacao: "Venda com ST",
		ValorProdutos:    1000,
		ValorIPI:         50,
		ValorICMS:        120,
		BaseICMSST:       1470,
		ValorICMSST:      56.40,
		ValorTotal:       1106.40,
		Status:           entity.ExitStatusDraft,
		CreatedBy:        user,
	}
	created, err := repo.CreateExit(ctx, exit)
	if err != nil {
		t.Fatalf("CreateExit: %v", err)
	}
	defer func() {
		testutil.Exec(t, pool, "DELETE FROM fiscal_exit_items WHERE fiscal_exit_id=$1", created.ID)
		testutil.Exec(t, pool, "DELETE FROM fiscal_exits WHERE id=$1", created.ID)
	}()

	item := &entity.FiscalExitItem{
		FiscalExitID: created.ID,
		Sequence:     1,
		Cfop:         "5401",
		Quantity:     1,
		UnitPrice:    1000,
		TotalPrice:   1000,
		BaseICMS:     1000,
		AliqICMS:     0.12,
		ValorICMS:    120,
		BaseICMSST:   1470,
		AliqICMSST:   0.12,
		ValorICMSST:  56.40,
		MVA:          0.40,
		CstICMS:      strptr("10"),
	}
	if _, err := repo.CreateExitItem(ctx, item); err != nil {
		t.Fatalf("CreateExitItem: %v", err)
	}

	// Header round-trip.
	got, err := repo.GetExitByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetExitByID: %v", err)
	}
	if got.BaseICMSST != 1470 || got.ValorICMSST != 56.40 {
		t.Errorf("exit ST header = base %.2f valor %.2f, want 1470/56.40", got.BaseICMSST, got.ValorICMSST)
	}

	// Item round-trip.
	items, err := repo.GetExitItems(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetExitItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	it := items[0]
	if it.ValorICMSST != 56.40 || it.BaseICMSST != 1470 || it.AliqICMSST != 0.12 || it.MVA != 0.40 {
		t.Errorf("item ST = base %.2f aliq %.4f valor %.2f mva %.4f, want 1470/0.12/56.40/0.40",
			it.BaseICMSST, it.AliqICMSST, it.ValorICMSST, it.MVA)
	}
	if it.CstICMS == nil || *it.CstICMS != "10" {
		t.Errorf("item CST = %v, want 10", it.CstICMS)
	}
}

// Covers CT-e creation with emission_data (JSONB) and the SEFAZ authorization
// persistence (migration 000149): focus_ref / protocolo / chave / status.
func TestIntegration_CTe_AuthorizationPersistence(t *testing.T) {
	pool := testutil.Pool(t)
	repo := fiscalrepo.NewFiscalRepositoryPG(pool)
	ctx := context.Background()

	emission := `{"natureza_operacao":"Prestação de serviço de transporte","tipo_cte":0,"modal":"01"}`
	cte := &entity.FiscalCTe{
		NumeroCTe:           testutil.UniqueCode(),
		CreatedBy:           uuid.New(),
		Serie:               "1",
		DataEmissao:         time.Now(),
		DataEntrada:         time.Now(),
		CnpjEmitente:        "11222333000181",
		RazaoSocialEmitente: "Transportadora Teste",
		Cfop:                "1352",
		ValorFrete:          500,
		ValorTotal:          500,
		ValorICMS:           60,
		BaseICMS:            500,
		AliqICMS:            0.12,
		TipoRateio:          "VALOR",
		Status:              "PENDENTE",
		EmissionData:        &emission,
		IsActive:            true,
	}
	created, err := repo.CreateCTe(ctx, cte)
	if err != nil {
		t.Fatalf("CreateCTe: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM fiscal_cte WHERE id=$1", created.ID)

	got, err := repo.GetCTeByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCTeByID: %v", err)
	}
	if got.EmissionData == nil || !strings.Contains(*got.EmissionData, "transporte") {
		t.Errorf("emission_data not persisted: %v", got.EmissionData)
	}
	if got.FocusRef != nil {
		t.Errorf("focus_ref should be nil before authorization, got %v", got.FocusRef)
	}
	if got.Status != "PENDENTE" {
		t.Errorf("status = %s, want PENDENTE", got.Status)
	}

	upd, err := repo.UpdateCTeAuthorization(ctx, created.ID, "CHAVE-CTE-001", "PROTO-001", "ref-cte-001")
	if err != nil {
		t.Fatalf("UpdateCTeAuthorization: %v", err)
	}
	if upd.Status != "AUTORIZADO" {
		t.Errorf("status = %s, want AUTORIZADO", upd.Status)
	}
	if upd.ChaveAcesso == nil || *upd.ChaveAcesso != "CHAVE-CTE-001" {
		t.Errorf("chave = %v, want CHAVE-CTE-001", upd.ChaveAcesso)
	}
	if upd.Protocolo == nil || *upd.Protocolo != "PROTO-001" {
		t.Errorf("protocolo = %v, want PROTO-001", upd.Protocolo)
	}
	if upd.FocusRef == nil || *upd.FocusRef != "ref-cte-001" {
		t.Errorf("focus_ref = %v, want ref-cte-001", upd.FocusRef)
	}
}
