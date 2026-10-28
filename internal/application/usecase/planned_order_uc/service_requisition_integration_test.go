//go:build integration

package planned_order_uc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	costcenterRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	purchasereqRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Verifies the subcontracting hook: firming an order whose item has an external
// operation with a service item raises a purchase requisition line for it.
func TestIntegration_FirmGeneratesServiceRequisition(t *testing.T) {
	q, pool := testutil.Queries(t)
	rRepo := routingRepo.New(q)
	reqRepo := purchasereqRepo.New(q, pool)
	uid := uuid.New()

	// Os repositórios de roteiro e requisição filtram por empresa: o contexto
	// precisa carregar o tenant como a requisição HTTP carregaria.
	var enterpriseID int64
	if err := pool.QueryRow(context.Background(), "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil || enterpriseID == 0 {
		t.Skip("integration database has no enterprise")
	}
	ctx := testutil.TenantContext(t, pool)

	serviceItem := testutil.UniqueCode()
	supplier := testutil.UniqueCode()

	// External operation with subcontracting data.
	op, _ := routingentity.NewOperation(testutil.UniqueCode(), "Zincagem", nil, routingentity.OriginExternal, nil, 0, 0, uid)
	op.ServiceItemCode = &serviceItem
	op.SupplierID = &supplier
	cost := 12.5
	op.CostPerUnit = &cost
	lead := int32(7)
	op.LeadTimeDays = &lead
	createdOp, err := rRepo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM operations WHERE id = $1", createdOp.ID)

	itemCode := testutil.UniqueCode()
	testutil.Exec(t, pool, "INSERT INTO items (code, business_code, warehouse_code, created_by, enterprise_id) VALUES ($1,$2,$3,$4,$5)",
		itemCode, fmt.Sprintf("IT-%d", itemCode), itemCode, uid, enterpriseID)
	defer testutil.Exec(t, pool, "DELETE FROM items WHERE code = $1", itemCode)

	rt, _ := routingentity.NewManufacturingRoute(testutil.UniqueCode(), itemCode, nil, 1, nil, true, nil, nil, uid)
	route, err := rRepo.CreateRoute(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	defer testutil.Exec(t, pool, "DELETE FROM manufacturing_routes WHERE id = $1", route.ID)
	ro, _ := routingentity.NewRouteOperation(route.ID, 10, createdOp.ID, nil, nil, nil, nil)
	if _, err := rRepo.AddRouteOperation(ctx, ro); err != nil {
		t.Fatalf("AddRouteOperation: %v", err)
	}

	// Centro de custo da ordem, que aprova requisições: a requisição de serviço
	// sai com o centro e espera o aprovador dele, como a requisição manual.
	aprovador := testutil.Actor(t, pool)
	var ccCode int64
	var ccID int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO cost_centers (code, description, type, start_date, created_by, enterprise_id)
		VALUES ((SELECT COALESCE(MAX(code),0)+1 FROM cost_centers), 'CENTRO DA OF', 'PRODUCTIVE', CURRENT_DATE, $1, $2) RETURNING code, id`, aprovador, enterpriseID).Scan(&ccCode, &ccID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE cost_center_gestao SET aprovador_id = $1, aprova_requisicoes = TRUE WHERE cost_center_id = $2`, aprovador, ccID)
	defer testutil.Exec(t, pool, "DELETE FROM cost_centers WHERE id = $1", ccID)
	defer testutil.Exec(t, pool, "DELETE FROM requisicao_aprovacoes_centro WHERE cost_center_id = $1", ccID)

	uc := &FirmPlannedOrderUseCase{ReqRepo: reqRepo, ExternalOps: rRepo, Auth: &serviceReqAuth{enterprise: enterpriseID, actor: uid},
		AprovacoesDoCentro: costcenterRepo.NewGestaoPG(pool)}
	order := &plannedentity.PlannedOrder{ItemCode: itemCode, Quantity: 10, CreatedBy: uid, CostCenterCode: &ccCode}

	reqCode, err := uc.generateServiceRequisition(ctx, order)
	if err != nil {
		t.Fatalf("generateServiceRequisition: %v", err)
	}
	if reqCode == 0 {
		t.Fatal("expected a requisition to be created")
	}
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisition_items WHERE requisition_code = $1", reqCode)
	defer testutil.Exec(t, pool, "DELETE FROM purchase_requisitions WHERE code = $1", reqCode)

	items, err := reqRepo.ListItems(ctx, reqCode)
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("requisition items = %d, want 1", len(items))
	}
	it := items[0]
	if it.ItemCode != serviceItem || it.Quantity != 10 || it.SuggestedPrice != 12.5 {
		t.Errorf("item = %+v, want service %d qty 10 price 12.5", it, serviceItem)
	}
	if it.CostCenterCode == nil || *it.CostCenterCode != ccCode {
		t.Errorf("a requisição de serviço leva o centro da ordem: %v", it.CostCenterCode)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM requisicao_aprovacoes_centro WHERE requisition_code = $1 AND cost_center_id = $2`, reqCode, ccID).Scan(&status); err != nil || status != "PENDENTE" {
		t.Errorf("aprovação do centro aberta: %q %v", status, err)
	}
}

// serviceReqAuth supplies the tenant and actor that the use case now reads from
// the JWT instead of a struct field.
type serviceReqAuth struct {
	ports.AuthService
	enterprise int64
	actor      uuid.UUID
}

func (a *serviceReqAuth) EnterpriseCode(context.Context) (int64, error) { return a.enterprise, nil }
func (a *serviceReqAuth) UserID(context.Context) (uuid.UUID, error)     { return a.actor, nil }
