//go:build integration

package customer_material_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

func somaSuprimento(t *testing.T, a *ambiente, item int64) (decimal.Decimal, map[ports.SupplySourceType]decimal.Decimal) {
	t.Helper()
	m, err := planned_order.NewPlannedOrderSupplyAdapter(sqlc.New(a.pool)).ListFirmSupplyForItems(a.ctx, []int64{item})
	if err != nil {
		t.Fatalf("lendo suprimento: %v", err)
	}
	total, porOrigem := decimal.Zero, map[ports.SupplySourceType]decimal.Decimal{}
	for _, s := range m[item] {
		total = total.Add(s.Quantity)
		porOrigem[s.SourceType] = porOrigem[s.SourceType].Add(s.Quantity)
	}
	return total, porOrigem
}

// TestOFManualEPedidoSemPlanejadaSaoSuprimento: o MRP só via estoque e ordem
// planejada firme. A OF manual e o pedido lançado pelo comprador não contavam,
// e o MRP sugeria de novo o que já estava sendo feito ou comprado.
func TestOFManualEPedidoSemPlanejadaSaoSuprimento(t *testing.T) {
	a := montar(t)
	item := a.item(t, "PECA")

	of, _ := a.ordemDeFabricacao(t, item, nil, "IN_PROGRESS", "4")
	total, origem := somaSuprimento(t, a, item)
	if !total.Equal(decimal.NewFromInt(6)) || !origem[ports.SupplySourceProductionOrder].Equal(decimal.NewFromInt(6)) {
		t.Fatalf("OF manual de 10 com 4 entregues = 6 a chegar; veio %s %v", total, origem)
	}

	// OF de ordem planejada firme: entra uma vez só, pela planejada.
	planejada, _ := a.ordemFirme(t, item, "10")
	a.ordemDeFabricacao(t, item, &planejada, "OPEN", "0")
	if total, _ = somaSuprimento(t, a, item); !total.Equal(decimal.NewFromInt(16)) {
		t.Fatalf("OF da planejada firme contou duas vezes: %s (esperado 6 + 10)", total)
	}
	testutil.Exec(t, a.pool, `UPDATE production_orders SET status = 'COMPLETED' WHERE id = $1`, of)
	if total, _ = somaSuprimento(t, a, item); !total.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("OF concluída ainda é suprimento: %s", total)
	}

	// Pedido do comprador, sem ordem planejada, em caixa de 12.
	comprado := a.item(t, "COMPRADO")
	fornecedor := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, a.pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3::uuid,$4)`,
		fornecedor, "12345678000195", a.userID, a.enterpriseID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM suppliers WHERE code = $1 AND enterprise_id = $2`, fornecedor, a.enterpriseID)
	})
	var pedido int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by) VALUES($1,$2,$3,$4::uuid) RETURNING code`,
		testutil.UniqueCode(), a.codigoDaEmpresa(t), fornecedor, a.userID).Scan(&pedido); err != nil {
		t.Fatalf("criando o pedido: %v", err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM purchase_order_items WHERE purchase_order_code = $1`, pedido)
		testutil.Exec(t, a.pool, `DELETE FROM purchase_orders WHERE code = $1`, pedido)
	})
	testutil.Exec(t, a.pool, `INSERT INTO purchase_order_items(purchase_order_code,item_code,purchase_uom,unit_price,requested_qty,received_qty,internal_qty,internal_uom)
		VALUES($1,$2,'CX',1,3,1,36,'UN')`, pedido, comprado)
	total, origem = somaSuprimento(t, a, comprado)
	if !total.Equal(decimal.NewFromInt(24)) || !origem[ports.SupplySourcePurchaseOrder].Equal(decimal.NewFromInt(24)) {
		t.Fatalf("3 CX (36 UN) com 1 CX recebida = 24 UN a chegar; veio %s %v", total, origem)
	}
	testutil.Exec(t, a.pool, `UPDATE purchase_orders SET status = 'CANCELLED' WHERE code = $1`, pedido)
	if total, _ = somaSuprimento(t, a, comprado); !total.IsZero() {
		t.Fatalf("pedido cancelado ainda é suprimento: %s", total)
	}

	if b := montar(t); func() bool { s, _ := somaSuprimento(t, b, item); return !s.IsZero() }() {
		t.Fatal("suprimento vazou para outra empresa")
	}
}

