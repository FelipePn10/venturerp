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

// O produzido pela OF da demanda de amostra fica reservado para ela; a saída
// para a finalidade consome a reserva e o encerramento libera o resto. A
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

	entrega := func(q int64) decimal.Decimal {
		t.Helper()
		tx, err := pool.Begin(bg)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(bg) }()
		ref, ofCode := stockentity.ReferenceTypeProductionOrder, of
		if err := stockrepo.CreateMovementTx(a.Ctx, tx, a.ID, &stockentity.StockMovement{ItemCode: code, WarehouseID: almox,
			MovementType: stockentity.MovementTypeIn, Quantity: decimal.NewFromInt(q), ReferenceType: &ref, ReferenceCode: &ofCode, CreatedBy: ator}); err != nil {
			t.Fatal(err)
		}
		r, err := stockrepo.ReservarProduzidoParaDemandasTx(a.Ctx, tx, a.ID, of, code, "", almox, decimal.NewFromInt(q), ator)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(bg); err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Entrega de 3: tudo para a amostra (a demanda de estoque não reserva).
	if r := entrega(3); !r.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("reservado na primeira entrega: %s", r)
	}
	// Entrega de 6: só faltam 2 para a amostra; o resto fica livre.
	if r := entrega(6); !r.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("reservado na segunda entrega: %s", r)
	}
	if v := qtd(`SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, code); !v.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("saldo reservado: %s", v)
	}

	// Saída de 2 para a amostra consome a reserva.
	repo := stockrepo.NewStockRepositorySQLC(pool)
	ref, dem := stockentity.ReferenceTypeIndependentDemand, int64(1)
	if _, err := repo.CreateMovement(a.Ctx, &stockentity.StockMovement{ItemCode: code, WarehouseID: almox, MovementType: stockentity.MovementTypeOut,
		Quantity: decimal.NewFromInt(2), ReferenceType: &ref, ReferenceCode: &dem, CreatedBy: ator}); err != nil {
		t.Fatal(err)
	}
	if v := qtd(`SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_code=1 AND status='ACTIVE'`, a.ID); !v.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("reserva depois da saída: %s", v)
	}
	if v := qtd(`SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, code); !v.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("saldo reservado depois da saída: %s", v)
	}

	// Encerrar a demanda libera o resto da reserva.
	demRepo := demandpg.NewIndependentDemandRepositorySQLC(sqlc.New(pool)).ComDB(pool)
	if ok, err := demRepo.Encerrar(a.Ctx, 1, "ENCERRADA", "a feira foi cancelada", ator); err != nil || !ok {
		t.Fatalf("encerrar: %v %v", ok, err)
	}
	if v := qtd(`SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id=$1 AND reference_code=1 AND status='ACTIVE'`, a.ID); !v.IsZero() {
		t.Fatalf("reserva depois de encerrar: %s", v)
	}
	if v := qtd(`SELECT SUM(reserved_qty) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, code); !v.IsZero() {
		t.Fatalf("saldo reservado depois de encerrar: %s", v)
	}
	// A outra empresa não enxerga nada disso.
	if n := scalar(`SELECT count(*) FROM stock_reservations WHERE enterprise_id=$1`, b.ID); n != 0 {
		t.Fatalf("reserva vazou para outra empresa: %d", n)
	}
}
