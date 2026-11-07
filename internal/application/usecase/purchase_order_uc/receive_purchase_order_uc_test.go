package purchase_order_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type receiveAuth struct {
	ports.AuthService
	canPO    bool
	canStock bool
	uid      uuid.UUID
}

func (a receiveAuth) CanUpdatePurchaseOrder(context.Context) bool { return a.canPO }
func (a receiveAuth) CanCreateStockMovement(context.Context) bool { return a.canStock }
func (a receiveAuth) UserID(context.Context) (uuid.UUID, error)   { return a.uid, nil }

type receivePORepo struct {
	porepo.PurchaseOrderRepository
	order      *poentity.PurchaseOrder
	items      []*poentity.PurchaseOrderItem
	registered map[int64]decimal.Decimal
}

func (r *receivePORepo) GetByCode(context.Context, int64) (*poentity.PurchaseOrder, error) {
	return r.order, nil
}

func (r *receivePORepo) ListItems(context.Context, int64) ([]*poentity.PurchaseOrderItem, error) {
	return r.items, nil
}

func (r *receivePORepo) RegisterItemReceipts(_ context.Context, _ int64, receivedByOrderItemCode map[int64]decimal.Decimal) (int, error) {
	r.registered = receivedByOrderItemCode
	return len(receivedByOrderItemCode), nil
}

type receiveStockRepo struct {
	stockrepo.StockRepository
	movements []*stockentity.StockMovement
	err       error
}

type receiveTolerance struct {
	action   string
	exceeded bool
}

func (f receiveTolerance) EvaluatePurchaseTolerance(context.Context, *int64, string, string, decimal.Decimal, decimal.Decimal) (string, string, bool, error) {
	return f.action, "configured tolerance", f.exceeded, nil
}

func (r *receiveStockRepo) CreateMovement(_ context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	if r.err != nil {
		return nil, r.err
	}
	m.ID = int64(len(r.movements) + 1)
	m.CreatedAt = time.Now()
	r.movements = append(r.movements, m)
	return m, nil
}

func receiveDTO() request.ReceivePurchaseOrderDTO {
	return request.ReceivePurchaseOrderDTO{
		PurchaseOrderCode: 10,
		Items: []request.ReceivePurchaseOrderItemDTO{{
			PurchaseOrderItemCode: 55,
			Quantity:              decimal.NewFromFloat(4),
			WarehouseID:           2,
		}},
	}
}

func receiveRepo() *receivePORepo {
	return &receivePORepo{
		order: &poentity.PurchaseOrder{Code: 10, Status: poentity.PurchaseOrderStatusAPPROVED},
		items: []*poentity.PurchaseOrderItem{{
			Code:          55,
			ItemCode:      1001,
			Mask:          "AZ",
			RequestedQty:  decimal.NewFromFloat(10),
			ReceivedQty:   decimal.NewFromFloat(2),
			UnitPrice:     decimal.NewFromFloat(7),
			InternalQty:   decimal.NewFromFloat(20),
			InternalPrice: decimal.NewFromFloat(3.5),
			Status:        poentity.PurchaseOrderItemStatusPARTIAL,
		}},
	}
}

