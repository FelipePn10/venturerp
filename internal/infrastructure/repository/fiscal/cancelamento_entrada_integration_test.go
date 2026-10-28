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
// exigia 115,8975 de saldo e a nota não podia ser cancelada. Com duas linhas do
// mesmo item o ajuste entra uma vez só, e a sobra de uma linha passa à seguinte.
func TestCancelamentoDeNotaAcertadaPorAjuste(t *testing.T) {
	casos := []struct {
		nome     string
		linhas   []string
		ajuste   string
		estornos []string
	}{
		{"NF 78402: uma linha", []string{"115.8975"}, "-97.8975", []string{"18"}},
		{"duas linhas do mesmo item, ajuste maior que a primeira", []string{"10", "10"}, "-12", []string{"8"}},
		{"duas linhas do mesmo item sem ajuste", []string{"4", "6"}, "", []string{"4", "6"}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) { cancelarNotaComAjuste(t, c.linhas, c.ajuste, c.estornos) })
	}
}

func cancelarNotaComAjuste(t *testing.T, linhas []string, ajuste string, estornos []string) {
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
	var titulo int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_pagar(numero_documento,data_emissao,data_vencimento,valor_bruto,status,fiscal_entry_id,criado_por,enterprise_id)
		VALUES('78402/1',$1::date,$1::date + 30,723.89,'ABERTO',$2,$3,$4) RETURNING id`, time.Now(), nota, ator, emp).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM contas_pagar WHERE enterprise_id=$1`, emp)
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
	esperado := decimal.Zero
	for i, q := range linhas {
		entrada := movimento(stockentity.MovementTypeIn, q, "6.246")
		testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,description,quantity,unit_price,total_price,cfop,uom,fator_conversao,quantidade_estoque,stock_movement_id,movimenta_estoque)
			VALUES($1,$2,$3,'TUBO RET 80 X 60',$4::numeric,7.9,10,'1102','KG',1,$4::numeric,$5,TRUE)`, nota, i+1, item, q, entrada)
		esperado = esperado.Add(decimal.RequireFromString(q))
	}
	if ajuste != "" {
		movimento(stockentity.MovementTypeAdjustment, ajuste, "0")
		esperado = esperado.Add(decimal.RequireFromString(ajuste))
	}

	saldo := func() decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, emp, item).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	if s := saldo(); !s.Equal(esperado) {
		t.Fatalf("saldo antes do cancelamento = %s, quer %s", s, esperado)
	}

	r := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		CancelarEntrada(context.Context, repository.CancelamentoEntrada) (*repository.ResultadoCancelamento, error)
	})
	res, err := r.CancelarEntrada(ctx, repository.CancelamentoEntrada{EntryID: nota, UserID: ator, Motivo: "conversão de unidade divergente"})
	if err != nil {
		t.Fatalf("cancelamento da nota acertada por ajuste: %v", err)
	}
	if res.MovimentosEstorno != len(estornos) {
		t.Fatalf("movimentos de estorno = %d, quer %d", res.MovimentosEstorno, len(estornos))
	}
	// O título da nota sai do contas a pagar: cancelado E inativo (a lista do
	// financeiro filtra is_active; antes ficava "cancelado" mas visível).
	var st string
	var ativo bool
	if err := pool.QueryRow(bg, `SELECT status, is_active FROM contas_pagar WHERE id=$1`, titulo).Scan(&st, &ativo); err != nil {
		t.Fatal(err)
	}
	if st != "CANCELADO" || ativo || res.TitulosCancelados != 1 {
		t.Fatalf("título da nota: status=%s ativo=%v cancelados=%d, quer CANCELADO/false/1", st, ativo, res.TitulosCancelados)
	}
	if s := saldo(); !s.IsZero() {
		t.Fatalf("saldo depois do estorno = %s, quer 0", s)
	}
	rows, err := pool.Query(bg, `SELECT quantity FROM stock_movements WHERE enterprise_id=$1 AND item_code=$2 AND movement_type='OUT' AND reference_code=$3 ORDER BY id`, emp, item, nota)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var q decimal.Decimal
		if err := rows.Scan(&q); err != nil {
			t.Fatal(err)
		}
		got = append(got, q.String())
	}
	if fmt.Sprint(got) != fmt.Sprint(estornos) {
		t.Fatalf("estornos = %v, quer %v", got, estornos)
	}
}
