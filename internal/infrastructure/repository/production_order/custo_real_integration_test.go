//go:build integration

package production_order_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

// O custo real da OF sai pelos componentes do padrão: horas das ETAPAS × taxa de
// máquina e de mão de obra do centro, serviço de terceiro recebido × preço, e
// só a OF da empresa da sessão. Antes só contava apontamento com início/fim e
// máquina, e o terceiro ficava de fora.
func TestCustoRealDaOFPorComponente(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM production_order_costs WHERE production_order_id IN (SELECT id FROM production_orders WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM third_party_service_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM third_party_service_prices WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_order_operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM route_operations WHERE route_id IN (SELECT id FROM manufacturing_routes WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM manufacturing_routes WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM operations WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM work_center_costs WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM machine_types WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)

	var centro int64
	must(pool.QueryRow(base, `INSERT INTO machine_types(code,name,type,created_by,enterprise_id) VALUES($1,'Laser',(enum_range(NULL::machine_type_enum))[1],$2,$3) RETURNING id`, testutil.UniqueCode(), actor, a.ID).Scan(&centro))
	_, err := pool.Exec(base, `INSERT INTO work_center_costs(work_center_id,cost_per_hour,machine_cost_per_hour,labor_cost_per_hour,updated_by,enterprise_id) VALUES($1,0,100,40,$2,$3)`, centro, actor, a.ID)
	must(err)

	var of int64
	must(pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id)
		VALUES($1,$2,10,10,'COMPLETED',$3,$4) RETURNING id`, testutil.UniqueCode(), item, actor, a.ID).Scan(&of))
	// Duas etapas no laser: 1,5 h + 0,5 h = 2 h → máquina 200, mão de obra 80.
	_, err = pool.Exec(base, `INSERT INTO production_order_operations(production_order_id,sequence,operation_name,work_center_id,status,actual_hours,enterprise_id)
		VALUES($1,10,'Corte',$2,'DONE',1.5,$3),($1,20,'Rebarba',$2,'DONE',0.5,$3),($1,30,'Zincagem',NULL,'DONE',0,$3)`, of, centro, a.ID)
	must(err)

	// Terceiro: 8 peças de volta; sem pedido, vale a VTER0100 (12,00 − 10% de
	// imposto recuperável = 10,80); o roteiro diz 9,00 e perde para a tabela.
	var operacao, roteiro, etapa int64
	must(pool.QueryRow(base, `INSERT INTO operations(code,name,origin,created_by,enterprise_id,cost_per_unit) VALUES($1,'Zincagem','TERCEIROS',$2,$3,9) RETURNING id`, testutil.UniqueCode(), actor, a.ID).Scan(&operacao))
	must(pool.QueryRow(base, `INSERT INTO manufacturing_routes(code,item_code,alternative,is_standard,is_active,created_by,enterprise_id) VALUES($1,$2,1,TRUE,TRUE,$3,$4) RETURNING id`, testutil.UniqueCode(), item, actor, a.ID).Scan(&roteiro))
	must(pool.QueryRow(base, `INSERT INTO route_operations(route_id,sequence,operation_id,is_active) VALUES($1,30,$2,TRUE) RETURNING id`, roteiro, operacao).Scan(&etapa))
	fornecedor := testutil.UniqueCode() % 1_000_000
	hoje := time.Now()
	_, err = pool.Exec(base, `INSERT INTO third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,fulfilled_quantity,start_date,due_date,status,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'UN',10,8,$8::date,$8::date,'RELEASED_WITHOUT_PO',$9)`, testutil.UniqueCode()%1_000_000, a.ID, of, etapa, operacao, item, fornecedor, hoje, actor)
	must(err)
	_, err = pool.Exec(base, `INSERT INTO third_party_service_prices(enterprise_id,item_code,mask,supplier_code,operation_id,uom,reference_date,preferred,unit_price,tax_percent,is_active,created_by)
		VALUES($1,$2,'',$3,$4,'UN',$5::date - 30,TRUE,12,10,TRUE,$6)`, a.ID, item, fornecedor, operacao, hoje, actor)
	must(err)

	repo := production_order.NewProductionOrderRepositoryPGX(pool)
	real, err := repo.CustoRealDaOrdem(a.Ctx, of)
	must(err)
	quer := func(nome string, got decimal.Decimal, v string) {
		t.Helper()
		if !got.Equal(decimal.RequireFromString(v)) {
			t.Fatalf("%s = %s, quer %s", nome, got, v)
		}
	}
	quer("máquina", real.MachineCostReal, "200")
	quer("mão de obra", real.LaborCostReal, "80")
	quer("terceiros", real.SubcontractCostReal.Round(2), "86.4")

	// Liquidação gravada e lida só pela empresa dona da OF.
	liq := entity.BuildSettlement(of, entity.ActualCostInputs{ProducedQty: decimal.NewFromInt(10), MachineCostReal: real.MachineCostReal, LaborCostReal: real.LaborCostReal, SubcontractCostReal: real.SubcontractCostReal},
		entity.StandardUnitCost{Machine: decimal.NewFromInt(18), Subcontract: decimal.NewFromInt(9)}, "", actor)
	_, err = repo.SettleCost(a.Ctx, liq)
	must(err)
	lida, err := repo.GetCost(a.Ctx, of)
	must(err)
	quer("máquina gravada", lida.MachineCostReal, "200")
	quer("terceiros padrão gravado", lida.SubcontractCostStd, "90")
	if _, err := repo.GetCost(b.Ctx, of); err == nil {
		t.Fatal("a empresa B leu a apuração de custo da OF da empresa A")
	}
}
