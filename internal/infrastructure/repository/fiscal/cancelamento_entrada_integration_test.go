//go:build integration

package fiscal_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Nota lançada em KG num item estocado em M e acertada por ajuste ligado à
// nota (o caso da NF 78402 da Tecnofer): o cancelamento estorna o que a nota
// deixou no estoque (115,8975 − 97,8975 = 18 M), não a entrada bruta — antes
// exigia 115,8975 de saldo e a nota não podia ser cancelada.
func TestCancelamentoDeNotaAcertadaPorAjuste(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	bg := context.Background()
	u := testutil.UniqueCode()
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WC-%d", u), ator, emp).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	item := u
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$2,$3,$4)`, item, almox, ator, emp)
	var nota int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,serie,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_total,status,stock_status,created_by,enterprise_id)
		VALUES(78402,'1',$1::date,$1::date,'27785001000145','TUBOS LTDA',723.89,723.89,'APPROVED','CONCLUIDO',$2,$3) RETURNING id`, time.Now(), ator, emp).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM stock_movements WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM stock_balances WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM items WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM warehouse WHERE enterprise_id=$1`, emp)
	})

	movimento := func(tipo string, qtd, preco string) int64 {
		t.Helper()
		ref := stockentity.ReferenceTypeNFEntry
		m := &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: tipo, Quantity: decimal.RequireFromString(qtd),
			UnitPrice: decimal.RequireFromString(preco), ReferenceType: &ref, ReferenceCode: &nota, CreatedBy: ator}
		tx, err := pool.Begin(bg)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(bg) }()
		if err := stockrepo.CreateMovementTx(ctx, tx, emp, m); err != nil {
			t.Fatalf("movimento %s %s: %v", tipo, qtd, err)
		}
		if err := tx.Commit(bg); err != nil {
			t.Fatal(err)
		}
		return m.ID
	}
	entrada := movimento(stockentity.MovementTypeIn, "115.8975", "6.246")
	movimento(stockentity.MovementTypeAdjustment, "-97.8975", "0")
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,description,quantity,unit_price,total_price,cfop,uom,fator_conversao,quantidade_estoque,stock_movement_id,movimenta_estoque)
		VALUES($1,1,$2,'TUBO RET 80 X 60',115.8975,7.9,915.59,'1102','KG',0.15530965,18,$3,TRUE)`, nota, item, entrada)

	saldo := func() decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, emp, item).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if s := saldo(); !s.Equal(decimal.NewFromInt(18)) {
		t.Fatalf("saldo antes do cancelamento = %s, quer 18", s)
	}

	r := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		CancelarEntrada(context.Context, repository.CancelamentoEntrada) (*repository.ResultadoCancelamento, error)
	})
	res, err := r.CancelarEntrada(ctx, repository.CancelamentoEntrada{EntryID: nota, UserID: ator, Motivo: "conversão de unidade divergente"})
	if err != nil {
		t.Fatalf("cancelamento da nota acertada por ajuste: %v", err)
	}
	if res.MovimentosEstorno != 1 {
		t.Fatalf("movimentos de estorno = %d, quer 1", res.MovimentosEstorno)
	}
	if s := saldo(); !s.IsZero() {
		t.Fatalf("saldo depois do estorno = %s, quer 0 (estornar 18 M, não 115,8975)", s)
	}
	var estorno decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT quantity FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2 AND movement_type='OUT' AND reference_code=$3`, emp, item, nota).Scan(&estorno); err != nil {
		t.Fatal(err)
	}
	if !estorno.Equal(decimal.NewFromInt(18)) {
		t.Fatalf("estorno = %s, quer 18", estorno)
	}
}
