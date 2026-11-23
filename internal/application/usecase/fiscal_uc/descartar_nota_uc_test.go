package fiscal_uc

import (
	"context"
	"testing"

	"github.com/google/uuid"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type notaDescartavel struct {
	exit       *fiscalentity.FiscalExit
	descartada bool
}

func (n *notaDescartavel) GetExitByID(context.Context, int64) (*fiscalentity.FiscalExit, error) {
	return n.exit, nil
}
func (n *notaDescartavel) DescartarRascunho(context.Context, int64, string, uuid.UUID) (bool, error) {
	n.descartada = true
	n.exit.Status = fiscalentity.ExitStatusCancelled
	return true, nil
}

type romaneioDescarte struct{ desligado bool }

func (r *romaneioDescarte) ReabrirPelaNotaCancelada(context.Context, int64) error {
	r.desligado = true
	return nil
}

// Bug W: rascunho é descartado e solta o romaneio; autorizada, não.
func TestDescartarNota(t *testing.T) {
	notas := &notaDescartavel{exit: &fiscalentity.FiscalExit{ID: 9, Status: fiscalentity.ExitStatusDraft}}
	rom := &romaneioDescarte{}
	uc := &DescartarNotaUseCase{Auth: authFiscalLiberado{}, Notas: notas, Romaneios: rom}
	if _, err := uc.Execute(context.Background(), 9, ""); err != nil {
		t.Fatal(err)
	}
	if !notas.descartada || !rom.desligado {
		t.Fatalf("descartada=%v romaneio desligado=%v", notas.descartada, rom.desligado)
	}

	notas = &notaDescartavel{exit: &fiscalentity.FiscalExit{ID: 9, Status: fiscalentity.ExitStatusAuthorized}}
	uc.Notas = notas
	if _, err := uc.Execute(context.Background(), 9, ""); err == nil || notas.descartada {
		t.Fatal("nota autorizada foi descartada")
	}
}
