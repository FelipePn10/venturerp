//go:build integration

package fiscal_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A alteração da nota especial grava período, data e linha de apuração (antes
// descartava) e não aceita a linha de outra empresa nem a nota de outra empresa.
func TestNotaEspecial_AlteracaoGravaEIsolaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ctx := context.Background()
		ids := []int64{a.ID, b.ID}
		for _, q := range []string{
			`DELETE FROM special_adjustment_notes WHERE empresa_id = ANY($1)`,
			`DELETE FROM icms_apuracao_lines WHERE enterprise_id = ANY($1)`,
		} {
			if _, err := pool.Exec(ctx, q, ids); err != nil {
				t.Errorf("limpeza: %v", err)
			}
		}
	})
	repo := repository.NewFiscalParamsRepository(sqlc.New(pool), pool)
	sufixo := testutil.UniqueCode() % 100_000
	linha := func(ctx context.Context, cod string) int64 {
		l, err := repo.CreateICMSApuracaoLine(ctx, &entity.ICMSApuracaoLine{Code: cod, Description: "teste", LineType: entity.LineTypeDebito, IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return l.ID
	}
	linhaA, linhaB := linha(a.Ctx, fmt.Sprintf("A%d", sufixo)), linha(b.Ctx, fmt.Sprintf("B%d", sufixo))

	n, err := repo.CreateSpecialAdjustmentNote(a.Ctx, &entity.SpecialAdjustmentNote{Purpose: entity.SpecialNoteAjuste,
		Status: entity.SpecialNoteRascunho, IssueDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Period: "2026-11"})
	if err != nil {
		t.Fatal(err)
	}
	n.Period, n.IssueDate, n.ICMSApuracaoLineID, n.TotalICMS = "2026-10", time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), &linhaA, decimal.NewFromInt(150)
	if _, err := repo.UpdateSpecialAdjustmentNote(a.Ctx, n); err != nil {
		t.Fatalf("empresa A não alterou a própria nota: %v", err)
	}
	got, err := repo.GetSpecialAdjustmentNote(a.Ctx, n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Period != "2026-10" || got.ICMSApuracaoLineID == nil || *got.ICMSApuracaoLineID != linhaA || got.IssueDate.Day() != 30 {
		t.Fatalf("a alteração não gravou período/data/linha: %+v", got)
	}

	n.ICMSApuracaoLineID = &linhaB
	if _, err := repo.UpdateSpecialAdjustmentNote(a.Ctx, n); err == nil {
		t.Fatal("a nota da empresa A aceitou a linha de apuração da B")
	}
	n.ICMSApuracaoLineID = nil
	if _, err := repo.GetSpecialAdjustmentNote(b.Ctx, n.ID); err == nil {
		t.Fatal("empresa B leu a nota da A")
	}
	n.Status = entity.SpecialNoteCancelada
	if _, err := repo.UpdateSpecialAdjustmentNote(b.Ctx, n); err == nil {
		t.Fatal("empresa B alterou (cancelou) a nota da A")
	}
	if l, err := repo.ListSpecialAdjustmentNotes(b.Ctx, int(b.ID), "2026-10"); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d nota(s) da A (%v)", len(l), err)
	}
}
