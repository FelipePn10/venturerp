package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

// UnidadeDeEstoque diz em que unidade o item é estocado (cadastro do item).
type UnidadeDeEstoque interface {
	UnidadeDeEstoqueDoItem(ctx context.Context, itemCode int64) (string, error)
}

// UnidadeInterna completa a linha do pedido com a quantidade e o preço na
// unidade de ESTOQUE, que é o que o recebimento e a nota de entrada usam.
//
// Antes a conversão era opcional e silenciosa: sem conversão cadastrada (ou com
// erro ao ler), a quantidade de compra virava a quantidade interna — 100 KG de
// tubo entravam como 100 METROS. E a unidade interna vinha de quem chamava, não
// do cadastro, então duas telas podiam gravar unidades internas diferentes
// para o mesmo item. As linhas geradas por requisição, cotação e sugestão nem
// tinham quantidade interna.
type UnidadeInterna struct {
	Unidades  UnidadeDeEstoque
	Conversor ports.UOMConverter
}

// Preencher define InternalUOM (a de estoque do cadastro), InternalQty e
// InternalPrice. Linha sem unidade de compra está na unidade de estoque.
// Unidade de compra diferente sem conversão é recusada.
func (u *UnidadeInterna) Preencher(ctx context.Context, l *entity.PurchaseOrderItem) error {
	if u == nil || u.Unidades == nil {
		if !l.InternalQty.IsPositive() {
			l.InternalQty = l.RequestedQty
		}
		if !l.InternalPrice.IsPositive() {
			l.InternalPrice = l.UnitPrice
		}
		return nil
	}
	estoque, err := u.Unidades.UnidadeDeEstoqueDoItem(ctx, l.ItemCode)
	if err != nil {
		return err
	}
	estoque = strings.ToUpper(strings.TrimSpace(estoque))
	if estoque != "" {
		l.InternalUOM = &estoque
	}
	compra := ""
	if l.PurchaseUOM != nil {
		compra = strings.ToUpper(strings.TrimSpace(*l.PurchaseUOM))
	}
	if compra == "" && estoque != "" {
		compra = estoque
	}
	if compra != "" {
		l.PurchaseUOM = &compra
	}
	if compra == "" || estoque == "" || compra == estoque {
		l.InternalQty, l.InternalPrice = l.RequestedQty, l.UnitPrice
		return nil
	}
	semConversao := errorsuc.NewValidationError(fmt.Sprintf(
		"o item %d é estocado em %s e a linha está em %s: cadastre a conversão %s→%s do item (VSUP0110) antes de comprar nessa unidade",
		l.ItemCode, estoque, compra, compra, estoque))
	if u.Conversor == nil {
		return semConversao
	}
	qf, achou, err := u.Conversor.ConvertQuantity(ctx, l.ItemCode, l.RequestedQty, compra, estoque)
	if err != nil {
		return err
	}
	q := qf
	if !achou || !q.Round(6).IsPositive() {
		return semConversao
	}
	// A quantidade gravada tem 6 casas (escala da coluna); o preço é dividido
	// pela quantidade exata convertida — dividir pela arredondada mudava o preço
	// (115,8975 KG = 17,9999981 M: 723,90 ÷ 18 ≠ 723,90 ÷ 17,9999981).
	l.InternalQty = q.Round(6)
	// Preço por unidade de estoque: o mesmo valor da linha dividido pela
	// quantidade convertida (não depende de uma segunda leitura do fator).
	l.InternalPrice = l.UnitPrice.Mul(l.RequestedQty).DivRound(q, 10) // escala da coluna internal_price
	return nil
}