func TestReceivePurchaseOrderUnauthorized(t *testing.T) {
	repo := receiveRepo()
	uc := ReceivePurchaseOrderUseCase{
		Repo:      repo,
		StockRepo: &receiveStockRepo{},
		Auth:      receiveAuth{canPO: false, canStock: true, uid: uuid.New()},
	}

	_, err := uc.Execute(context.Background(), receiveDTO())
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestReceivePurchaseOrderCreatesInboundMovementAndRegistersLine(t *testing.T) {
	repo := receiveRepo()
	stock := &receiveStockRepo{}
	uid := uuid.New()
	uc := ReceivePurchaseOrderUseCase{
		Repo:      repo,
		StockRepo: stock,
		Auth:      receiveAuth{canPO: true, canStock: true, uid: uid},
	}

	out, err := uc.Execute(context.Background(), receiveDTO())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || len(out.ReceivedLines) != 1 || len(out.Movements) != 1 {
		t.Fatalf("unexpected response: %+v", out)
	}
	if got := repo.registered[55]; !got.Equal(decimal.NewFromFloat(4)) {
		t.Fatalf("registered qty = %v, want 4", got)
	}
	mov := stock.movements[0]
	if mov.MovementType != stockentity.MovementTypeIn {
		t.Fatalf("movement type = %s, want IN", mov.MovementType)
	}
	if !mov.Quantity.Equal(decimal.NewFromFloat(8)) {
		t.Fatalf("stock quantity = %v, want converted quantity 8", mov.Quantity)
	}
	if !mov.UnitPrice.Equal(decimal.NewFromFloat(3.5)) || !mov.TotalPrice.Equal(decimal.NewFromFloat(28)) {
		t.Fatalf("cost values wrong: unit=%v total=%v", mov.UnitPrice, mov.TotalPrice)
	}
	if mov.ReferenceType == nil || *mov.ReferenceType != stockentity.ReferenceTypePurchaseOrder {
		t.Fatalf("reference type = %v, want PURCHASE_ORDER", mov.ReferenceType)
	}
	if mov.ReferenceCode == nil || *mov.ReferenceCode != 10 {
		t.Fatalf("reference code = %v, want 10", mov.ReferenceCode)
	}
	if mov.CreatedBy != uid {
		t.Fatalf("created_by = %v, want %v", mov.CreatedBy, uid)
	}
}

func TestReceivePurchaseOrderRejectsExcessQuantity(t *testing.T) {
	repo := receiveRepo()
	stock := &receiveStockRepo{}
	dto := receiveDTO()
	dto.Items[0].Quantity = decimal.NewFromFloat(99)
	uc := ReceivePurchaseOrderUseCase{
		Repo:      repo,
		StockRepo: stock,
		Auth:      receiveAuth{canPO: true, canStock: true, uid: uuid.New()},
	}

	_, err := uc.Execute(context.Background(), dto)
	if err == nil {
		t.Fatal("expected error for excess quantity")
	}
	if len(stock.movements) != 0 {
		t.Fatal("stock movement must not be created when validation fails")
	}
}

func TestReceivePurchaseOrderConfiguredToleranceWarnsOrBlocks(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		wantErr      bool
	}{{"warn", "WARN", false}, {"block", "BLOCK", true}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := receiveRepo()
			stock := &receiveStockRepo{}
			dto := receiveDTO()
			dto.Items[0].Quantity = decimal.NewFromFloat(9)
			uc := ReceivePurchaseOrderUseCase{Repo: repo, StockRepo: stock, Auth: receiveAuth{canPO: true, canStock: true, uid: uuid.New()}, Tolerances: receiveTolerance{action: tc.action, exceeded: true}}
			out, err := uc.Execute(context.Background(), dto)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr && len(stock.movements) != 0 {
				t.Fatal("blocked receipt created stock")
			}
			if !tc.wantErr && (out == nil || len(out.Warnings) != 1) {
				t.Fatalf("warning missing: %+v", out)
			}
		})
	}
}

func (r *receivePORepo) EncerrarSaldoPorTolerancia(context.Context, int64, map[int64]decimal.Decimal) error {
	return nil
}

type regraFixa struct{ regra RegraDoItem }

func (r regraFixa) RegraDeRecebimento(context.Context, int64) (RegraDoItem, error) {
	return r.regra, nil
}

func TestToleranciaOlhaOTotalDosLotesDaLinha(t *testing.T) {
	dois := decimal.NewFromInt(2)
	dto := func(qs ...int64) request.ReceivePurchaseOrderDTO {
		d := request.ReceivePurchaseOrderDTO{PurchaseOrderCode: 10}
		for _, q := range qs {
			d.Items = append(d.Items, request.ReceivePurchaseOrderItemDTO{PurchaseOrderItemCode: 55, Quantity: decimal.NewFromInt(q), WarehouseID: 2})
		}
		return d
	}
	novo := func(regra RegraDoItem) ReceivePurchaseOrderUseCase {
		repo := receiveRepo()
		repo.items[0].RequestedQty, repo.items[0].ReceivedQty = decimal.NewFromInt(100), decimal.Zero
		repo.items[0].InternalQty = decimal.Zero
		return ReceivePurchaseOrderUseCase{Repo: repo, StockRepo: &receiveStockRepo{}, Regras: regraFixa{regra}, Auth: receiveAuth{canPO: true, canStock: true}}
	}
	uc := novo(RegraDoItem{ToleranciaExcessoPct: &dois})
	if _, err := uc.Execute(context.Background(), dto(60, 60)); err == nil {
		t.Fatal("dois lotes de 60 passam dos 102 da tolerância, mesmo cada um cabendo sozinho")
	}
	uc = novo(RegraDoItem{ToleranciaFaltaPct: &dois})
	out, err := uc.Execute(context.Background(), dto(98, 2))
	if err != nil {
		t.Fatal(err)
	}
	for _, aviso := range out.Warnings {
		t.Fatalf("98 + 2 = 100, não há falta para encerrar: %s", aviso)
	}
	uc = novo(RegraDoItem{ToleranciaFaltaPct: &dois})
	out, err = uc.Execute(context.Background(), dto(49, 50))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Warnings) != 1 {
		t.Fatalf("falta de 1 no total, dentro de 2%%, encerra a linha uma vez: %v", out.Warnings)
	}
}
