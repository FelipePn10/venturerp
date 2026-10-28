package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type sequenciaDeSaida interface {
	SequenciaDoItemDeSaida(ctx context.Context, itemID int64) (int, error)
}

// referenciarDevolucaoDeVenda liga cada item da NF-e de entrada da devolução de
// venda ao item da NF de saída devolvida (grupo DFeReferenciado: chave + nItem).
// Desde a reforma a SEFAZ recusa a devolução sem ele (rejeição 321
// "NF-e de devolução não possui documento fiscal referenciado [nItem]").
func referenciarDevolucaoDeVenda(ctx context.Context, repo any, exit *entity.FiscalExit, items []*entity.FiscalExitItem, payload *focusnfe.NFEPayload) error {
	if exit.Finalidade != 4 || exit.DevolucaoVendaID == nil || exit.NFeReferenciada == nil || *exit.NFeReferenciada == "" {
		return nil
	}
	r, ok := repo.(sequenciaDeSaida)
	if !ok {
		return nil
	}
	for i, x := range items {
		if i >= len(payload.Items) || x.FiscalExitItemOrigemID == nil {
			continue
		}
		n, err := r.SequenciaDoItemDeSaida(ctx, *x.FiscalExitItemOrigemID)
		if err != nil {
			return err
		}
		payload.Items[i].DFeRefChave = *exit.NFeReferenciada
		payload.Items[i].DFeRefItem = n
	}
	return nil
}
