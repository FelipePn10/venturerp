//go:build integration

package nfse_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
	nfserepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func sp(s string) *string { return &s }

// Round-trips an NFS-e through create → get → authorize → list → cancel against a
// real Postgres (migration 000150).
func TestIntegration_NFSe_CRUDAndAuthorization(t *testing.T) {
	pool := testutil.Pool(t)
	repo := nfserepo.NewNFSeRepositoryPG(pool)
	ctx := context.Background()
	user := uuid.New()

	n := &entity.NFSe{
		TipoRPS:            1,
		DataEmissao:        time.Now(),
		Status:             entity.NFSeStatusRascunho,
		NaturezaOperacao:   1,
		TomadorRazaoSocial: sp("Cliente Serviços SA"),
		TomadorEmail:       sp("financeiro@cliente.com"),
		ItemListaServico:   "14.01",
		Discriminacao:      "Manutenção de equipamento industrial",
		CodigoMunicipio:    "4106902",
		ValorServicos:      1000,
		ValorDeducoes:      0,
		AliquotaISS:        0.05,
		IssRetido:          false,
		ValorISS:           50,
		ValorLiquido:       1000,
		Notes:              sp("teste integração"),
		CreatedBy:          user,
	}
	created, err := repo.Create(ctx, n)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM nfse WHERE id=$1", created.ID)

	// Round-trip the persisted fields.
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.NFSeStatusRascunho {
		t.Errorf("status = %s, want RASCUNHO", got.Status)
	}
	if got.ItemListaServico != "14.01" || got.ValorServicos != 1000 || got.ValorISS != 50 {
		t.Errorf("unexpected NFS-e fields: item=%s servicos=%.2f iss=%.2f", got.ItemListaServico, got.ValorServicos, got.ValorISS)
	}
	if got.TomadorRazaoSocial == nil || *got.TomadorRazaoSocial != "Cliente Serviços SA" {
		t.Errorf("tomador = %v", got.TomadorRazaoSocial)
	}

	// Authorization persists the city-hall identifiers and flips status.
	upd, err := repo.UpdateAuthorization(ctx, created.ID, "2024-NFSE-0001", "VERIF-123", "https://nfse.exemplo/v/123", "ref-nfse-1")
	if err != nil {
		t.Fatalf("UpdateAuthorization: %v", err)
	}
	if upd.Status != entity.NFSeStatusAutorizada {
		t.Errorf("status = %s, want AUTORIZADA", upd.Status)
	}
	if upd.NumeroNFSe == nil || *upd.NumeroNFSe != "2024-NFSE-0001" {
		t.Errorf("numero_nfse = %v", upd.NumeroNFSe)
	}
	if upd.CodigoVerificacao == nil || *upd.CodigoVerificacao != "VERIF-123" {
		t.Errorf("codigo_verificacao = %v", upd.CodigoVerificacao)
	}
	if upd.FocusRef == nil || *upd.FocusRef != "ref-nfse-1" {
		t.Errorf("focus_ref = %v", upd.FocusRef)
	}

	// Present in the listing.
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, x := range list {
		if x.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Error("created NFS-e not present in List")
	}

	// Cancel flips status.
	canc, err := repo.UpdateStatus(ctx, created.ID, entity.NFSeStatusCancelada)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if canc.Status != entity.NFSeStatusCancelada {
		t.Errorf("status = %s, want CANCELADA", canc.Status)
	}
}
