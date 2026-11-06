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
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
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

// Fornecedor fatura o material consignado que já foi consumido: a nota gera a conta a
// pagar, mas o que já está no estoque não entra de novo. Passou do consumido, entra.
func TestFaturamentoDoConsumoNaoEntraDeNovoNoEstoque(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	bg := context.Background()
	u := testutil.UniqueCode()
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id) VALUES($1,$1,$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`,
		fmt.Sprintf("WF-%d", u), ator, emp).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	item := u
	fornecedor := u % 1_000_000
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,$2,$3,$4)`, []any{item, almox, ator, emp}},
		{`INSERT INTO suppliers(code,name,person_type,document_type,document_number,created_by,enterprise_id,register_date) VALUES($1,'Forn','JURIDICA','CNPJ','27785001000145',$2,$3,CURRENT_DATE)`, []any{fornecedor, ator, emp}},
	} {
		if _, err := pool.Exec(bg, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM stock_consignments WHERE enterprise_id=$1`, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`,
			`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`, `DELETE FROM suppliers WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, pool, q, emp)
		}
	})
	estoque := stockrepo.NewStockRepositorySQLC(pool)
	c, err := estoque.ReceberConsignacao(ctx, stockentity.Consignacao{Parceiro: fornecedor, ItemCode: item, Quantidade: decimal.NewFromInt(8), Valor: decimal.NewFromInt(30), Emissao: time.Now(), CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := estoque.ConsumirConsignacao(ctx, stockentity.BaixaConsignacao{ID: c.ID, Quantidade: decimal.NewFromInt(3), Almox: almox, Ator: ator}); err != nil {
		t.Fatal(err)
	}
	saldo := func() decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, emp, item).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}
	r := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		AprovarEntrada(context.Context, repository.AprovacaoEntrada) (*repository.ResultadoAprovacao, error)
		CancelarEntrada(context.Context, repository.CancelamentoEntrada) (*repository.ResultadoCancelamento, error)
	})
	nota := func(numero, qtd int64) int64 {
		var id, linha int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,serie,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_total,status,stock_status,created_by,enterprise_id,supplier_code)
			VALUES($1,'1',$2::date,$2::date,'27785001000145','FORN',150,150,'PENDING','PENDENTE',$3,$4,$5) RETURNING id`, numero, time.Now(), ator, emp, fornecedor).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,description,quantity,unit_price,total_price,cfop,uom,fator_conversao,quantidade_estoque,movimenta_estoque,warehouse_id)
			VALUES($1,1,$2,'PECA',$3::numeric,30,$3::numeric*30,'1113','UN',1,$3::numeric,TRUE,$4) RETURNING id`, id, item, qtd, almox).Scan(&linha); err != nil {
			t.Fatal(err)
		}
		res, err := r.AprovarEntrada(ctx, repository.AprovacaoEntrada{EntryID: id, UserID: ator, Movimentos: []repository.MovimentoEntrada{{
			ItemID: linha, ItemCode: item, WarehouseID: almox, Quantidade: decimal.NewFromInt(qtd), CustoTotal: decimal.NewFromInt(qtd * 30), MovimentaEstoque: true}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("nota %d: consignada %s", numero, res.QtdConsignada)
		return id
	}
	if s := saldo(); !s.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("depois do consumo: %s", s)
	}
	primeira := nota(9911, 2)
	if s := saldo(); !s.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("a nota de 2 entrou de novo: %s", s)
	}
	nota(9912, 4) // falta faturar 1 do consumo; os outros 3 são compra nova
	if s := saldo(); !s.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("depois da segunda nota: %s", s)
	}
	lista, err := estoque.ListarConsignacoes(ctx, stockentity.ConsignacaoEntrada, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 1 || !lista[0].Faturada.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("faturado: %+v", lista)
	}
	if _, err := r.CancelarEntrada(ctx, repository.CancelamentoEntrada{EntryID: primeira, UserID: ator, Motivo: "teste"}); err != nil {
		t.Fatal(err)
	}
	if s := saldo(); !s.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("cancelar a nota que só faturou mexeu no estoque: %s", s)
	}
	lista, err = estoque.ListarConsignacoes(ctx, stockentity.ConsignacaoEntrada, false)
	if err != nil {
		t.Fatal(err)
	}
	if !lista[0].Faturada.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("faturado depois do cancelamento: %s", lista[0].Faturada)
	}
}
