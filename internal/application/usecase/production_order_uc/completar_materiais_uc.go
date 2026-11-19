package production_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

type completadorDeMateriais interface {
	CompletarMateriais(ctx context.Context, ordemID int64, materiais []*entity.ProductionOrderMaterial) (int, error)
}

// CompletarMateriaisDaOFUseCase dá à OF que nasceu sem lista de materiais a
// lista da estrutura vigente do item, pela mesma conta da criação da OF
// (MateriaisDaEstrutura). Toda OF aberta passa a mostrar — na tela e na OP
// impressa — o que vai consumir, e a baixa automática passa a valer para ela.
type CompletarMateriaisDaOFUseCase struct {
	Montador *CreateProductionOrderUseCase
	Repo     completadorDeMateriais
}

// Completar grava a lista da OF e devolve quantos materiais entraram (0 quando
// o item não tem componente vigente ou a OF já ganhou a lista). A sessão do
// contexto precisa ser da empresa da OF.
func (uc *CompletarMateriaisDaOFUseCase) Completar(ctx context.Context, of entity.OFSemMateriais) (int, error) {
	consumo := time.Now()
	if of.StartDate != nil {
		consumo = *of.StartDate
	}
	materiais, _, err := uc.Montador.MateriaisDaEstrutura(ctx, of.ItemCode, of.Mask, of.PlannedQty, consumo, of.CreatedBy)
	if err != nil {
		return 0, err
	}
	if len(materiais) == 0 {
		return 0, nil
	}
	marcarBaixaDaListaCompletada(materiais, of.JaProduziu)
	return uc.Repo.CompletarMateriais(ctx, of.ID, materiais)
}

// marcarBaixaDaListaCompletada anota a origem da lista e, se a OF já
// produziu, tira a baixa automática: ela sai na proporção do TOTAL produzido e
// o próximo apontamento lançaria hoje, de uma vez, o material de tudo o que foi
// feito antes. A lista entra para mostrar o consumo; a saída fica manual
// (requisição ou consumo). Devolve quantos ficaram com baixa automática.
func marcarBaixaDaListaCompletada(materiais []*entity.ProductionOrderMaterial, jaProduziu bool) int {
	nota := "lista completada pela estrutura vigente (a OF não tinha lista de materiais)"
	if jaProduziu {
		nota = "lista completada pela estrutura vigente com a OF já em produção: baixa manual, para não lançar de uma vez o consumo do que já foi produzido"
	}
	automaticas := 0
	for _, m := range materiais {
		m.Notes = &nota
		if jaProduziu {
			m.AutomaticIssue = false
		}
		if m.AutomaticIssue {
			automaticas++
		}
	}
	return automaticas
}
