package production_order_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

// Beneficiamento: a estrutura do produto processado leva o material que o
// cliente manda (CFOP 5901). Esse material não é da empresa — a OF não pode
// pedi-lo ao estoque, consumi-lo nem dar entrada do produto no estoque próprio.

func TestCriarOFNaoGeraDemandaDoMaterialDoCliente(t *testing.T) {
	repo := &fakePORepo{automatic: map[int64]int64{20: 2}}
	structure := &fakeDeliveryStructure{children: []*structentity.ItemStructure{
		{ChildCode: 20, Quantity: decimal.NewFromFloat(1)},                         // parafuso da empresa
		{ChildCode: 30, Quantity: decimal.NewFromFloat(2), CustomerSupplied: true}, // tubo do cliente
	}}
	uc := CreateProductionOrderUseCase{
		Repo: repo, Auth: fakeAuth{canPlanned: true}, Structure: structure,
		Items: &fakeOrderItems{item: &itementity.Item{Code: 10, BusinessCode: "BEN-10"}},
	}
	if _, err := uc.Execute(context.Background(), request.CreateProductionOrderDTO{ItemCode: "BEN-10", PlannedQty: decimal.NewFromInt(5), CreatedBy: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	if len(repo.createdMaterials) != 1 || repo.createdMaterials[0].ItemCode != 20 {
		t.Fatalf("a OF deve listar só o material próprio, veio %+v", repo.createdMaterials)
	}
}

func TestBaixaAutomaticaNaoConsomeMaterialDoCliente(t *testing.T) {
	children := structureChildrenForBackflush([]*structentity.ItemStructure{
		{ChildCode: 20, Quantity: decimal.NewFromFloat(1)},
		{ChildCode: 30, Quantity: decimal.NewFromFloat(2), CustomerSupplied: true},
	})
	if len(children) != 1 || children[0].code != 20 {
		t.Fatalf("o apontamento só pode baixar o material próprio, veio %+v", children)
	}
}

func TestConcluirOFDeBeneficiamentoNaoDaEntradaNoEstoqueProprio(t *testing.T) {
	quantity := decimal.NewFromInt(4)
	po := &fakePORepo{
		order:     &poentity.ProductionOrder{ID: 9, ItemCode: 100, PlannedQty: decimal.NewFromInt(4), CreatedBy: uuid.New()},
		automatic: map[int64]int64{200: 9, 300: 9},
	}
	stock := &fakeStockRepo{}
	structure := &fakeDeliveryStructure{children: []*structentity.ItemStructure{
		{ChildCode: 200, Quantity: decimal.NewFromFloat(2)},                         // material próprio, baixa automática
		{ChildCode: 300, Quantity: decimal.NewFromFloat(1), CustomerSupplied: true}, // material do cliente
		{ChildCode: 400, Quantity: decimal.NewFromFloat(1), IsCoproduct: true},      // cavaco do material do cliente
	}}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock, Structure: structure}
	if _, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{ID: 9, Quantity: &quantity, WarehouseID: i64p(4), IdempotencyKey: "ben-9"}); err != nil {
		t.Fatal(err)
	}
	if po.delivery == nil || !po.delivery.Quantity.Equal(quantity) {
		t.Fatalf("a entrega da OF tem de ficar registrada: %+v", po.delivery)
	}
	if len(stock.movements) != 1 {
		t.Fatalf("esperado só o REP do material próprio, veio %+v", stock.movements)
	}
	m := stock.movements[0]
	if m.ItemCode != 200 || m.MovementType != stockentity.MovementTypePlannedRequisition {
		t.Fatalf("movimento inesperado: %+v", m)
	}
}

func TestConcluirOFFalhaSeNaoLeAEstrutura(t *testing.T) {
	quantity := decimal.NewFromInt(1)
	po := &fakePORepo{order: &poentity.ProductionOrder{ID: 3, ItemCode: 100, PlannedQty: decimal.NewFromInt(1), CreatedBy: uuid.New()}}
	stock := &fakeStockRepo{}
	uc := CompleteProductionOrderUseCase{Repo: po, Auth: fakeAuth{canSales: true}, StockRepo: stock, Structure: estruturaQueFalha{}}
	_, err := uc.Execute(context.Background(), request.CompleteProductionOrderDTO{ID: 3, Quantity: &quantity, WarehouseID: i64p(4), IdempotencyKey: "ben-3"})
	if err == nil {
		t.Fatal("sem a estrutura não dá para saber se a OF é de beneficiamento: tem de falhar")
	}
	if len(stock.movements) != 0 || po.delivery != nil {
		t.Fatalf("nada pode ser gravado: movimentos=%+v entrega=%+v", stock.movements, po.delivery)
	}
}

type estruturaQueFalha struct{}

func (estruturaQueFalha) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	return nil, errors.New("banco fora")
}
