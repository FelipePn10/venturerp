package production_order_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

type completadorDuble struct {
	gravados []*entity.ProductionOrderMaterial
}

func (c *completadorDuble) CompletarMateriais(_ context.Context, _ int64, m []*entity.ProductionOrderMaterial) (int, error) {
	c.gravados = m
	return len(m), nil
}

// A lista completada numa OF que já produziu entra com baixa MANUAL: a baixa
// automática sai na proporção do total produzido e lançaria hoje, de uma vez,
// o material de tudo o que foi feito antes da lista existir.
func TestListaCompletadaEmOFJaProduzidaFicaComBaixaManual(t *testing.T) {
	for _, jaProduziu := range []bool{false, true} {
		materiais := []*entity.ProductionOrderMaterial{{ItemCode: 1, AutomaticIssue: true}, {ItemCode: 2, AutomaticIssue: false}}
		automaticas := marcarBaixaDaListaCompletada(materiais, jaProduziu)
		if jaProduziu {
			for _, m := range materiais {
				if m.AutomaticIssue {
					t.Fatalf("OF já produzida: o material %d ficou com baixa automática", m.ItemCode)
				}
			}
			if automaticas != 0 {
				t.Fatalf("esperava 0 automáticas, veio %d", automaticas)
			}
		} else if !materiais[0].AutomaticIssue || materiais[1].AutomaticIssue || automaticas != 1 {
			t.Fatalf("OF sem produção deveria manter o padrão do cadastro: %+v", materiais)
		}
		if materiais[0].Notes == nil {
			t.Fatal("a nota da lista completada não foi gravada")
		}
	}
}
