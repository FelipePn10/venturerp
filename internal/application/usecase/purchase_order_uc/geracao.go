package purchase_order_uc

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// Geracao é o fim comum de todo pedido que nasce de outro fluxo (requisição,
// cotação): as linhas recebem o almoxarifado do cadastro do item e o pedido
// passa pela mesma aprovação (alçada) que o pedido digitado. Antes esses
// pedidos nasciam "aprovados" sem avaliar valor nenhum.
type Geracao struct {
	Repo          repository.PurchaseOrderRepository
	Almoxarifados AlmoxarifadoPadrao
	Aprovacao     *ApprovePurchaseOrderUseCase
}

// NovaCapa prepara a capa gerada: rascunho, alçada ainda não avaliada.
func (g *Geracao) NovaCapa(o *entity.PurchaseOrder) {
	o.Status = entity.PurchaseOrderStatusDRAFT
	o.AlcadaStatus = "N"
}

// CompletarLinhas preenche almoxarifado e total de cada linha. Linha cujo item
// não tem almoxarifado no cadastro fica sem e o aviso volta para a tela.
func (g *Geracao) CompletarLinhas(ctx context.Context, itens []*entity.PurchaseOrderItem) ([]string, error) {
	var avisos []string
	for _, it := range itens {
		it.TotalPrice = entity.TotalDaLinha(it.RequestedQty, it.UnitPrice, it.DiscountPct)
		if it.WarehouseID != nil && *it.WarehouseID > 0 {
			continue
		}
		if g == nil || g.Almoxarifados == nil {
			continue
		}
		wh, err := g.Almoxarifados.AlmoxarifadoPadraoDoItem(ctx, it.ItemCode)
		if err != nil {
			return nil, err
		}
		if wh == nil {
			avisos = append(avisos, fmt.Sprintf("item %d sem almoxarifado de suprimentos no cadastro: informe o almoxarifado da linha em VPDC0200", it.ItemCode))
			continue
		}
		it.WarehouseID = wh
	}
	return avisos, nil
}

// Aprovar passa o pedido gerado pela alçada e devolve a capa atualizada. Uma
// recusa de validação (linha sem almoxarifado, por exemplo) não desfaz o
// pedido: ele fica em rascunho e o motivo volta como aviso.
func (g *Geracao) Aprovar(ctx context.Context, po *entity.PurchaseOrder) (*entity.PurchaseOrder, string, error) {
	if g == nil || g.Aprovacao == nil {
		return po, fmt.Sprintf("pedido %d gerado em rascunho: aprove em VPDC0200", po.Code), nil
	}
	r, err := g.Aprovacao.Execute(ctx, po.Code)
	if err != nil {
		var v *errorsuc.ValidationError
		if errors.As(err, &v) {
			return po, fmt.Sprintf("pedido %d ficou em rascunho: %s", po.Code, err.Error()), nil
		}
		return nil, "", err
	}
	atual, err := g.Repo.GetByCode(ctx, po.Code)
	if err != nil {
		return nil, "", err
	}
	atual.Items = po.Items
	aviso := ""
	if r.AlcadaStatus == alcadaBlocked || r.AlcadaStatus == alcadaRejected {
		aviso = fmt.Sprintf("pedido %d acima da alçada do comprador: aguarda autorização em VPDC0200", po.Code)
	}
	return atual, aviso, nil
}
