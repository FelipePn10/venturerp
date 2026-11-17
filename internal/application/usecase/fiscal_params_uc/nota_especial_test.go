package fiscal_params_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type notasFake struct {
	repository.FiscalParamsRepository
	n *entity.SpecialAdjustmentNote
}

func (f *notasFake) GetSpecialAdjustmentNote(context.Context, int64) (*entity.SpecialAdjustmentNote, error) {
	c := *f.n
	return &c, nil
}
func (f *notasFake) UpdateSpecialAdjustmentNote(_ context.Context, n *entity.SpecialAdjustmentNote) (*entity.SpecialAdjustmentNote, error) {
	c := *n
	f.n = &c
	return n, nil
}

// O rascunho se altera mesmo sem a situação no corpo (antes o banco recusava);
// emitir exige código e ICMS; a emitida não se altera; cancelar tira do SPED.
func TestNotaEspecial_Ciclo(t *testing.T) {
	ctx := context.Background()
	f := &notasFake{n: &entity.SpecialAdjustmentNote{ID: 1, Status: entity.SpecialNoteRascunho, Purpose: entity.SpecialNoteAjuste, Period: "2026-11"}}
	uc := &SpecialAdjustmentNoteUseCase{Repo: f}

	if _, err := uc.Update(ctx, &entity.SpecialAdjustmentNote{ID: 1, Purpose: entity.SpecialNoteAjuste, Period: "2026-11", IssueDate: time.Now(), TotalICMS: decimal.NewFromInt(100)}); err != nil {
		t.Fatalf("alterar rascunho: %v", err)
	}
	if f.n.Status != entity.SpecialNoteRascunho {
		t.Fatalf("alterar não pode mudar a situação: %s", f.n.Status)
	}
	if _, err := uc.Emitir(ctx, 1); err == nil {
		t.Fatal("sem código de ajuste não emite")
	}
	cod := int64(7)
	f.n.AdjustmentCodeID = &cod
	f.n.Purpose = entity.SpecialNoteComplementar
	if _, err := uc.Emitir(ctx, 1); err == nil {
		t.Fatal("nota complementar não vai ao SPED: não emite aqui")
	}
	f.n.Purpose = entity.SpecialNoteAjuste
	if _, err := uc.Emitir(ctx, 1); err != nil {
		t.Fatalf("emitir: %v", err)
	}
	if f.n.Status != entity.SpecialNoteEmitida {
		t.Fatalf("situação após emitir: %s", f.n.Status)
	}
	if _, err := uc.Update(ctx, &entity.SpecialAdjustmentNote{ID: 1, Period: "2026-11", IssueDate: time.Now()}); err == nil {
		t.Fatal("a emitida não se altera")
	}
	if _, err := uc.Cancelar(ctx, 1); err != nil || f.n.Status != entity.SpecialNoteCancelada {
		t.Fatalf("cancelar: %v %s", err, f.n.Status)
	}
	if _, err := uc.Cancelar(ctx, 1); err == nil {
		t.Fatal("cancelar duas vezes devia recusar")
	}
}