// TestEmpenhoDaOFBaixaComOConsumo: o material da OF que ainda não saiu é
// demanda do componente; o consumo lançado atualiza o atendido (gatilho) e o
// empenho baixa. Linha de retorno não é demanda.
func TestEmpenhoDaOFBaixaComOConsumo(t *testing.T) {
	a := montar(t)
	pai, comp := a.item(t, "PAI"), a.item(t, "COMPONENTE")
	of, _ := a.ordemDeFabricacao(t, pai, nil, "OPEN", "0")
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM production_consumptions WHERE production_order_id = $1`, of)
	})
	testutil.Exec(t, a.pool, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,quantity,warehouse_id,created_by,route_operation_sequence)
		VALUES($1,$2,'DEMAND',$3,3,1,$4::uuid,NULL),($1,$2,'DEMAND',$3,7,1,$4::uuid,20),($1,$2,'RETURN',$3,5,1,$4::uuid,NULL)`,
		of, a.enterpriseID, comp, a.userID)

	adapter := planned_order.NewPlannedOrderSupplyAdapter(sqlc.New(a.pool))
	empenho := func() decimal.Decimal {
		t.Helper()
		lista, err := adapter.ListOpenOrderAllocations(a.ctx)
		if err != nil {
			t.Fatal(err)
		}
		total := decimal.Zero
		for _, e := range lista {
			if e.ProductionOrderID == of && e.ItemCode == comp {
				total = total.Add(e.Quantity)
			}
		}
		return total
	}
	atendido := func() string {
		t.Helper()
		var s string
		if err := a.pool.QueryRow(a.ctx, `SELECT string_agg(attended_quantity::numeric(10,2)::text, '/' ORDER BY route_operation_sequence NULLS FIRST, id)
			FROM production_order_materials WHERE production_order_id = $1 AND material_kind = 'DEMAND'`, of).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if got := empenho(); !got.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("empenho = %s, esperado 10 (3 + 7; a linha de retorno não conta)", got)
	}

	var consumo int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO production_consumptions(production_order_id,item_code,consumed_qty,consumption_date,created_by)
		VALUES($1,$2,5,CURRENT_DATE,$3::uuid) RETURNING id`, of, comp, a.userID).Scan(&consumo); err != nil {
		t.Fatal(err)
	}
	if got := atendido(); got != "3.00/2.00" {
		t.Fatalf("5 consumidos: o início da ordem enche primeiro (3) e a etapa 20 leva 2; veio %s", got)
	}
	if got := empenho(); !got.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("empenho depois de consumir 5 = %s, esperado 5", got)
	}
	testutil.Exec(t, a.pool, `INSERT INTO production_consumptions(production_order_id,item_code,consumed_qty,consumption_date,created_by)
		VALUES($1,$2,7,CURRENT_DATE,$3::uuid)`, of, comp, a.userID)
	if got := atendido(); got != "3.00/9.00" {
		t.Fatalf("consumo além da necessidade fica na última linha; veio %s", got)
	}
	if got := empenho(); !got.IsZero() {
		t.Fatalf("tudo consumido e ainda há empenho: %s", got)
	}
	// Estorno do consumo devolve o empenho.
	testutil.Exec(t, a.pool, `DELETE FROM production_consumptions WHERE id = $1`, consumo)
	if got := atendido(); got != "3.00/4.00" {
		t.Fatalf("estorno de 5: atendido %s, esperado 3/4", got)
	}
	testutil.Exec(t, a.pool, `UPDATE production_orders SET status = 'COMPLETED' WHERE id = $1`, of)
	if got := empenho(); !got.IsZero() {
		t.Fatalf("OF concluída ainda empenha material: %s", got)
	}
}
