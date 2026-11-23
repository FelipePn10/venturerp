package production_order_uc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Baixa automática dos materiais da OF (backflush).
//
// A fonte é a LISTA DE MATERIAIS DA OF (aba Materiais), não a estrutura: ela
// já veio da estrutura vigente na criação e guarda o que mudou depois —
// substituto, material incluído ou retirado, almoxarifado. Sai só o material
// marcado "baixa automática" (padrão do cadastro do item; desmarcado = sai
// por requisição ou pelo consumo manual).
//
// Quanto sai: a parte da quantidade do material proporcional ao que a OF já
// produziu (bom apontado, ou entregue, sobre o planejado, no máximo tudo);
// na entrega final, tudo. Do exigido sai só o que ainda não saiu por esta OF
// (apontamento anterior, consumo manual, requisição) — nunca duas vezes.

type materiaisDaOrdem interface {
	ListMaterials(ctx context.Context, productionOrderID int64, kind entity.MaterialKind) ([]*entity.ProductionOrderMaterial, error)
}

// fracaoProduzida e baixasAutomaticas: a regra é do domínio da OF (a leitura
// do coletor também baixa) — ver entity.FracaoProduzida.
func fracaoProduzida(produzido, planejado decimal.Decimal) decimal.Decimal {
	return entity.FracaoProduzida(produzido, planejado)
}

// lerMateriaisAutomaticos devolve os materiais da OF com baixa automática.
// Repositório sem a lista (dublê de teste) devolve nenhum.
func lerMateriaisAutomaticos(ctx context.Context, repo repository.ProductionOrderRepository, ordem int64) ([]*entity.ProductionOrderMaterial, error) {
	m, ok := repo.(materiaisDaOrdem)
	if !ok {
		return nil, nil
	}
	lista, err := m.ListMaterials(ctx, ordem, entity.MaterialDemand)
	if err != nil {
		return nil, fmt.Errorf("lendo os materiais da OF para a baixa automática: %w", err)
	}
	out := make([]*entity.ProductionOrderMaterial, 0, len(lista))
	for _, x := range lista {
		if x.AutomaticIssue && x.Quantity.IsPositive() {
			out = append(out, x)
		}
	}
	return out, nil
}

func baixasAutomaticas(materiais []*entity.ProductionOrderMaterial, baixado map[int64]decimal.Decimal, fracao decimal.Decimal,
	deposito *int64, tipo string, nota *string, ordem int64, ator uuid.UUID) []*stockentity.StockMovement {
	return entity.BaixasAutomaticas(materiais, baixado, fracao, deposito, tipo, nota, ordem, ator)
}
