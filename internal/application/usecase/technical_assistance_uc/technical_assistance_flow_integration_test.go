package technical_assistance_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	prodentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	prodrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
)

type assistanceSalesOrders struct {
	orderrepo.SalesOrderRepository
	order *orderentity.SalesOrder
	items []*orderentity.SalesOrderItem
}

func (f *assistanceSalesOrders) NextOrderNumber(context.Context, int64) (int64, error) {
	return 700, nil
}
func (f *assistanceSalesOrders) Create(_ context.Context, v *orderentity.SalesOrder) (*orderentity.SalesOrder, error) {
	v.Code = 701
	f.order = v
	return v, nil
}
func (f *assistanceSalesOrders) CreateItem(_ context.Context, v *orderentity.SalesOrderItem) (*orderentity.SalesOrderItem, error) {
	f.items = append(f.items, v)
	return v, nil
}

type assistanceProductionOrders struct {
	prodrepo.ProductionOrderRepository
	order *prodentity.ProductionOrder
}

func (f *assistanceProductionOrders) GetNextOrderNumber(context.Context) (int64, error) {
	return 800, nil
}
func (f *assistanceProductionOrders) Create(_ context.Context, v *prodentity.ProductionOrder) (*prodentity.ProductionOrder, error) {
	v.ID = 801
	f.order = v
	return v, nil
}

func TestTechnicalAssistanceReturnNoteAndOrderGenerationFlow(t *testing.T) {
	salesReason, productionReason := int64(31), int64(32)
	repo := &fakeTARepo{reasons: map[int64]*entity.DefectReason{salesReason: {Code: salesReason, GeneratesSalesOrder: true, RequiresReturnNote: true}, productionReason: {Code: productionReason, GeneratesProductionOrder: true}}, call: &entity.Call{Code: 1, CallNumber: 10, EnterpriseCode: 1, CustomerCode: 20, Status: entity.CallStatusPending, OpenedAt: time.Now()}, items: []*entity.CallItem{{Code: 1, ItemCode: 100, Quantity: 1, DefectReasonCode: &salesReason}, {Code: 2, ItemCode: 200, Quantity: 2, DefectReasonCode: &productionReason}}}
	sales := &assistanceSalesOrders{}
	production := &assistanceProductionOrders{}
	uc := &UseCase{Repo: repo, Auth: taAllowAuth{}, SalesOrders: sales, ProductionOrders: production}
	if _, err := uc.AddReturnNote(context.Background(), request.AddTechnicalAssistanceReturnNoteDTO{CallCode: 1, NoteNumber: "123", EmissionDate: "2026-08-27"}); err != nil {
		t.Fatalf("nota de retorno: %v", err)
	}
	division, table, payment, warehouse := int64(1), int64(2), int64(3), int64(4)
	result, err := uc.GenerateOrders(context.Background(), request.GenerateTechnicalAssistanceOrdersDTO{CallCode: 1, SalesDivisionCode: &division, PriceTableCode: &table, PaymentTermCode: &payment, WarehouseCode: &warehouse})
	if err != nil {
		t.Fatal(err)
	}
	if result.SalesOrderCode == nil || *result.SalesOrderCode != 701 || result.ProductionOrderID == nil || *result.ProductionOrderID != 801 || result.GeneratedLinks != 2 {
		t.Fatalf("resultado inesperado: %#v", result)
	}
	if sales.order == nil || production.order == nil || len(repo.notes) != 1 {
		t.Fatalf("persistências não executadas")
	}
}

func TestTechnicalAssistanceOrderGenerationExplainsMissingData(t *testing.T) {
	reason := int64(33)
	repo := &fakeTARepo{reasons: map[int64]*entity.DefectReason{reason: {Code: reason, GeneratesSalesOrder: true}}, call: &entity.Call{Code: 1, Status: entity.CallStatusPending}, items: []*entity.CallItem{{DefectReasonCode: &reason}}}
	_, err := (&UseCase{Repo: repo, Auth: taAllowAuth{}}).GenerateOrders(context.Background(), request.GenerateTechnicalAssistanceOrdersDTO{CallCode: 1})
	if err == nil {
		t.Fatal("esperava pré-condição")
	}
	for _, text := range []string{"divisão de vendas", "tabela de preço", "condição de pagamento", "almoxarifado"} {
		if !strings.Contains(err.Error(), text) {
			t.Fatalf("erro %q não contém %q", err, text)
		}
	}
}
