package entity

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Baixa automática dos materiais da OF (backflush) — regra única para o
// apontamento da tela, a leitura do coletor e a entrega.
//
// A fonte é a LISTA DE MATERIAIS DA OF, não a estrutura. Quanto sai: a parte da
// quantidade do material proporcional ao que a OF já produziu (no máximo tudo);
// do exigido sai só o que ainda não saiu por esta OF — nunca duas vezes.

// FracaoProduzida: produzido / planejado, entre 0 e 1.
func FracaoProduzida(produzido, planejado decimal.Decimal) decimal.Decimal {
	if !planejado.IsPositive() || produzido.GreaterThanOrEqual(planejado) {
		return decimal.NewFromInt(1)
	}
	if !produzido.IsPositive() {
		return decimal.Zero
	}
	return produzido.Div(planejado)
}

// BaixasAutomaticas monta as saídas que faltam até `fracao` dos materiais.
// O exigido é somado por item (o mesmo item pode estar em duas linhas) e
// comparado com o que já saiu dele por esta OF. deposito, quando informado,
// troca o almoxarifado de todos (escolha do apontador).
func BaixasAutomaticas(materiais []*ProductionOrderMaterial, baixado map[int64]decimal.Decimal, fracao decimal.Decimal,
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
		falta := QuantidadeAindaABaixar(l.exigido.Round(6), baixado[item])
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

// QuantidadeAindaABaixar: o exigido menos o que já saiu, nunca negativo.
func QuantidadeAindaABaixar(exigido, baixado decimal.Decimal) decimal.Decimal {
	return decimal.Max(exigido.Sub(baixado), decimal.Zero)
}
