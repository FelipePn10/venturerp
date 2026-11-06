//go:build integration

package fiscal_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/frete"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A linha da nota não tem máscara: ligada ao pedido, a entrada e o cancelamento usam a
// máscara dele. Sem isso a reposição por máscara nunca via a compra chegar.
func TestEntradaLigadaAoPedidoEntraNaMascaraDoPedido(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	bg := context.Background()
	u := testutil.UniqueCode()
	codigoEmpresa, err := tenant.Code(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,$1,$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WM-%d", u), ator, emp).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	item := u
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$2,$3,$4)`, item, almox, ator, emp)
	fornecedor := u % 1_000_000
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,person_type,document_type,document_number,created_by,enterprise_id,register_date) VALUES($1,'Forn','JURIDICA','CNPJ','27785001000145',$2,$3,CURRENT_DATE)`, fornecedor, ator, emp)
	var pedido, linha, nota, linhaNota int64
	if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,status,supplier_code,created_by) VALUES($1,$2,'APPROVED',$3,$4) RETURNING code`,
		u%1_000_000, codigoEmpresa, fornecedor, ator).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO purchase_order_items(purchase_order_code,sequence,item_code,mask,requested_qty,received_qty,unit_price,total_price,status)
		VALUES($1,1,$2,'AZUL',10,0,5,50,'OPEN') RETURNING code`, pedido, item).Scan(&linha); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,serie,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_total,status,stock_status,created_by,enterprise_id)
		VALUES(9901,'1',$1::date,$1::date,'27785001000145','FORN',50,50,'PENDING','PENDENTE',$2,$3) RETURNING id`, time.Now(), ator, emp).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,description,quantity,unit_price,total_price,cfop,uom,fator_conversao,quantidade_estoque,movimenta_estoque,warehouse_id,purchase_order_item_code)
		VALUES($1,1,$2,'PECA',10,5,50,'1102','UN',1,10,TRUE,$3,$4) RETURNING id`, nota, item, almox, linha).Scan(&linhaNota); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`,
			`DELETE FROM stock_balances WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, emp)
		}
		testutil.Exec(t, pool, `DELETE FROM purchase_order_items WHERE purchase_order_code=$1`, pedido)
		testutil.Exec(t, pool, `DELETE FROM purchase_orders WHERE code=$1`, pedido)
		testutil.Exec(t, pool, `DELETE FROM suppliers WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM items WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM warehouse WHERE enterprise_id=$1`, emp)
	})
	saldo := func(mask string) decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3`, emp, item, mask).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	r := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		AprovarEntrada(context.Context, repository.AprovacaoEntrada) (*repository.ResultadoAprovacao, error)
		CancelarEntrada(context.Context, repository.CancelamentoEntrada) (*repository.ResultadoCancelamento, error)
	})
	res, err := r.AprovarEntrada(ctx, repository.AprovacaoEntrada{EntryID: nota, UserID: ator, Movimentos: []repository.MovimentoEntrada{{
		ItemID: linhaNota, ItemCode: item, WarehouseID: almox, Quantidade: decimal.NewFromInt(10), CustoTotal: decimal.NewFromInt(50),
		MovimentaEstoque: true, PurchaseOrderItemCode: &linha, FatorPedido: decimal.NewFromInt(1)}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Movimentados) != 1 || res.Movimentados[0].Mask != "AZUL" {
		t.Fatalf("%+v", res.Movimentados)
	}
	if s := saldo("AZUL"); !s.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("saldo da máscara = %s", s)
	}
	if s := saldo(""); !s.IsZero() {
		t.Fatalf("entrou sem máscara: %s", s)
	}
	// Frete e devolução leem a mesma máscara: o saldo sem máscara é zero.
	itensFrete, err := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		FreteItens(context.Context, []int64) ([]frete.Item, error)
	}).FreteItens(ctx, []int64{nota})
	if err != nil {
		t.Fatal(err)
	}
	if len(itensFrete) != 1 || itensFrete[0].Mask != "AZUL" || !itensFrete[0].EmEstoque.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("item do frete: %+v", itensFrete)
	}
	// A devolução ao fornecedor copia a máscara da linha lida aqui.
	doc, err := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		GetEntryDocument(context.Context, int64) (*entity.FiscalEntry, error)
	}).GetEntryDocument(ctx, nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Itens) != 1 || doc.Itens[0].Mask != "AZUL" {
		t.Fatalf("linha lida para a devolução: %+v", doc.Itens)
	}
	if _, err := r.CancelarEntrada(ctx, repository.CancelamentoEntrada{EntryID: nota, UserID: ator, Motivo: "teste"}); err != nil {
		t.Fatal(err)
	}
	if s := saldo("AZUL"); !s.IsZero() {
		t.Fatalf("o cancelamento não baixou a máscara: %s", s)
	}
}
