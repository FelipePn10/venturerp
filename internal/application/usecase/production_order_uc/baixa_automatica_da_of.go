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

// fracaoProduzida: produzido / planejado, entre 0 e 1.
func fracaoProduzida(produzido, planejado decimal.Decimal) decimal.Decimal {
	if !planejado.IsPositive() || produzido.GreaterThanOrEqual(planejado) {
		return decimal.NewFromInt(1)
	}
	if !produzido.IsPositive() {
		return decimal.Zero
	}
	return produzido.Div(planejado)
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

// baixasAutomaticas monta as saídas que faltam até `fracao` dos materiais.
// O exigido é somado por item (o mesmo item pode estar em duas linhas) e
// comparado com o que já saiu dele por esta OF. deposito, quando informado,
// troca o almoxarifado de todos (escolha do apontador).
func baixasAutomaticas(materiais []*entity.ProductionOrderMaterial, baixado map[int64]decimal.Decimal, fracao decimal.Decimal,
	deposito *int64, tipo string, nota *string, ordem int64, ator uuid.UUID) []*stockentity.StockMovement {
	type linha struct {
		item, armazem int64
		mascara       string
		exigido       decimal.Decimal
	}
	ordemDosItens := []int64{}
	porItem := map[int64]*linha{}
	for _, m := range materiais {
		l, ok := porItem[m.ItemCode]
		if !ok {
			l = &linha{item: m.ItemCode, armazem: m.WarehouseID, mascara: m.Mask}
			porItem[m.ItemCode] = l
			ordemDosItens = append(ordemDosItens, m.ItemCode)
		}
		l.exigido = l.exigido.Add(m.Quantity.Mul(fracao))
	}
	refType := stockentity.ReferenceTypeProductionOrder
	refCode := ordem
	var out []*stockentity.StockMovement
	for _, item := range ordemDosItens {
		l := porItem[item]
		falta := quantidadeAindaABaixar(l.exigido.Round(6), baixado[item])
		if !falta.IsPositive() {
			continue
		}
		armazem := l.armazem
		if deposito != nil && *deposito > 0 {
			armazem = *deposito
		}
		out = append(out, &stockentity.StockMovement{ItemCode: item, Mask: l.mascara, WarehouseID: armazem, MovementType: tipo,
			Quantity: falta, ReferenceType: &refType, ReferenceCode: &refCode, Notes: nota, CreatedBy: ator})
	}
	return out
}
