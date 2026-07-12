//go:build integration

package purchase_order

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPurchaseOrderConsultationFiltersCalculatesAndScopesTenant(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	var enterpriseID, enterpriseCode int64
	if err := pool.QueryRow(ctx, "SELECT id,code FROM enterprise ORDER BY id LIMIT 1").Scan(&enterpriseID, &enterpriseCode); err != nil {
		t.Fatal(err)
	}
	orderNumber := testutil.UniqueCode()
	itemA, itemB := testutil.UniqueCode(), testutil.UniqueCode()
	var orderCode, foreignOrder int64
	if err := pool.QueryRow(ctx, `INSERT INTO purchase_orders(order_number,enterprise_code,created_by,currency_code,freight_value,order_type,kanban_origin,buyer_employee_code,request_type_code) VALUES($1,$2,$3,'USD',5,'OCL',true,77,9) RETURNING code`, orderNumber, enterpriseCode, uuid.New()).Scan(&orderCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO purchase_orders(order_number,enterprise_code,created_by) VALUES($1,$2,$3) RETURNING code`, orderNumber, enterpriseCode+987654, uuid.New()).Scan(&foreignOrder); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM purchase_order_items WHERE purchase_order_code=ANY($1)", []int64{orderCode, foreignOrder})
		pool.Exec(ctx, "DELETE FROM purchase_orders WHERE code=ANY($1)", []int64{orderCode, foreignOrder})
		pool.Exec(ctx, "DELETE FROM purchase_order_currency_rates WHERE enterprise_id=$1 AND currency_code='USD'", enterpriseID)
	})
	_, err := pool.Exec(ctx, `INSERT INTO purchase_order_items(purchase_order_code,sequence,item_code,requested_qty,received_qty,cancelled_qty,unit_price,discount_pct,additions,ipi_pct,icms_pct,icms_st_pct) VALUES
		($1,1,$2,10,6,4,10,10,5,10,18,5),($1,2,$3,10,2,1,2,0,0,0,0,0),($4,1,$2,1,0,0,999,0,0,0,0,0)`, orderCode, itemA, itemB, foreignOrder)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO purchase_order_currency_rates(enterprise_id,currency_code,rate_date,rate_to_base) VALUES($1,'USD',CURRENT_DATE,5) ON CONFLICT(enterprise_id,currency_code,rate_date) DO UPDATE SET rate_to_base=EXCLUDED.rate_to_base`, enterpriseID)
	if err != nil {
		t.Fatal(err)
	}
	var attachmentID int64
	if err := pool.QueryRow(ctx, `INSERT INTO purchase_order_attachments(purchase_order_code,file_name,content_type,content) VALUES($1,'manual.pdf','application/pdf',$2) RETURNING id`, orderCode, []byte("pdf")).Scan(&attachmentID); err != nil {
		t.Fatal(err)
	}
	repo := &PurchaseOrderRepositorySQLC{db: pool}
	today := time.Now()
	rows, err := repo.Consult(ctx, purchase_order_uc.PurchaseOrderConsultationFilter{EnterpriseID: enterpriseID, OrderFrom: &orderNumber, OrderTo: &orderNumber, ItemFrom: &itemA, ItemTo: &itemA, Position: purchase_order_uc.PositionCancelled, AllItems: true, Convert: true, TargetCurrency: "BRL", BaseDate: &today, OnlyKanban: true, BuyerCode: ptr64(77), OrderType: "OCL", RequestTypeCode: ptr64(9), Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d want 1: %+v", len(rows), rows)
	}
	got := rows[0]
	if len(got.Items) != 2 || got.Items[0].Position != purchase_order_uc.PositionAttended || got.Items[1].Position != purchase_order_uc.PositionPending {
		t.Fatalf("positions/items=%+v", got.Items)
	}
	if !got.ProductsTotal.Equal(decimal.NewFromInt(600)) || !got.Freight.Equal(decimal.NewFromInt(25)) || !got.ProductsWithFreight.Equal(decimal.NewFromInt(625)) {
		t.Fatalf("totals=%s freight=%s with=%s", got.ProductsTotal, got.Freight, got.ProductsWithFreight)
	}
	if len(got.Attachments) != 1 || got.Attachments[0].ID != attachmentID {
		t.Fatalf("attachments=%+v", got.Attachments)
	}
	file, err := repo.GetAttachment(ctx, enterpriseID, orderCode, attachmentID)
	if err != nil || string(file.Content) != "pdf" {
		t.Fatalf("file=%+v err=%v", file, err)
	}
	if _, err := repo.GetAttachment(ctx, enterpriseID+999, orderCode, attachmentID); err != purchase_order_uc.ErrAttachmentNotFound {
		t.Fatalf("tenant attachment err=%v", err)
	}
}

func ptr64(v int64) *int64 { return &v }
