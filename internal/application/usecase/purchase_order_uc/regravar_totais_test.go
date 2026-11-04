package purchase_order_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type regravarRepo struct {
	porepo.PurchaseOrderRepository
	itens []*poentity.PurchaseOrderItem
	salvo *poentity.PurchaseOrder
}

func (r *regravarRepo) ListItems(context.Context, int64) ([]*poentity.PurchaseOrderItem, error) {
	return r.itens, nil
}

func (r *regravarRepo) Update(_ context.Context, o *poentity.PurchaseOrder) (*poentity.PurchaseOrder, error) {
	r.salvo = o
	return o, nil
}

// Editar a capa de um pedido aprovado sem mudar o total não devolve o pedido à
// alçada. Com o total em decimal, "o.TotalNet != antes" comparava o ponteiro
// interno do decimal — sempre diferente — e qualquer edição zerava a alçada.
func TestRegravarComTotaisMantemAlcadaQuandoOTotalNaoMuda(t *testing.T) {
	d := decimal.RequireFromString
	itens := []*poentity.PurchaseOrderItem{{Code: 1, IsActive: true, RequestedQty: d("3"), UnitPrice: d("10.05")}}
	o := &poentity.PurchaseOrder{Code: 7, Status: poentity.PurchaseOrderStatusAPPROVED, AlcadaStatus: "A"}
	o.AplicarTotais(poentity.CalcularTotais(o, itens))

	repo := &regravarRepo{itens: itens}
	if _, err := regravarComTotais(context.Background(), repo, o); err != nil {
		t.Fatal(err)
	}
	if repo.salvo.AlcadaStatus != "A" || repo.salvo.Status != poentity.PurchaseOrderStatusAPPROVED {
		t.Fatalf("pedido sem mudança de valor voltou à alçada: %s/%s", repo.salvo.Status, repo.salvo.AlcadaStatus)
	}

	// Valor mudou: aí sim volta para a alçada.
	repo.itens = []*poentity.PurchaseOrderItem{{Code: 1, IsActive: true, RequestedQty: d("4"), UnitPrice: d("10.05")}}
	if _, err := regravarComTotais(context.Background(), repo, o); err != nil {
		t.Fatal(err)
	}
	if repo.salvo.AlcadaStatus != "N" {
		t.Fatalf("pedido com valor novo manteve a alçada: %s", repo.salvo.AlcadaStatus)
	}
}
