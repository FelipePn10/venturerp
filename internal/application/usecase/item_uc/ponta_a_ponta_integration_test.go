//go:build integration

package item_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_governanca_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	fiscaldom "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	shipentity "github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	independentDemand "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	industrialCalendar "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	govpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/itemgovernanca"
	mrpCalculation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	planned "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	productionPlan "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan"
	porepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	restrictionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	salesForecastRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	shiprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Os processos de verdade, cada um pelo caso de uso que a API usa: MRP completo,
// linha de pedido de venda, recebimento de compra e despacho do romaneio.
func TestPontaAPonta(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode() % 700_000
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		// A chave de empresa ficou nas linhas dos documentos: elas saem primeiro,
		// senão a empresa fica presa e o fornecedor sobra para o próximo teste.
		for _, q := range []string{`DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code=$1)`,
			`DELETE FROM purchase_orders WHERE enterprise_code=$1`, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`,
			`DELETE FROM sales_orders WHERE enterprise_code=$1`} {
			if _, err := pool.Exec(context.Background(), q, a.Code); err != nil {
				t.Logf("limpeza %s: %v", q, err)
			}
		}
		for _, q := range []string{`DELETE FROM shipments WHERE enterprise_id=$1`, `DELETE FROM suppliers WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`} {
			if _, err := pool.Exec(context.Background(), q, a.ID); err != nil {
				t.Logf("limpeza %s: %v", q, err)
			}
		}
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(bg, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	q := sqlc.New(pool)
	itens := itemrepo.NewRepositoryItemSQLC(q).ComPool(pool)
	item := func(n int64, nome string, tipo int, set string) (int64, string) {
		t.Helper()
		code, cod := 80_000_000+u+n*1_000_000, fmt.Sprintf("E2E%d-%d", n, u)
		exec(`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_type,engineering_weight) VALUES($1,$2,$3,$4,$5,$6,$7,'{"gross":1,"net":1,"unit":"KG"}'::jsonb)`, code, cod, almox, actor, a.ID, nome, tipo)
		if set != "" {
			exec(`UPDATE items SET `+set+` WHERE code=$1`, code)
		}
		return code, cod
	}

	t.Run("MRP completo", func(t *testing.T) {
		normal, _ := item(1, "COMPRADO NORMAL", 1, "planning_purchase_lead_time_days=4")
		desligado, _ := item(2, "DESLIGADO NO MRP", 1, "planning_active=FALSE")
		prototipo, _ := item(3, "PROTOTIPO", 1, "is_prototype=TRUE")
		sucata, _ := item(4, "SUCATA", 1, "is_scrap=TRUE")
		processo, _ := item(5, "PROCESSO", 1, "is_process_item=TRUE")
		plano := 70_000_000 + u
		exec(`INSERT INTO production_plans(code,name,created_by,enterprise_id,independent_demands) VALUES($1,'E2E',$2,$3,TRUE)`, plano, actor, a.ID)
		for i, c := range []int64{normal, desligado, prototipo, sucata, processo} {
			exec(`INSERT INTO independent_demands(code,item_code,quantity,demand_date,created_by,enterprise_id) VALUES($1,$2,10,$3::date,$4,$5)`,
				plano+int64(i)+1, c, time.Now().AddDate(0, 0, 30), actor, a.ID)
		}
		mrpRepo := mrpCalculation.NewMRPCalculationRepositorySQLC(q, pool)
		svc := mrpservice.NewMRPService(mrpRepo, structure.NewItemStructureRepository(q).WithHistory(pool), independentDemand.NewIndependentDemandRepositorySQLC(q).ComDB(pool),
			industrialCalendar.NewIndustrialCalendarRepositorySQLC(q), itens, planned.NewPlannedOrderSupplyAdapter(q), productionPlan.NewProductionPlanRepositorySQLC(q),
			salesForecastRepo.NewSalesForecastRepositorySQLC(q, pool), restrictionRepo.NewRestrictionRepositorySQLC(q), routingRepo.New(q), structure_query.NewStructureQueryRepository(q))
		log, err := svc.Calculate(a.Ctx, plano, 1, false)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("MRP: %+v", log)
		rows, err := pool.Query(bg, `SELECT DISTINCT item_code FROM mrp_planned_suggestions WHERE plan_code=$1`, plano)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		gerados := map[int64]bool{}
		for rows.Next() {
			var c int64
			if err := rows.Scan(&c); err != nil {
				t.Fatal(err)
			}
			gerados[c] = true
		}
		if !gerados[normal] {
			t.Fatalf("o item normal gera sugestão: %v, log %+v", gerados, log)
		}
		for nome, c := range map[string]int64{"desligado": desligado, "protótipo": prototipo, "sucata": sucata, "processo": processo} {
			if gerados[c] {
				t.Fatalf("%s não pode gerar ordem", nome)
			}
		}
		var inicio, necessidade time.Time
		if err := pool.QueryRow(bg, `SELECT start_date, need_date FROM mrp_planned_suggestions WHERE plan_code=$1 AND item_code=$2`, plano, normal).Scan(&inicio, &necessidade); err != nil {
			t.Fatal(err)
		}
		if !inicio.Before(necessidade) {
			t.Fatalf("o prazo de compra do item antecipa o início: %s → %s", inicio, necessidade)
		}
	})

	cliente := 880_000 + u%90_000
	exec(`INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE E2E','11444777000161',$2,$3)`, cliente, actor, a.ID)
	tabela := 60_000_000 + u
	var tabelaID int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_tables(code,description,enterprise_id) VALUES($1,'E2E',$2) RETURNING id`, tabela, a.ID).Scan(&tabelaID); err != nil {
		t.Fatal(err)
	}
	pedidoVenda := func() int64 {
		t.Helper()
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,customer_code,created_by,price_table_code) VALUES($1,$2,$3,$4,$5) RETURNING code`,
			testutil.UniqueCode()%900_000, a.Code, cliente, actor, tabela).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}

	t.Run("Linha de pedido de venda", func(t *testing.T) {
		_, parafuso := item(10, "PARAFUSO CAIXA 100", 1, "commercial_sale_multiple=100, commercial_minimum_sale_quantity=100, accepts_fractional_quantity=FALSE, commercial_estimated_delivery_days=15")
		_, peca := item(11, "PECA DE OUTRO CLIENTE", 0, fmt.Sprintf("commercial_exclusive_customer_code=%d", cliente+1))
		exec(`INSERT INTO sales_table_prices(sales_table_id,item_code,price) SELECT $1, code::text, 10 FROM items WHERE enterprise_id=$2`, tabelaID, a.ID)
		uc := &sales_order_uc.CreateSalesOrderItemUseCase{Repo: salesorder.NewSalesOrderRepositorySQLC(q), Auth: &auth.AuthService{}, Items: itens,
			Prices: customerrepo.New(q, pool)}
		dto := func(cod, qtd string) request.CreateSalesOrderItemDTO {
			return request.CreateSalesOrderItemDTO{SalesOrderCode: pedidoVenda(), Sequence: 1, ItemCode: request.TextCode(cod), RequestedQty: decimal.RequireFromString(qtd), UnitPrice: decimal.NewFromInt(10), DigitDate: time.Now().Format("2006-01-02")}
		}
		if _, err := uc.Execute(a.Ctx, dto(parafuso, "150")); err == nil || !strings.Contains(err.Error(), "múltiplos") {
			t.Fatalf("150 fora do múltiplo: %v", err)
		}
		if _, err := uc.Execute(a.Ctx, dto(peca, "1")); err == nil || !strings.Contains(err.Error(), "exclusivo") {
			t.Fatalf("exclusivo de outro cliente: %v", err)
		}
		criado, err := uc.Execute(a.Ctx, dto(parafuso, "200"))
		if err != nil {
			t.Fatal(err)
		}
		var entrega *time.Time
		if err := pool.QueryRow(bg, `SELECT delivery_date FROM sales_order_items WHERE code=$1`, criado.Code).Scan(&entrega); err != nil {
			t.Fatal(err)
		}
		if entrega == nil || entrega.Before(time.Now().AddDate(0, 0, 13)) {
			t.Fatalf("sem data, a linha assume o prazo estimado: %v", entrega)
		}

		// A data do pedido vence o prazo do cadastro: o MRP lê a da linha primeiro.
		comData := dto(parafuso, "200")
		exec(`UPDATE sales_orders SET delivery_date = CURRENT_DATE + 60 WHERE code=$1`, comData.SalesOrderCode)
		if criado, err = uc.Execute(a.Ctx, comData); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `SELECT delivery_date FROM sales_order_items WHERE code=$1`, criado.Code).Scan(&entrega); err != nil {
			t.Fatal(err)
		}
		if entrega != nil {
			t.Fatalf("a linha sem data segue a do pedido: %v", entrega)
		}
	})

	fornecedor := 870_000 + u%90_000
	exec(`INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORN E2E','11444777000161',$2,$3)`, fornecedor, actor, a.ID)
	compra := func(itemCode int64, qtd int) (int64, int64) {
		t.Helper()
		var pedido, linha int64
		if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by,status) VALUES($1,$2,$3,$4,'APPROVED') RETURNING code`,
			testutil.UniqueCode()%900_000, a.Code, fornecedor, actor).Scan(&pedido); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `INSERT INTO purchase_order_items(purchase_order_code,sequence,item_code,requested_qty,unit_price,total_price,internal_qty,internal_price) VALUES($1,1,$2,$3::numeric,10,$3::numeric*10,$3::numeric,10) RETURNING code`,
			pedido, itemCode, qtd).Scan(&linha); err != nil {
			t.Fatal(err)
		}
		return pedido, linha
	}

	t.Run("Recebimento de compra", func(t *testing.T) {
		po := porepo.NewPurchaseOrderRepositorySQLC(pool)
		estoque := stockrepo.NewStockRepositorySQLC(pool)
		uc := &purchase_order_uc.ReceivePurchaseOrderUseCase{Repo: po, StockRepo: estoque, Auth: &auth.AuthService{}, Regras: po,
			Transacao: func(ctx context.Context, fn func(purchase_order_uc.RecebimentoRepos) error) error {
				tx, err := pool.Begin(ctx)
				if err != nil {
					return err
				}
				defer tx.Rollback(ctx)
				if err := fn(purchase_order_uc.RecebimentoRepos{Pedidos: porepo.NewPurchaseOrderRepositoryTx(tx), Estoque: stockrepo.NewStockRepositoryTx(tx)}); err != nil {
					return err
				}
				return tx.Commit(ctx)
			}}
		chapa, _ := item(20, "CHAPA CORRIDA", 1, "supplies_requires_certificate=TRUE, supplies_receiving_checklist=TRUE, supplies_under_delivery_tolerance_pct=5, supplies_over_delivery_tolerance_pct=10, warehouse_secondary_uom='KG', warehouse_secondary_factor=25")
		pedido, linha := compra(chapa, 100)
		recebe := func(q string, lote, corrida, cert *string, checklist bool, peso *decimal.Decimal) error {
			_, err := uc.Execute(a.Ctx, request.ReceivePurchaseOrderDTO{PurchaseOrderCode: pedido, Items: []request.ReceivePurchaseOrderItemDTO{{
				PurchaseOrderItemCode: linha, Quantity: decimal.RequireFromString(q), WarehouseID: almox, Lot: lote, HeatNumber: corrida, Certificate: cert, ChecklistConfirmed: checklist, SecondaryQuantity: peso}}})
			return err
		}
		texto := func(s string) *string { return &s }
		if err := recebe("96", texto("L1"), nil, nil, true, nil); err == nil || !strings.Contains(err.Error(), "corrida") {
			t.Fatalf("sem corrida: %v", err)
		}
		if err := recebe("96", texto("L1"), texto("C1"), texto("X1"), false, nil); err == nil || !strings.Contains(err.Error(), "check list") {
			t.Fatalf("sem check list: %v", err)
		}
		if err := recebe("111", texto("L1"), texto("C1"), texto("X1"), true, nil); err == nil || !strings.Contains(err.Error(), "tolerância") {
			t.Fatalf("acima da tolerância: %v", err)
		}
		peso := decimal.NewFromInt(2432)
		if err := recebe("96", texto("L1"), texto("C1"), texto("X1"), true, &peso); err != nil {
			t.Fatal(err)
		}
		var status, cabecalho, corrida string
		var cancelado, secundario decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT i.status, i.cancelled_qty, p.status FROM purchase_order_items i JOIN purchase_orders p ON p.code=i.purchase_order_code WHERE i.code=$1`, linha).Scan(&status, &cancelado, &cabecalho); err != nil {
			t.Fatal(err)
		}
		if status != "RECEIVED" || !cancelado.Equal(decimal.NewFromInt(4)) || cabecalho != "RECEIVED" {
			t.Fatalf("faltar 4 de 100 com tolerância de 5%% encerra: linha %s, cancelado %s, pedido %s", status, cancelado, cabecalho)
		}
		if err := pool.QueryRow(bg, `SELECT heat_number FROM stock_lots WHERE enterprise_id=$1 AND item_code=$2 AND lot='L1'`, a.ID, chapa).Scan(&corrida); err != nil || corrida != "C1" {
			t.Fatalf("corrida: %q %v", corrida, err)
		}
		if err := pool.QueryRow(bg, `SELECT secondary_quantity FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2`, a.ID, chapa).Scan(&secundario); err != nil || !secundario.Equal(peso) {
			t.Fatalf("o peso real vence o nominal: %s %v", secundario, err)
		}
	})

	t.Run("Despacho do romaneio", func(t *testing.T) {
		estoque := stockrepo.NewStockRepositorySQLC(pool)
		caixa, _ := item(30, "CAIXA PAPELAO", 1, "")
		guarda, _ := item(31, "GUARDA ROUPA", 0, fmt.Sprintf("commercial_volumes_per_unit=3, commercial_assemble_shipping_volumes=TRUE, commercial_packaging_item_code=%d", caixa))
		exec(`INSERT INTO stock_balances(item_code,mask,warehouse_id,quantity,enterprise_id) VALUES($1,'',$3,100,$2),($4,'',$3,10,$2)`, caixa, a.ID, almox, guarda)
		repo := shiprepo.NewShipmentRepositoryPG(pool)
		uc := &shipment_uc.ShipmentUseCase{Repo: repo, Stock: estoque, Regras: shiprepo.NewRegrasDoRomaneioPG(pool),
			Transacao: func(ctx context.Context, fn func(shipment_uc.RomaneioRepos) error) error {
				tx, err := pool.Begin(ctx)
				if err != nil {
					return err
				}
				defer tx.Rollback(ctx)
				if err := fn(shipment_uc.RomaneioRepos{Romaneios: shiprepo.NewShipmentRepositoryTx(tx), Estoque: stockrepo.NewStockRepositoryTx(tx)}); err != nil {
					return err
				}
				return tx.Commit(ctx)
			}}
		tipo := shipentity.ShipmentRefSalesOrder
		so := pedidoVenda()
		r, err := uc.Create(a.Ctx, shipment_uc.CreateShipmentInput{ReferenceType: &tipo, SalesOrderCode: &so, TotalVolumes: 6, CreatedBy: actor})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = uc.AddItem(a.Ctx, shipment_uc.AddShipmentItemInput{ShipmentCode: r.Code, Sequence: 1, ItemCode: guarda, WarehouseID: &almox, Quantity: decimal.NewFromInt(2), UnitNetWeight: decimal.NewFromInt(1), UnitGrossWeight: decimal.NewFromInt(1)}); err != nil {
			t.Fatal(err)
		}
		if err = uc.Separate(a.Ctx, r.Code, actor); err != nil {
			t.Fatal(err)
		}
		full, err := uc.Get(a.Ctx, r.Code)
		if err != nil {
			t.Fatal(err)
		}
		if err = uc.ConferItem(a.Ctx, r.Code, full.Items[0].ID, decimal.NewFromInt(2)); err != nil {
			t.Fatal(err)
		}
		if err = uc.Confer(a.Ctx, r.Code, actor); err != nil {
			t.Fatal(err)
		}
		if err = uc.Ship(a.Ctx, r.Code, actor, false); err == nil || !strings.Contains(err.Error(), "volumes") {
			t.Fatalf("sem volumes não despacha: %v", err)
		}
		if _, err = uc.AddVolume(a.Ctx, shipment_uc.AddVolumeInput{ShipmentCode: r.Code, VolumeNumber: 1, GrossWeight: decimal.NewFromInt(1), NetWeight: decimal.NewFromInt(1)}); err != nil {
			t.Fatal(err)
		}
		if err = uc.Ship(a.Ctx, r.Code, actor, false); err != nil {
			t.Fatal(err)
		}
		var saldo decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, caixa).Scan(&saldo); err != nil {
			t.Fatal(err)
		}
		if !saldo.Equal(decimal.NewFromInt(94)) { // 2 unidades × 3 volumes
			t.Fatalf("o despacho baixa uma caixa por volume: saldo %s", saldo)
		}
	})

	t.Run("GTIN por embalagem, revisão e duas unidades", func(t *testing.T) {
		gov := &item_governanca_uc.UseCase{Repo: govpg.New(pool), Auth: &auth.AuthService{}, Items: itens}
		codigo, cod := item(40, "CHAPA DUAS UNIDADES", 1, "warehouse_secondary_uom='KG', warehouse_secondary_factor=25, accounting_gtin='7891000100103'")
		_, outro := item(41, "OUTRO", 1, "")
		caixa := []dom.Embalagem{{GTIN: "17891000100100", Unidade: "CX", Quantidade: decimal.NewFromInt(12)}}
		if _, err := gov.SalvarEmbalagens(a.Ctx, cod, caixa); err != nil {
			t.Fatal(err)
		}
		if _, err := gov.SalvarEmbalagens(a.Ctx, outro, caixa); err == nil {
			t.Fatal("o mesmo GTIN não pode estar em dois itens")
		}
		if _, err := gov.SalvarEmbalagens(a.Ctx, outro, []dom.Embalagem{{GTIN: "7891000100103", Unidade: "UN", Quantidade: decimal.NewFromInt(1)}}); err == nil {
			t.Fatal("GTIN principal de outro item")
		}
		// O leitor e a entrada de nota acham o item pelo GTIN da caixa.
		estoque := stockrepo.NewStockRepositorySQLC(pool)
		lido, err := estoque.ResolverLeitura(a.Ctx, "17891000100100", nil)
		if err != nil || lido == nil || lido.ItemCode == nil || *lido.ItemCode != codigo {
			t.Fatalf("leitura: %+v %v", lido, err)
		}
		fiscal := fiscalrepo.NewFiscalRepositoryPG(pool)
		achado, err := fiscal.(interface {
			ResolveByEAN(context.Context, *int64, string) (*fiscaldom.ItemMatch, error)
		}).ResolveByEAN(a.Ctx, nil, "17891000100100")
		if err != nil || achado == nil || achado.ItemCode != codigo {
			t.Fatalf("entrada de nota: %+v %v", achado, err)
		}
		ident, err := fiscal.(interface {
			IdentificacaoFiscal(context.Context, []int64) (map[int64]fiscaldom.IdentificacaoFiscal, error)
		}).IdentificacaoFiscal(a.Ctx, []int64{codigo})
		if err != nil || ident[codigo].GTINPorUnidade["CX"] != "17891000100100" {
			t.Fatalf("nota: %+v %v", ident, err)
		}

		// Revisão vigente hoje atualiza o item; a futura não; a OF guarda a dela.
		// A data vem do banco: é ele que decide a vigência, e o processo pode estar
		// em outro fuso (depois das 21h de São Paulo, em UTC já é o dia seguinte).
		var hoje, amanha string
		if err := pool.QueryRow(bg, `SELECT CURRENT_DATE::text, (CURRENT_DATE + 10)::text`).Scan(&hoje, &amanha); err != nil {
			t.Fatal(err)
		}
		if _, err := gov.CriarRevisao(a.Ctx, cod, item_governanca_uc.NovaRevisaoDTO{Revision: "a", ValidFrom: hoje, Reason: "Liberação"}); err != nil {
			t.Fatal(err)
		}
		lista, err := gov.CriarRevisao(a.Ctx, cod, item_governanca_uc.NovaRevisaoDTO{Revision: "B", ValidFrom: amanha, Reason: "ECN 12"})
		if err != nil {
			t.Fatal(err)
		}
		if len(lista) != 2 || lista[1].Ate == nil {
			t.Fatalf("a anterior fecha na véspera: %+v", lista)
		}
		var vigente string
		if err := pool.QueryRow(bg, `SELECT revision FROM items WHERE code=$1`, codigo).Scan(&vigente); err != nil || vigente != "A" {
			t.Fatalf("a revisão futura não vira a vigente: %q %v", vigente, err)
		}
		var revisaoDaOF string
		if err := pool.QueryRow(bg, `INSERT INTO production_orders(order_number,item_code,planned_qty,enterprise_id,created_by) VALUES($1,$2,1,$3,$4) RETURNING item_revision`,
			testutil.UniqueCode()%900_000, codigo, a.ID, actor).Scan(&revisaoDaOF); err != nil || revisaoDaOF != "A" {
			t.Fatalf("a OF guarda a revisão: %q %v", revisaoDaOF, err)
		}

		// Entrada com peso real, saída pelo fator e estorno exato.
		exec(`INSERT INTO stock_balances(item_code,mask,warehouse_id,quantity,enterprise_id) VALUES($1,'',$2,0,$3)`, codigo, almox, a.ID)
		peso := decimal.NewFromInt(240)
		entrada := &stockentity.StockMovement{ItemCode: codigo, WarehouseID: almox, MovementType: stockentity.MovementTypeIn, Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(1), SecondaryQuantity: &peso, CreatedBy: actor}
		if _, err := estoque.CreateMovement(a.Ctx, entrada); err != nil {
			t.Fatal(err)
		}
		saida := &stockentity.StockMovement{ItemCode: codigo, WarehouseID: almox, MovementType: stockentity.MovementTypeOut, Quantity: decimal.NewFromInt(2), CreatedBy: actor}
		if _, err := estoque.CreateMovement(a.Ctx, saida); err != nil {
			t.Fatal(err)
		}
		s, err := gov.SaldoDuasUnidades(a.Ctx, cod)
		if err != nil {
			t.Fatal(err)
		}
		if d, _ := decimal.NewFromString(s.SegundoSaldo); !d.Equal(decimal.NewFromInt(190)) || *s.SegundaUnidade != "KG" {
			t.Fatalf("240 kg menos 2 × 25 = 190: %+v", s)
		}

		// Ajuste e estorno do ajuste: o segundo saldo volta ao que era.
		var ajuste int64
		if err := pool.QueryRow(bg, `INSERT INTO stock_movements(item_code,mask,warehouse_id,movement_type,quantity,created_by,enterprise_id)
			VALUES($1,'',$2,'ADJUSTMENT',4,$3,$4) RETURNING id`, codigo, almox, actor, a.ID).Scan(&ajuste); err != nil {
			t.Fatal(err)
		}
		exec(`INSERT INTO stock_movements(item_code,mask,warehouse_id,movement_type,quantity,created_by,enterprise_id,reverses_movement_id)
			VALUES($1,'',$2,'ADJUSTMENT',-4,$3,$4,$5)`, codigo, almox, actor, a.ID, ajuste)
		if s, err = gov.SaldoDuasUnidades(a.Ctx, cod); err != nil {
			t.Fatal(err)
		}
		if d, _ := decimal.NewFromString(s.SegundoSaldo); !d.Equal(decimal.NewFromInt(190)) {
			t.Fatalf("o estorno do ajuste anula o ajuste: %+v", s)
		}

		// A revisão futura chega à data: a OF já nasce nela e o job leva ao cadastro.
		exec(`UPDATE item_revisoes SET valid_from = CURRENT_DATE - 5, valid_until = CURRENT_DATE - 1 WHERE item_code=$1 AND revision='A'`, codigo)
		exec(`UPDATE item_revisoes SET valid_from = CURRENT_DATE WHERE item_code=$1 AND revision='B'`, codigo)
		if err := pool.QueryRow(bg, `INSERT INTO production_orders(order_number,item_code,planned_qty,enterprise_id,created_by) VALUES($1,$2,1,$3,$4) RETURNING item_revision`,
			testutil.UniqueCode()%900_000, codigo, a.ID, actor).Scan(&revisaoDaOF); err != nil || revisaoDaOF != "B" {
			t.Fatalf("a OF nasce na revisão vigente: %q %v", revisaoDaOF, err)
		}
		if n, err := govpg.New(pool).PromoverRevisoes(bg); err != nil || n < 1 {
			t.Fatalf("promoção: %d %v", n, err)
		}
		if err := pool.QueryRow(bg, `SELECT revision FROM items WHERE code=$1`, codigo).Scan(&vigente); err != nil || vigente != "B" {
			t.Fatalf("o cadastro acompanha a vigência: %q %v", vigente, err)
		}

		// Hierarquia: palete sem código de barras, feito de caixas. A nota lê o nível sem GTIN.
		cx, un := "CX", "UN"
		dez, quarenta := decimal.NewFromInt(10), decimal.NewFromInt(40)
		niveis, err := gov.SalvarEmbalagens(a.Ctx, cod, []dom.Embalagem{
			{GTIN: "17891000100100", Unidade: "CX", Contem: &un, ContemQuantidade: &dez},
			{Unidade: "PL", Contem: &cx, ContemQuantidade: &quarenta},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(niveis) != 2 || niveis[1].Unidade != "PL" || !niveis[1].Quantidade.Equal(decimal.NewFromInt(400)) || niveis[1].GTIN != "" {
			t.Fatalf("palete de 40 caixas de 10: %+v", niveis)
		}
		ident, err = fiscal.(interface {
			IdentificacaoFiscal(context.Context, []int64) (map[int64]fiscaldom.IdentificacaoFiscal, error)
		}).IdentificacaoFiscal(a.Ctx, []int64{codigo})
		if err != nil || !ident[codigo].QuantidadePorUnidade["PL"].Equal(decimal.NewFromInt(400)) || ident[codigo].GTINPorUnidade["PL"] != "" {
			t.Fatalf("a nota enxerga o palete: %+v %v", ident[codigo], err)
		}

		// Saldo por lote: o mesmo item em dois lotes, cada um com seu peso real.
		exec(`INSERT INTO stock_movements(item_code,mask,warehouse_id,movement_type,quantity,secondary_quantity,lot,created_by,enterprise_id)
			VALUES($1,'',$2,'IN',3,75,'L-A',$3,$4),($1,'',$2,'IN',2,41,'L-B',$3,$4)`, codigo, almox, actor, a.ID)
		if s, err = gov.SaldoDuasUnidades(a.Ctx, cod); err != nil {
			t.Fatal(err)
		}
		porLote := map[string]string{}
		for _, d := range s.Detalhes {
			if d.Lote != nil {
				q, _ := decimal.NewFromString(d.SegundoSaldo)
				porLote[*d.Lote] = q.String()
			}
		}
		if porLote["L-A"] != "75" || porLote["L-B"] != "41" {
			t.Fatalf("o peso de cada lote: %+v", s.Detalhes)
		}

		// Atributo livre: normaliza, sugere o nome e acha o item pelo valor.
		livres, err := gov.SalvarAtributosLivres(a.Ctx, cod, []dom.AtributoLivre{{Nome: "tonalidade", Valor: fmt.Sprintf("Nogueira %d", u)}})
		if err != nil || len(livres) != 1 || livres[0].Nome != "TONALIDADE" {
			t.Fatalf("atributo livre: %+v %v", livres, err)
		}
		if nomes, err := gov.NomesDeAtributosLivres(a.Ctx); err != nil || len(nomes) != 1 {
			t.Fatalf("nomes: %v %v", nomes, err)
		}
		if resumo, err := gov.ResumoDeAtributosLivres(a.Ctx); err != nil || len(resumo) != 1 || resumo[0].Item != cod || !strings.Contains(resumo[0].Texto, "Nogueira") {
			t.Fatalf("resumo para a busca: %+v %v", resumo, err)
		}
		achados, err := itemrepo.NewBuscaPorDescricaoPG(pool).ItensPorDescricao(a.Ctx, fmt.Sprintf("Nogueira %d", u), 5)
		if err != nil || len(achados) != 1 || achados[0].Codigo != cod {
			t.Fatalf("a busca acha o item pelo atributo: %+v %v", achados, err)
		}

		// O GTIN da caixa de um item não vira o GTIN principal de outro.
		outroItem := testutil.UniqueCode()
		exec(`INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,engineering_weight,engineering_type)
			VALUES($1,$2,'OUTRO',$3,$4,$5,'{"gross":1,"net":1,"unit":"KG"}',2)`, almox, outroItem, actor, a.ID, fmt.Sprintf("OUT-%d", outroItem))
		_, err = item_uc.NewUpdateItemUseCase(itens, permissaoTotal{ator: actor}).ExecuteBusinessCode(a.Ctx, fmt.Sprintf("OUT-%d", outroItem), atualizacao(t, `{"accounting":{"gtin":"17891000100100"}}`))
		if err == nil || !strings.Contains(err.Error(), "embalagem") {
			t.Fatalf("o GTIN já é da caixa de outro item: %v", err)
		}
	})
}
