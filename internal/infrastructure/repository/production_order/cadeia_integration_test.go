//go:build integration

package production_order_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	productionrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestCadeiaOFPaiFilha: o caminho real da Tecnofer — o RN precisa da bucha (BU)
// e do tubo (TP), que também são fabricados. Firmar as sugestões, em qualquer
// ordem, liga a OF da bucha à OF do RN e ao pedido do RN; a bucha entregue fica
// reservada para aquela OF do RN e a OF de outro pedido não a leva; a OF do RN
// consome a reserva; encerrada, libera o resto. Nada disso vaza para outra
// empresa.
func TestCadeiaOFPaiFilha(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)

	rn, bu, tp := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, rn, bu, tp)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)

	// Pedido de venda do RN (o comercial usa enterprise_code).
	var pedido, linha int64
	if err := pool.QueryRow(base, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by)
		VALUES($1,$2,'P',$3,$4) RETURNING code`, testutil.UniqueCode()%1_000_000, a.Code, testutil.UniqueCode(), actor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price)
		VALUES($1,1,$2,10,100) RETURNING code`, pedido, rn).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM sales_order_planned_orders WHERE sales_order_item_code = $1`, linha)
		testutil.Exec(t, pool, `DELETE FROM sales_order_items WHERE code = $1`, linha)
		testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE code = $1`, pedido)
	})

	// O pegging do cálculo: RN cobre o pedido; BU e TP cobrem a sugestão do RN.
	sugRN, sugBU, sugTP := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	hoje := time.Now()
	testutil.Exec(t, pool, `INSERT INTO mrp_pegging(enterprise_id,plan_code,item_code,demand_kind,demand_code,need_date,supply_kind,supply_code,quantity) VALUES
		($1,0,$2,'PEDIDO_VENDA',$3,$4::date,'SUGESTAO',$5,10),
		($1,0,$6,'SUGESTAO',$5,$4::date,'SUGESTAO',$7,20),
		($1,0,$8,'SUGESTAO',$5,$4::date,'SUGESTAO',$9,10)`,
		a.ID, rn, linha, hoje, sugRN, bu, sugBU, tp, sugTP)

	planejada := func(item, sugestao int64, qtd string) (int64, int64) {
		t.Helper()
		var id, code int64
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm,mrp_suggestion_code)
			VALUES($1,$2,$3::numeric,'PRODUCTION','SALES_ORDER',$4::date,$5,1,$6,TRUE,$7) RETURNING id, code`,
			testutil.UniqueCode()%1_000_000, item, qtd, hoje, actor, a.ID, sugestao).Scan(&id, &code); err != nil {
			t.Fatalf("criando a ordem planejada: %v", err)
		}
		return id, code
	}
	demandas := independent_demand.NewIndependentDemandRepositorySQLC(sqlc.New(pool)).ComDB(pool)
	firmar := func(ctx context.Context, sugestao, code int64) {
		t.Helper()
		if err := demandas.VincularSugestao(ctx, sugestao, code, actor); err != nil {
			t.Fatalf("firmando a sugestão %d: %v", sugestao, err)
		}
	}
	contar := func(sql string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(base, sql, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	ligadoAoPedido := func(planned int64) bool {
		return contar(`SELECT COUNT(*) FROM sales_order_planned_orders WHERE planned_order_id = $1 AND sales_order_item_code = $2`, planned, linha) == 1
	}

	// De baixo para cima: a bucha é firmada antes do RN.
	buID, buCode := planejada(bu, sugBU, "20")
	firmar(a.Ctx, sugBU, buCode)
	if ligadoAoPedido(buID) {
		t.Fatal("a bucha ganhou o pedido antes de o RN ser firmado")
	}
	// Outra empresa firmando com os mesmos códigos não liga nada.
	firmar(b.Ctx, sugRN, buCode)
	if n := contar(`SELECT COUNT(*) FROM production_chain_links WHERE child_planned_order_id = $1`, buID); n != 0 {
		t.Fatalf("a outra empresa criou %d vínculo(s) na cadeia da empresa A", n)
	}
	rnID, rnCode := planejada(rn, sugRN, "10")
	for range 2 { // firmar de novo não duplica
		firmar(a.Ctx, sugRN, rnCode)
	}
	if n := contar(`SELECT COUNT(*) FROM production_chain_links WHERE child_planned_order_id = $1 AND parent_planned_order_id = $2`, buID, rnID); n != 1 {
		t.Fatalf("vínculo bucha → RN = %d, esperado 1 (firmando de baixo para cima)", n)
	}
	if !ligadoAoPedido(rnID) || !ligadoAoPedido(buID) {
		t.Fatal("o pedido do RN não chegou à ordem da bucha")
	}
	// De cima para baixo: o tubo é firmado depois do RN.
	tpID, tpCode := planejada(tp, sugTP, "10")
	firmar(a.Ctx, sugTP, tpCode)
	if n := contar(`SELECT COUNT(*) FROM production_chain_links WHERE child_planned_order_id = $1 AND parent_planned_order_id = $2`, tpID, rnID); n != 1 || !ligadoAoPedido(tpID) {
		t.Fatalf("tubo → RN = %d vínculo(s), pedido=%v; esperado 1 e o pedido", n, ligadoAoPedido(tpID))
	}

	// As OFs: a do RN precisa de 20 buchas; a OF de OUTRO pedido, que começa
	// antes, também — no estoque comum ela levaria a bucha.
	of := func(item int64, planned *int64, status string, inicio time.Time) (int64, int64) {
		t.Helper()
		var id, numero int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
			VALUES($1,$2,20,$3,$4::date,$5,$6,$7) RETURNING id, order_number`, testutil.UniqueCode()%1_000_000, item, status, inicio, actor, a.ID, planned).Scan(&id, &numero); err != nil {
			t.Fatalf("criando a OF: %v", err)
		}
		return id, numero
	}
	material := func(ofID, item int64) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
			VALUES($1,$2,'DEMAND',$3,'',20,0,$4,$5) RETURNING id`, ofID, a.ID, item, almox, actor).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ofRN, numeroRN := of(rn, &rnID, "OPEN", hoje.AddDate(0, 0, 5))
	materialRN := material(ofRN, bu)
	ofBU, _ := of(bu, &buID, "IN_PROGRESS", hoje)
	ofOutra, _ := of(rn, nil, "OPEN", hoje.AddDate(0, 0, 1))
	material(ofOutra, bu)

	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	// Conferência do início: a OF da bucha está aberta, o tubo nem tem OF, e não
	// há bucha no estoque.
	pend, err := repo.PendenciasDoInicio(a.Ctx, ofRN)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend.Filhas) != 2 || len(pend.Faltas) != 1 {
		t.Fatalf("pendências = %d filha(s) e %d falta(s), esperado 2 e 1: %+v", len(pend.Filhas), len(pend.Faltas), pend)
	}
	if recusa, _ := entity.AvaliarInicio(entity.ConferenciaInicioBloquear, pend); recusa == "" {
		t.Fatal("BLOQUEAR com OF filha aberta deixou a OF começar")
	}
	if outra, err := repo.PendenciasDoInicio(b.Ctx, ofRN); err != nil || !outra.Vazia() {
		t.Fatalf("a outra empresa enxergou as pendências da OF: %+v (%v)", outra, err)
	}
	// OF do tubo cancelada (o tubo vai ser comprado): deixa de ser pendência —
	// a falta da peça, se houver, a conferência do material já mostra.
	ofTP, _ := of(tp, &tpID, "CANCELLED", hoje)
	if pend, err = repo.PendenciasDoInicio(a.Ctx, ofRN); err != nil || len(pend.Filhas) != 1 || pend.Filhas[0].ProductionOrderID != ofBU {
		t.Fatalf("com a OF do tubo cancelada, só a bucha é pendente: %+v (%v)", pend.Filhas, err)
	}
	testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id=$1`, ofTP)

	// Pelo leitor do chão de fábrica vale a mesma conferência: BLOQUEAR recusa
	// e a OF continua aberta; AVISAR inicia e devolve o aviso na mensagem.
	parametro52 := func(valor string) {
		t.Helper()
		testutil.Exec(t, pool, `DELETE FROM planning_params WHERE enterprise_id=$1 AND param_number=52`, a.ID)
		testutil.Exec(t, pool, `INSERT INTO planning_params(param_number,param_key,value,updated_by,enterprise_id)
			VALUES(52,'CONFERE_COMPONENTES_INICIO_OF',$1,$2,$3)`, valor, actor, a.ID)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM planning_params WHERE enterprise_id=$1 AND param_number=52`, a.ID)
	})
	hash := sha256.Sum256([]byte(uuid.NewString()))
	if err := repo.CreateScanToken(a.Ctx, &entity.ScanToken{EnterpriseID: a.ID, ProductionOrderID: ofRN, TokenHash: hash[:], CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	dispositivo := "cadeia-" + uuid.NewString()
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM production_scan_events WHERE device_id=$1`, dispositivo) })
	ler := func() (*entity.ScanResult, error) {
		return repo.ExecuteScan(a.Ctx, entity.ScanCommand{EnterpriseID: a.ID, UserID: actor, TokenHash: hash[:], Action: entity.ScanStart,
			IdempotencyKey: uuid.NewString(), DeviceID: dispositivo, Fingerprint: []byte{1}})
	}
	situacao := func() string {
		t.Helper()
		var st string
		if err := pool.QueryRow(base, `SELECT status FROM production_orders WHERE id=$1`, ofRN).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	parametro52("BLOQUEAR")
	if _, err := ler(); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("OF %d", ofBU)) {
		t.Fatalf("scanner com BLOQUEAR: esperado recusa citando a OF %d da bucha, veio %v", ofBU, err)
	}
	if st := situacao(); st != "OPEN" {
		t.Fatalf("a OF recusada pelo scanner ficou %s", st)
	}
	parametro52("AVISAR")
	lido, err := ler()
	if err != nil || !strings.Contains(lido.Message, fmt.Sprintf("OF %d", ofBU)) || situacao() != "IN_PROGRESS" {
		t.Fatalf("scanner com AVISAR: esperado iniciar com o aviso da OF %d: %+v (%v), situação %s", ofBU, lido, err, situacao())
	}
	// Volta ao estado do roteiro do teste (a OF do RN começa aberta).
	testutil.Exec(t, pool, `UPDATE production_orders SET status='OPEN' WHERE id=$1`, ofRN)

	// A OF da bucha entrega 20: ficam reservadas para a OF do RN.
	if _, err := repo.RegisterDeliveryWithMovements(a.Ctx, &entity.ProductionDelivery{
		ProductionOrderID: ofBU, Quantity: decimal.NewFromInt(20), IdempotencyKey: uuid.NewString(), MovementClass: stockentity.MovementTypeProductionEntry,
		WarehouseID: almox, CreatedBy: actor, ItemCode: bu,
	}, []*stockentity.StockMovement{{
		ItemCode: bu, WarehouseID: almox, MovementType: stockentity.MovementTypeProductionEntry, Quantity: decimal.NewFromInt(20),
		ReferenceType: ptr(stockentity.ReferenceTypeProductionOrder), ReferenceCode: &ofBU, CreatedBy: actor,
	}}); err != nil {
		t.Fatalf("entregando a bucha: %v", err)
	}
	reservado := func() decimal.Decimal {
		t.Helper()
		var q decimal.Decimal
		if err := pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='OF_COMPONENTE'
			AND reference_code=$2 AND reference_item_code=$3 AND status='ACTIVE'`, a.ID, ofRN, materialRN).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if got := reservado(); !got.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("reservado para a OF do RN = %s, esperado 20", got)
	}
	// O MRP não desconta a reserva: o empenho da OF do RN já é a necessidade
	// (descontar os dois sugeria bucha em dobro).
	fotos, err := mrp_calculation.NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool).ListAllStockSnapshots(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if foto := fotos[bu]; foto == nil || !foto.DisponivelParaNecessidade().Equal(decimal.NewFromInt(20)) {
		t.Fatalf("o MRP descontou a bucha reservada para a OF do RN: %+v", foto)
	}
	linhas, err := repo.MaterialAvailability(a.Ctx, []int64{ofRN, ofOutra})
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range linhas {
		switch l.ProductionOrderID {
		case ofRN:
			if !l.Falta().IsZero() || !l.ReservedForOrder.Equal(decimal.NewFromInt(20)) {
				t.Fatalf("OF do RN: falta %s, reservado %s; esperado 0 e 20", l.Falta(), l.ReservedForOrder)
			}
		case ofOutra:
			if !l.Falta().Equal(decimal.NewFromInt(20)) {
				t.Fatalf("OF do outro pedido: falta %s, esperado 20 (a bucha é da OF do RN)", l.Falta())
			}
		}
	}

	// A OF do outro pedido não leva a bucha reservada.
	estoque := stockrepo.NewStockRepositorySQLC(pool)
	saida := func(ctx context.Context, ofID int64, qtd int64) error {
		_, err := estoque.CreateMovement(ctx, &stockentity.StockMovement{
			ItemCode: bu, WarehouseID: almox, MovementType: stockentity.MovementTypeOut, Quantity: decimal.NewFromInt(qtd),
			ReferenceType: ptr(stockentity.ReferenceTypeProductionOrder), ReferenceCode: &ofID, CreatedBy: actor,
		})
		return err
	}
	if err := saida(a.Ctx, ofOutra, 5); err == nil || !strings.Contains(err.Error(), "reservado") {
		t.Fatalf("a OF de outro pedido levou a bucha reservada: %v", err)
	}
	// A OF do RN consome a reserva dela.
	if err := saida(a.Ctx, ofRN, 8); err != nil {
		t.Fatalf("a OF do RN não conseguiu baixar a bucha dela: %v", err)
	}
	if got := reservado(); !got.Equal(decimal.NewFromInt(12)) {
		t.Fatalf("reservado depois da baixa de 8 = %s, esperado 12", got)
	}

	// A cadeia vista da bucha e do RN; da outra empresa, nada.
	cadeiaBU, err := repo.CadeiaDaOF(a.Ctx, ofBU)
	if err != nil {
		t.Fatal(err)
	}
	if len(cadeiaBU.Pedidos) != 1 || cadeiaBU.Pedidos[0].SalesOrderCode != pedido ||
		len(cadeiaBU.Pais) != 1 || cadeiaBU.Pais[0].OrderNumber == nil || *cadeiaBU.Pais[0].OrderNumber != numeroRN {
		t.Fatalf("cadeia da OF da bucha = %+v, esperado o pedido e a OF %d do RN", cadeiaBU, numeroRN)
	}
	cadeiaRN, err := repo.CadeiaDaOF(a.Ctx, ofRN)
	if err != nil {
		t.Fatal(err)
	}
	if len(cadeiaRN.Filhas) != 2 {
		t.Fatalf("filhas da OF do RN = %d, esperado 2 (bucha e tubo)", len(cadeiaRN.Filhas))
	}
	if outra, err := repo.CadeiaDaOF(b.Ctx, ofBU); err != nil || len(outra.Pedidos)+len(outra.Pais)+len(outra.Filhas) != 0 {
		t.Fatalf("a outra empresa enxergou a cadeia: %+v (%v)", outra, err)
	}

	// Encerrada a OF do RN, a sobra é liberada e a outra OF pode usar.
	testutil.Exec(t, pool, `UPDATE production_orders SET status='COMPLETED' WHERE id=$1`, ofRN)
	if got := reservado(); !got.IsZero() {
		t.Fatalf("a OF do RN concluída ainda segura %s de bucha", got)
	}
	var naLinha decimal.Decimal
	if err := pool.QueryRow(base, `SELECT COALESCE(SUM(reserved_qty),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`,
		a.ID, bu, almox).Scan(&naLinha); err != nil {
		t.Fatal(err)
	}
	if !naLinha.IsZero() {
		t.Fatalf("o saldo ficou com %s reservado depois de liberar", naLinha)
	}
	if err := saida(a.Ctx, ofOutra, 5); err != nil {
		t.Fatalf("liberada a reserva, a outra OF ainda não baixa: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

// TestReservaDoComponenteAcompanhaOSaldo: o estorno da entrega (saída liberada)
// tira do almoxarifado a peça reservada; a reserva tem de cair junto, senão
// fica reserva de peça que não existe e ela trava toda saída do item ali —
// inclusive a peça que entrou depois, para qualquer um.
func TestReservaDoComponenteAcompanhaOSaldo(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, nil)
	pai, filho := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, pai, filho)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	hoje := time.Now()

	var paiPlan, filhoPlan int64
	for _, p := range []struct {
		item int64
		dst  *int64
	}{{pai, &paiPlan}, {filho, &filhoPlan}} {
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
			VALUES($1,$2,10,'PRODUCTION','SALES_ORDER',$3::date,$4,1,$5,TRUE) RETURNING id`, testutil.UniqueCode()%1_000_000, p.item, hoje, actor, a.ID).Scan(p.dst); err != nil {
			t.Fatal(err)
		}
	}
	testutil.Exec(t, pool, `INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by) VALUES($1,$2,$3,10,$4)`, a.ID, filhoPlan, paiPlan, actor)
	of := func(item, plan int64) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
			VALUES($1,$2,10,'OPEN',$3::date,$4,$5,$6) RETURNING id`, testutil.UniqueCode()%1_000_000, item, hoje, actor, a.ID, plan).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ofPai, ofFilho := of(pai, paiPlan), of(filho, filhoPlan)
	testutil.Exec(t, pool, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
		VALUES($1,$2,'DEMAND',$3,'',10,0,$4,$5)`, ofPai, a.ID, filho, almox, actor)

	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	if _, err := repo.RegisterDeliveryWithMovements(a.Ctx, &entity.ProductionDelivery{
		ProductionOrderID: ofFilho, Quantity: decimal.NewFromInt(10), IdempotencyKey: uuid.NewString(), MovementClass: stockentity.MovementTypeProductionEntry,
		WarehouseID: almox, CreatedBy: actor, ItemCode: filho,
	}, []*stockentity.StockMovement{{
		ItemCode: filho, WarehouseID: almox, MovementType: stockentity.MovementTypeProductionEntry, Quantity: decimal.NewFromInt(10),
		ReferenceType: ptr(stockentity.ReferenceTypeProductionOrder), ReferenceCode: &ofFilho, CreatedBy: actor,
	}}); err != nil {
		t.Fatal(err)
	}
	estoque := stockrepo.NewStockRepositorySQLC(pool)
	mov := func(tipo, ref string, qtd int64, code *int64) error {
		_, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: filho, WarehouseID: almox, MovementType: tipo,
			Quantity: decimal.NewFromInt(qtd), ReferenceType: &ref, ReferenceCode: code, CreatedBy: actor})
		return err
	}
	// Estorno de 6 das 10: sobram 4 no almoxarifado, a reserva cai para 4.
	if err := mov(stockentity.MovementTypeOut, stockentity.ReferenceTypeReversal, 6, nil); err != nil {
		t.Fatalf("estorno recusado: %v", err)
	}
	var reservado, naLinha decimal.Decimal
	ler := func() {
		t.Helper()
		if err := pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='OF_COMPONENTE' AND reference_code=$2 AND status='ACTIVE'`,
			a.ID, ofPai).Scan(&reservado); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(base, `SELECT COALESCE(SUM(reserved_qty),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`,
			a.ID, filho, almox).Scan(&naLinha); err != nil {
			t.Fatal(err)
		}
	}
	ler()
	if !reservado.Equal(decimal.NewFromInt(4)) || !naLinha.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("depois do estorno de 6: reserva %s, reservado no saldo %s; esperado 4 e 4", reservado, naLinha)
	}
	// Entram 5 avulsas: são livres, qualquer documento as usa.
	if err := mov(stockentity.MovementTypeIn, stockentity.ReferenceTypeManual, 5, nil); err != nil {
		t.Fatal(err)
	}
	if err := mov(stockentity.MovementTypeOut, stockentity.ReferenceTypeManual, 5, nil); err != nil {
		t.Fatalf("a peça livre ficou presa por reserva de peça estornada: %v", err)
	}
	// A OF pai consome as 4 dela.
	if err := mov(stockentity.MovementTypeOut, stockentity.ReferenceTypeProductionOrder, 4, &ofPai); err != nil {
		t.Fatalf("a OF pai não baixou a peça dela: %v", err)
	}
	ler()
	if !reservado.IsZero() || !naLinha.IsZero() {
		t.Fatalf("no fim: reserva %s, reservado no saldo %s; esperado zero", reservado, naLinha)
	}
}

// TestOFPaiCriadaDepoisDaEntrega: firmando de baixo para cima, a bucha (lote
// agrupado de 30 para dois RN) é feita e entregue antes de a OF do primeiro RN
// existir. O segundo RN já tinha OF e levou as 10 dele na entrega. Ao nascer, a
// OF do primeiro RN reserva as 20 que sobraram da bucha — e só elas: 5 buchas
// compradas, livres no mesmo almoxarifado, continuam livres. Outra empresa
// criando OF com a mesma ordem planejada não leva nada.
func TestOFPaiCriadaDepoisDaEntrega(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	rn, bu := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, actor, rn, bu)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	hoje := time.Now()

	planejada := func(item int64, qtd int) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
			VALUES($1,$2,$3,'PRODUCTION','SALES_ORDER',$4::date,$5,1,$6,TRUE) RETURNING id`, testutil.UniqueCode()%1_000_000, item, qtd, hoje, actor, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	buPlan, rn1Plan, rn2Plan := planejada(bu, 30), planejada(rn, 25), planejada(rn, 10)
	testutil.Exec(t, pool, `INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by)
		VALUES($1,$2,$3,20,$5),($1,$2,$4,10,$5)`, a.ID, buPlan, rn1Plan, rn2Plan, actor)

	var ofBU, ofRN2 int64
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
		VALUES($1,$2,30,'IN_PROGRESS',$3::date,$4,$5,$6) RETURNING id`, testutil.UniqueCode()%1_000_000, bu, hoje, actor, a.ID, buPlan).Scan(&ofBU); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,start_date,created_by,enterprise_id,planned_order_id)
		VALUES($1,$2,10,'OPEN',$3::date,$4,$5,$6) RETURNING id`, testutil.UniqueCode()%1_000_000, rn, hoje, actor, a.ID, rn2Plan).Scan(&ofRN2); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO production_order_materials(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,attended_quantity,warehouse_id,created_by)
		VALUES($1,$2,'DEMAND',$3,'',10,0,$4,$5)`, ofRN2, a.ID, bu, almox, actor)

	repo := productionrepo.NewProductionOrderRepositoryPGX(pool)
	if _, err := repo.RegisterDeliveryWithMovements(a.Ctx, &entity.ProductionDelivery{
		ProductionOrderID: ofBU, Quantity: decimal.NewFromInt(30), IdempotencyKey: uuid.NewString(), MovementClass: stockentity.MovementTypeProductionEntry,
		WarehouseID: almox, CreatedBy: actor, ItemCode: bu, IsFinal: true,
	}, []*stockentity.StockMovement{{
		ItemCode: bu, WarehouseID: almox, MovementType: stockentity.MovementTypeProductionEntry, Quantity: decimal.NewFromInt(30),
		ReferenceType: ptr(stockentity.ReferenceTypeProductionOrder), ReferenceCode: &ofBU, CreatedBy: actor,
	}}); err != nil {
		t.Fatalf("entregando a bucha: %v", err)
	}
	estoque := stockrepo.NewStockRepositorySQLC(pool)
	if _, err := estoque.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: bu, WarehouseID: almox, MovementType: stockentity.MovementTypeIn,
		Quantity: decimal.NewFromInt(5), ReferenceType: ptr("MANUAL"), CreatedBy: actor}); err != nil {
		t.Fatalf("entrada das buchas compradas: %v", err)
	}
	reservadoPara := func(ofID int64) decimal.Decimal {
		t.Helper()
		var q decimal.Decimal
		if err := pool.QueryRow(base, `SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE reference_type='OF_COMPONENTE'
			AND reference_code=$1 AND status='ACTIVE'`, ofID).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if got := reservadoPara(ofRN2); !got.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("o RN que já tinha OF levou %s na entrega, esperado 10", got)
	}

	nova := func(ctx context.Context) (*entity.ProductionOrder, error) {
		return repo.CreateWithMaterials(ctx, &entity.ProductionOrder{OrderNumber: testutil.UniqueCode() % 1_000_000, PlannedOrderID: &rn1Plan, ItemCode: rn,
			PlannedQty: decimal.NewFromInt(25), Status: entity.StatusOpen, IsActive: true, CreatedBy: actor},
			[]*entity.ProductionOrderMaterial{{Kind: entity.MaterialDemand, ItemCode: bu, Quantity: decimal.NewFromInt(25), WarehouseID: almox, CreatedBy: actor}})
	}
	// Outra empresa, mesma ordem planejada: não reserva a bucha da empresa A.
	if outra, err := nova(b.Ctx); err == nil {
		if got := reservadoPara(outra.ID); !got.IsZero() {
			t.Fatalf("a OF da outra empresa reservou %s buchas da empresa A", got)
		}
		testutil.Exec(t, pool, `DELETE FROM production_order_materials WHERE production_order_id=$1`, outra.ID)
		testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id=$1`, outra.ID)
	}
	ofRN1, err := nova(a.Ctx)
	if err != nil {
		t.Fatalf("criando a OF do RN depois da entrega: %v", err)
	}
	if got := reservadoPara(ofRN1.ID); !got.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("a OF do RN criada depois reservou %s, esperado 20 (30 entregues − 10 do outro RN; as 5 compradas ficam livres)", got)
	}
	var livre decimal.Decimal
	if err := pool.QueryRow(base, `SELECT COALESCE(SUM(quantity - reserved_qty),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, bu).Scan(&livre); err != nil {
		t.Fatal(err)
	}
	if !livre.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("livre no estoque = %s, esperado 5 (só as compradas)", livre)
	}
	if got := reservadoPara(ofRN2); !got.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("a reserva do outro RN mudou para %s", got)
	}
}
