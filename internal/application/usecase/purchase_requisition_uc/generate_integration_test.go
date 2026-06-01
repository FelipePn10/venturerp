//go:build integration

package purchase_requisition_uc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	supplierentity "github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	itemsupplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier"
	porepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	reqrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// allowAuth embeds ports.AuthService (nil) and only overrides the permission the
// generation use case actually calls.
type allowAuth struct{ ports.AuthService }

func (allowAuth) CanCreatePurchaseOrder(context.Context) bool { return true }

func TestIntegration_GeneratePurchaseOrders_E2E(t *testing.T) {
	q, pool := testutil.Queries(t)
	ctx := context.Background()

	suppRepo := supplierrepo.New(q, pool)
	reqRepository := reqrepo.New(q, pool)
	poRepository := porepo.NewPurchaseOrderRepositorySQLC(pool)
	itemSupplierUC := item_supplier_uc.NewItemSupplierUseCase(itemsupplierrepo.New(q, pool))
	supplierUC := supplier_uc.NewSupplierUseCase(suppRepo)

	// 1) A registered supplier (PO.supplier_code FK requires it to exist).
	supplierCode := testutil.UniqueCode()
	ie := "1234567890"
	s, err := supplierentity.NewSupplier(supplierCode, supplierentity.SupplierInput{
		Name: "Fornecedor E2E", PersonType: supplierentity.PersonJuridica,
		DocumentType: supplierentity.DocumentEstrangeiro, DocumentNumber: fmt.Sprintf("EXT%d", supplierCode),
		StateRegistration: &ie,
	}, uuid.New())
	if err != nil {
		t.Fatalf("NewSupplier: %v", err)
	}
	if _, err := suppRepo.CreateSupplier(ctx, s); err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}

	const itemCode = int64(556677)

	// 2) Preferred supplier for the item.
	if _, err := itemSupplierUC.Upsert(ctx, request.UpsertItemPreferredSupplierDTO{
		ItemCode: itemCode, SupplierCode: supplierCode, Ranking: 1, CreatedBy: uuid.New(),
	}); err != nil {
		t.Fatalf("Upsert preferred supplier: %v", err)
	}

	// 3) A purchase requisition with one open item.
	reqUC := purchase_requisition_uc.NewPurchaseRequisitionUseCase(reqRepository)
	req, err := reqUC.Create(ctx, request.CreatePurchaseRequisitionDTO{
		EnterpriseCode: 1, CreatedBy: uuid.New(),
		Items: []request.RequisitionItemInput{{ItemCode: itemCode, Quantity: 10, SuggestedPrice: 5}},
	})
	if err != nil {
		t.Fatalf("create requisition: %v", err)
	}
	reqItemID := req.Items[0].ID

	// Cleanup (FK-safe order): PO items → PO → supplier/requisition/preferred.
	var generatedPOCode int64
	defer func() {
		if generatedPOCode != 0 {
			testutil.Exec(t, pool, "DELETE FROM purchase_order_items WHERE purchase_order_code = $1", generatedPOCode)
			testutil.Exec(t, pool, "DELETE FROM purchase_orders WHERE code = $1", generatedPOCode)
		}
		testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", req.Code)
		testutil.Exec(t, pool, "DELETE FROM item_preferred_suppliers WHERE item_code = $1", itemCode)
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE code = $1", supplierCode)
	}()

	// 4) Generate purchase orders from the selected requisition item.
	gen := &purchase_requisition_uc.GeneratePurchaseOrdersUseCase{
		Reqs:             reqRepository,
		POs:              poRepository,
		Auth:             allowAuth{},
		Preferred:        itemSupplierUC,
		SupplierDefaults: supplierUC,
		PriceProvider:    nil, // suggested price from the requisition is used
	}
	res, err := gen.Execute(ctx, request.GeneratePurchaseOrdersDTO{
		EnterpriseCode: 1, CreatedBy: uuid.New(),
		Selections: []request.GenerationSelection{{RequisitionItemID: reqItemID, QtyToAttend: 10}},
	})
	if err != nil {
		t.Fatalf("GeneratePurchaseOrders: %v", err)
	}
	if len(res.Orders) != 1 {
		t.Fatalf("expected 1 generated order, got %d (skipped=%v)", len(res.Orders), res.Skipped)
	}
	po := res.Orders[0]
	generatedPOCode = po.Code

	if po.SupplierCode == nil || *po.SupplierCode != supplierCode {
		t.Errorf("PO supplier = %v, want %d", po.SupplierCode, supplierCode)
	}
	if len(po.Items) != 1 || po.Items[0].ItemCode != itemCode || po.Items[0].RequestedQty != 10 {
		t.Fatalf("unexpected PO item: %+v", po.Items)
	}
	if po.Items[0].UnitPrice != 5 {
		t.Errorf("PO item price = %v, want 5 (from requisition suggested price)", po.Items[0].UnitPrice)
	}

	// 5) Requisition item must be marked fully attended.
	item, err := reqRepository.GetItem(ctx, reqItemID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.AttendedQty != 10 || item.Balance() != 0 {
		t.Errorf("requisition attendance = %v (balance %v), want 10 / 0", item.AttendedQty, item.Balance())
	}
}
