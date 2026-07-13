//go:build integration

package routing_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

func TestThirdPartyRoutingDetailsRoundTrip(t *testing.T) {
	_, pool := testutil.Queries(t)
	ctx := context.Background()
	actor := uuid.New()
	itemCode, operationCode, routeCode := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	if _, err := pool.Exec(ctx, `INSERT INTO items(code,warehouse_code,created_by) VALUES($1,$1,$2)`, itemCode, actor); err != nil {
		t.Fatal(err)
	}
	var operationID, routeID int64
	if err := pool.QueryRow(ctx, `INSERT INTO operations(code,name,origin,supplier_id,service_item_code,cost_per_unit,lead_time_days,third_party_remittance,created_by)
		VALUES($1,'External details','TERCEIROS',77,$2,12.5,5,'GENERIC',$3) RETURNING id`, operationCode, itemCode, actor).Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,created_by) VALUES($1,$2,1,TRUE,$3) RETURNING id`, routeCode, itemCode, actor).Scan(&routeID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM manufacturing_routes WHERE id=$1`, routeID)
		_, _ = pool.Exec(ctx, `DELETE FROM operations WHERE id=$1`, operationID)
		_, _ = pool.Exec(ctx, `DELETE FROM items WHERE code=$1`, itemCode)
	})
	remittance := "ORDER_ITEM"
	supplier, serviceItem, cost, lead := int64(88), itemCode, 25.75, int32(3)
	repo := routingrepo.New(sqlc.New(pool))
	created, err := repo.AddRouteOperation(ctx, &entity.RouteOperation{RouteID: routeID, Sequence: 10, OperationID: operationID, Situation: entity.RouteOpApproved, SupplierID: &supplier, ServiceItemCode: &serviceItem, CostPerUnit: &cost, LeadTimeDays: &lead, ThirdPartyRemittance: &remittance})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetRouteOperations(ctx, routeID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("route operations=%+v err=%v", rows, err)
	}
	got := rows[0]
	if got.ID != created.ID || got.SupplierID == nil || *got.SupplierID != supplier || got.ServiceItemCode == nil || *got.ServiceItemCode != serviceItem || got.CostPerUnit == nil || *got.CostPerUnit != cost || got.LeadTimeDays == nil || *got.LeadTimeDays != lead || got.ThirdPartyRemittance == nil || *got.ThirdPartyRemittance != remittance {
		t.Fatalf("third-party details were not preserved: %+v", got)
	}
}
