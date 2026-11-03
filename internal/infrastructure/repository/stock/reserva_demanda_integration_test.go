//go:build integration

package stock_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	demandpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O produzido pela OF da demanda de amostra fica reservado para ela, no lote em
// que entrou; o estoque livre de antes não. A saída para a finalidade consome a
// reserva, o estorno da saída a refaz e o encerramento libera o resto. A
// demanda de estoque (expectativa de venda) não reserva nada.
func TestReservaDoProduzidoParaADemanda(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) { testutil.ApagarEmpresas(t, pool, a.ID, b.ID) })
	code := testutil.UniqueCode() % 1_000_000_000
	scalar := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := pool.QueryRow(bg, q, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return v
	}
	qtd := func(q string, args ...any) decimal.Decimal {
		t.Helper()
		var v decimal.Decimal
		if err := pool.QueryRow(bg, q, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return v
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1::bigint,'RD-'||$1::bigint::text,0,'ATIVO',$2,$3)`, code, ator, a.ID)
	almox := scalar(`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES('RD','Expedição',$1,'INTERNO','NORMAL',TRUE,TRUE,$2) RETURNING id`, ator, a.ID)
	amostra := scalar(`INSERT INTO independent_demands(code,item_code,quantity,demand_date,created_by,enterprise_id,motivo)
		VALUES(1,$1,5,CURRENT_DATE+10,$2,$3,'AMOSTRA') RETURNING id`, code, ator, a.ID)
	estoque := scalar(`INSERT INTO independent_demands(code,item_code,quantity,demand_date,created_by,enterprise_id,motivo)
		VALUES(2,$1,4,CURRENT_DATE+10,$2,$3,'ESTOQUE') RETURNING id`, code, ator, a.ID)
	of := scalar(`INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id)
		VALUES($1,$2,9,'IN_PROGRESS',$3,$4) RETURNING id`, code%100000, code, ator, a.ID)
	testutil.Exec(t, pool, `INSERT INTO independent_demand_orders(enterprise_id,demand_id,production_order_id,quantity,created_by)
		VALUES($1,$2,$3,5,$5),($1,$4,$3,4,$5)`, a.ID, amostra, of, estoque, ator)

	ref, ofCode := stockentity.ReferenceTypeProductionOrder, of
	repo := stockrepo.NewStockRepositorySQLC(pool)
	// Estoque livre de antes da OF: não é da demanda e não pode ser reservado.
	if _, err := repo.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: code, WarehouseID: almox,
		MovementType: stockentity.MovementTypeIn, Quantity: decimal.NewFromInt(4), CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	entrega := func(q int64, lote string) decimal.Decimal {
		t.Helper()
		tx, err := pool.Begin(bg)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(bg) }()
		m := &stockentity.StockMovement{ItemCode: code, WarehouseID: almox, MovementType: stockentity.MovementTypeProductionEntry,
			Quantity: decimal.NewFromInt(q), ReferenceType: &ref, ReferenceCode: &ofCode, Lot: &lote, CreatedBy: ator}
		if err := stockrepo.CreateMovementTx(a.Ctx, tx, a.ID, m); err != nil {
			t.Fatal(err)
		}
		r, err := stockrepo.ReservarEntradaDaOFTx(a.Ctx, tx, a.ID, of, m, ator)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(bg); err != nil {
			t.Fatal(err)
		}
		return r
	}
	reservadoNoSaldo := func() decimal.Decimal {
		return qtd(`SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, code)
	}
	reservadoNoLote := func(lote string) decimal.Decimal {
		return qtd(`SELECT COALESCE(SUM(reserved_qty),0) FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND lot=$3`, a.ID, code, lote)
	}
	reservaAtiva := func() decimal.Decimal {
		return qtd(`SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_code=1 AND status='ACTIVE'`, a.ID)
	}
	igual := func(o string, v decimal.Decimal, n int64) {
		t.Helper()
		if !v.Equal(decimal.NewFromInt(n)) {
			t.Fatalf("%s: %s, esperado %d", o, v, n)
		}
	}
	// Entrega de 3 no lote L1: tudo para a amostra (a demanda de estoque não reserva).
	igual("reservado na primeira entrega", entrega(3, "L1"), 3)
	// Entrega de 6 no lote L2: só faltam 2 para a amostra; o resto fica livre.
	igual("reservado na segunda entrega", entrega(6, "L2"), 2)
	igual("saldo reservado", reservadoNoSaldo(), 5)
	// A reserva fica no lote em que a peça entrou — é ele que a onda de separação lê.
	igual("lote L1 reservado", reservadoNoLote("L1"), 3)
	igual("lote L2 reservado", reservadoNoLote("L2"), 2)

	// Saída de 2 para a amostra consome a reserva.
	dem := int64(1)
	refDem := stockentity.ReferenceTypeIndependentDemand
	saida, err := repo.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: code, WarehouseID: almox, MovementType: stockentity.MovementTypeOut,
		Quantity: decimal.NewFromInt(2), ReferenceType: &refDem, ReferenceCode: &dem, CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	igual("reserva depois da saída", reservaAtiva(), 3)
	igual("saldo reservado depois da saída", reservadoNoSaldo(), 3)
	igual("lote L1 depois da saída", reservadoNoLote("L1"), 1)

	// Estornada a saída, a peça volta reservada para a amostra.
	if _, err := repo.EstornarMovimento(a.Ctx, saida.ID, ator, "saiu por engano"); err != nil {
		t.Fatalf("estorno: %v", err)
	}
	igual("reserva depois do estorno", reservaAtiva(), 5)
	igual("saldo reservado depois do estorno", reservadoNoSaldo(), 5)

	// Encerrar a demanda libera o resto da reserva, no saldo e nos lotes.
	demRepo := demandpg.NewIndependentDemandRepositorySQLC(sqlc.New(pool)).ComDB(pool)
	if ok, err := demRepo.Encerrar(a.Ctx, 1, "ENCERRADA", "a feira foi cancelada", ator); err != nil || !ok {
		t.Fatalf("encerrar: %v %v", ok, err)
	}
	igual("reserva depois de encerrar", reservaAtiva(), 0)
	igual("saldo reservado depois de encerrar", reservadoNoSaldo(), 0)
	igual("lotes depois de encerrar", reservadoNoLote("L1").Add(reservadoNoLote("L2")), 0)
	// A outra empresa não enxerga nada disso.
	if n := scalar(`SELECT count(*) FROM stock_reservations WHERE enterprise_id=$1`, b.ID); n != 0 {
		t.Fatalf("reserva vazou para outra empresa: %d", n)
	}
}
