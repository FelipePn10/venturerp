package fiscal_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type seqFake map[int64]int

func (s seqFake) SequenciaDoItemDeSaida(_ context.Context, id int64) (int, error) { return s[id], nil }

// A NF-e de entrada da devolução de venda referencia a saída POR ITEM — sem
// isso a SEFAZ rejeitava com 321 (visto na homologação da Focus, 2026-10-26).
func TestReferenciarDevolucaoDeVendaPorItem(t *testing.T) {
	chave, dev, o1, o2 := "41261052454668000102550010000000371968536900", int64(14), int64(2502), int64(2503)
	exit := &entity.FiscalExit{Finalidade: 4, DevolucaoVendaID: &dev, NFeReferenciada: &chave}
	items := []*entity.FiscalExitItem{{FiscalExitItemOrigemID: &o2}, {FiscalExitItemOrigemID: &o1}}
	p := &focusnfe.NFEPayload{Items: make([]focusnfe.NFEItem, 2)}
	if err := referenciarDevolucaoDeVenda(context.Background(), seqFake{o1: 1, o2: 3}, exit, items, p); err != nil {
		t.Fatal(err)
	}
	if p.Items[0].DFeRefChave != chave || p.Items[0].DFeRefItem != 3 || p.Items[1].DFeRefItem != 1 {
		t.Fatalf("referência por item: %+v", p.Items)
	}
	// Nota que não é devolução de venda não ganha referência.
	q := &focusnfe.NFEPayload{Items: make([]focusnfe.NFEItem, 2)}
	if err := referenciarDevolucaoDeVenda(context.Background(), seqFake{}, &entity.FiscalExit{Finalidade: 1}, items, q); err != nil || q.Items[0].DFeRefChave != "" {
		t.Fatalf("venda normal: %+v %v", q.Items, err)
	}
}
