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

// A baixa automática sai da LISTA DE MATERIAIS DA OF: só o marcado
// automático, na proporção do produzido, sem repetir o que já saiu, e o
// substituto (item_code) no lugar do original.
func TestBaixaAutomaticaSaiDaListaDaOF(t *testing.T) {
	original := int64(21)
	materiais := []*poentity.ProductionOrderMaterial{
		{ItemCode: 20, Quantity: decimal.NewFromInt(10), WarehouseID: 7, AutomaticIssue: true},
		{ItemCode: 22, SubstitutedItemCode: &original, Quantity: decimal.NewFromInt(4), WarehouseID: 8, AutomaticIssue: true},
		{ItemCode: 20, Quantity: decimal.NewFromInt(2), WarehouseID: 7, AutomaticIssue: true}, // mesma peça em outra linha
	}
	baixado := map[int64]decimal.Decimal{20: decimal.NewFromInt(3)} // consumo manual anterior
	movs := baixasAutomaticas(materiais, baixado, fracaoProduzida(decimal.NewFromInt(5), decimal.NewFromInt(10)), nil,
		stockentity.MovementTypeOut, nil, 9, uuid.New())
	if len(movs) != 2 {
		t.Fatalf("esperadas 2 saídas, veio %+v", movs)
	}
	if movs[0].ItemCode != 20 || !movs[0].Quantity.Equal(decimal.NewFromInt(3)) || movs[0].WarehouseID != 7 {
		t.Fatalf("item 20: metade de 12 = 6, menos 3 já baixados = 3 no almoxarifado 7; veio %+v", movs[0])
	}
	if movs[1].ItemCode != 22 || !movs[1].Quantity.Equal(decimal.NewFromInt(2)) || movs[1].WarehouseID != 8 {
		t.Fatalf("substituto 22: metade de 4 = 2 no almoxarifado 8; veio %+v", movs[1])
	}
	// Tudo já baixado: nada sai de novo. Depósito escolhido troca o de todos.
	movs = baixasAutomaticas(materiais, map[int64]decimal.Decimal{20: decimal.NewFromInt(12), 22: decimal.NewFromInt(4)},
		decimal.NewFromInt(1), i64p(5), stockentity.MovementTypeOut, nil, 9, uuid.New())
	if len(movs) != 0 {
		t.Fatalf("nada falta baixar, veio %+v", movs)
	}
	movs = baixasAutomaticas(materiais[:1], nil, decimal.NewFromInt(1), i64p(5), stockentity.MovementTypeOut, nil, 9, uuid.New())
	if len(movs) != 1 || movs[0].WarehouseID != 5 {
		t.Fatalf("o depósito do apontador vale para todos, veio %+v", movs)
	}
	if !fracaoProduzida(decimal.NewFromInt(15), decimal.NewFromInt(10)).Equal(decimal.NewFromInt(1)) {
		t.Fatal("produzir além do planejado não baixa mais que a lista")
	}
}

func TestConcluirOFDeBeneficiamentoNaoDaEntradaNoEstoqueProprio(t *testing.T) {
	quantity := decimal.NewFromInt(4)
	po := &fakePORepo{
		order:     &poentity.ProductionOrder{ID: 9, ItemCode: 100, PlannedQty: decimal.NewFromInt(4), CreatedBy: uuid.New()},
		automatic: map[int64]int64{200: 9, 300: 9},
		// A OF de beneficiamento só lista o material próprio (o do cliente fica
		// fora na criação — ver o teste acima).
		materials: []*poentity.ProductionOrderMaterial{{ItemCode: 200, Quantity: decimal.NewFromInt(8), WarehouseID: 9, AutomaticIssue: true}},
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
