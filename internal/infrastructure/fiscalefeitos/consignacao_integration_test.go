//go:build integration

package fiscalefeitos_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	itemRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	stockRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func dq(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// tipoDeNota cadastra um tipo que só mexe no estoque.
func (c cenario) tipoDeNota(t *testing.T, tipo, estoque string) int64 {
	t.Helper()
	var id int64
	if err := c.pool.QueryRow(context.Background(),
		`INSERT INTO invoice_types(code,description,type,stock_movement,updates_inventory,generates_financial_title,generates_revenue,baixa_pedido,enterprise_id)
		 VALUES($1,$2,$3::invoice_type_enum,$4::invoice_stock_enum,TRUE,FALSE,FALSE,FALSE,$5) RETURNING id`,
		testutil.UniqueCode(), tipo, tipo, estoque, c.empresa).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (c cenario) notaDoTipo(t *testing.T, tipo int64, cliente *int64, qtd string) int64 {
	t.Helper()
	var id int64
	if err := c.pool.QueryRow(context.Background(),
		`INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,valor_total,status,created_by,enterprise_id,invoice_type_id,customer_code,chave_acesso)
		 VALUES($1,CURRENT_DATE,'5917','Consignação',100,'AUTHORIZED',$2,$3,$4,$5,$6) RETURNING id`,
		testutil.UniqueCode()%1_000_000, c.ator, c.empresa, tipo, cliente, nil).Scan(&id); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, c.pool, `INSERT INTO fiscal_exit_items(fiscal_exit_id,item_code,cfop,quantity,unit_price,total_price) VALUES($1,$2,'5917',$3::numeric,20,$3::numeric*20)`, id, c.item, qtd)
	return id
}

func (c cenario) efeitosComTipos(t *testing.T) *fiscal_uc.EfeitosNFe {
	t.Helper()
	q, _ := testutil.Queries(t)
	return &fiscal_uc.EfeitosNFe{
		Fiscal: fiscalRepo.NewFiscalRepositoryPG(c.pool), Itens: itemRepo.NewRepositoryItemSQLC(q), Pendencias: fiscalRepo.NewPendenciasPG(c.pool),
		TiposNota: fiscalRepo.NewCadastroTributarioPG(c.pool), Consignacao: stockRepo.NewStockRepositorySQLC(c.pool),
		Transacao: func(ctx context.Context, fn func(fiscal_uc.EfeitosRepos) error) error {
			tx, err := c.pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)
			if err := fn(fiscal_uc.EfeitosRepos{Financeiro: financialRepo.NewFinancialRepositoryTx(tx), Estoque: stockRepo.NewStockRepositoryTx(tx),
				Pendencias: fiscalRepo.NewPendenciasTx(tx), Consignacao: stockRepo.NewStockRepositoryTx(tx)}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
	}
}

func (c cenario) aplicar(t *testing.T, e *fiscal_uc.EfeitosNFe, nota int64) error {
	t.Helper()
	if err := e.Registrar(c.ctx, nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	return e.Executar(c.ctx, nota, fiscalentity.PendenciaAutorizacao, c.ator)
}

func saldo(t *testing.T, c cenario, almox int64) decimal.Decimal {
	t.Helper()
	var q decimal.Decimal
	if err := c.pool.QueryRow(context.Background(), `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, c.empresa, c.item, almox).Scan(&q); err != nil {
		t.Fatal(err)
	}
	return q
}

func exige(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func igualQ(t *testing.T, oque string, v decimal.Decimal, quero string) {
	t.Helper()
	if !v.Equal(dq(quero)) {
		t.Errorf("%s: %s, quero %s", oque, v, quero)
	}
}

func novoAlmoxConsignacao(t *testing.T, c cenario) int64 {
	t.Helper()
	almox := testutil.NovoAlmoxarifado(t, c.pool, c.empresa)
	testutil.Exec(t, c.pool, `UPDATE warehouse SET disposition=FALSE WHERE id=$1`, almox)
	testutil.Exec(t, c.pool, `INSERT INTO stock_parameters(enterprise_id,consignment_warehouse_id,consignment_days) VALUES($1,$2,90)`, c.empresa, almox)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notification_outbox WHERE enterprise_id=$1`, `DELETE FROM stock_consignments WHERE enterprise_id=$1`, `DELETE FROM stock_parameters WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_exit_pendencias WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`,
			`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`,
			`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM invoice_types WHERE enterprise_id=$1`,
		} {
			testutil.Exec(t, c.pool, q, c.empresa)
		}
	})
	return almox
}

// Remessa, venda parcial, devolução e cancelamento pelo caminho real dos efeitos da nota.
func TestConsignacaoRemessaVendaDevolucaoECancelamento(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	e := c.efeitosComTipos(t)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	cliente := int64(777)
	remessa := c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA")
	venda := c.tipoDeNota(t, "VENDA_CONSIGNACAO", "ATUALIZA")

	n1 := c.notaDoTipo(t, remessa, &cliente, "6")
	exige(t, c.aplicar(t, e, n1))
	igualQ(t, "origem", saldo(t, c, c.almox), "4")
	igualQ(t, "consignação", saldo(t, c, consig), "6")
	custo, err := contabil(c).CustoDaSaida(c.ctx, n1)
	exige(t, err)
	igualQ(t, "a remessa não tem CMV", custo, "0")

	// Vender acima do saldo é barrado antes da SEFAZ.
	n2 := c.notaDoTipo(t, venda, &cliente, "7")
	ex, err := fiscalRepo.NewFiscalRepositoryPG(c.pool).GetExitByID(c.ctx, n2)
	exige(t, err)
	itens, err := fiscalRepo.NewFiscalRepositoryPG(c.pool).GetExitItems(c.ctx, n2)
	exige(t, err)
	if err := e.PreConferirDepositos(c.ctx, ex, itens); err == nil {
		t.Fatal("venda acima do saldo consignado passou da conferência")
	}

	n3 := c.notaDoTipo(t, venda, &cliente, "2")
	exige(t, c.aplicar(t, e, n3))
	igualQ(t, "consignação depois da venda", saldo(t, c, consig), "4")
	custo, err = contabil(c).CustoDaSaida(c.ctx, n3)
	exige(t, err)
	igualQ(t, "o CMV nasce na venda", custo, "40")
	s, err := repo.SaldoConsignado(c.ctx, cliente, c.item, "", nil)
	exige(t, err)
	igualQ(t, "saldo do cliente", s, "4")

	// Remessa com venda não cancela.
	if err := repo.PodeCancelarNotaDeEstoque(c.ctx, n1); err == nil {
		t.Fatal("cancelou remessa com venda")
	}

	// Cancelar a venda devolve o saldo ao cliente e o estoque à consignação.
	_, err = contabil(c).EstornarEstoqueDaSaida(c.ctx, n3, c.ator)
	exige(t, err)
	igualQ(t, "consignação depois do cancelamento", saldo(t, c, consig), "6")
	s, err = repo.SaldoConsignado(c.ctx, cliente, c.item, "", nil)
	exige(t, err)
	igualQ(t, "saldo volta", s, "6")
	_, err = contabil(c).EstornarEstoqueDaSaida(c.ctx, n3, c.ator)
	exige(t, err)
	s, _ = repo.SaldoConsignado(c.ctx, cliente, c.item, "", nil)
	igualQ(t, "cancelar de novo não soma", s, "6")

	lista, err := repo.ListarConsignacoes(c.ctx, "SAIDA", true)
	exige(t, err)
	if len(lista) != 1 {
		t.Fatalf("%+v", lista)
	}
	doc := "NF 1918"
	cons, err := repo.DevolverConsignacao(c.ctx, stockentity.BaixaConsignacao{ID: lista[0].ID, Quantidade: dq("6"), Documento: &doc, Ator: c.ator})
	exige(t, err)
	if cons.Status != stockentity.ConsignacaoEncerrada {
		t.Errorf("zerou e não encerrou: %+v", cons)
	}
	igualQ(t, "origem cheia", saldo(t, c, c.almox), "10")
	igualQ(t, "consignação vazia", saldo(t, c, consig), "0")
	if _, err := repo.DevolverConsignacao(c.ctx, stockentity.BaixaConsignacao{ID: lista[0].ID, Quantidade: dq("1"), Ator: c.ator}); err == nil {
		t.Fatal("devolveu de consignação encerrada")
	}
}

func TestConsignacaoCancelarARemessaVoltaTudo(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	e := c.efeitosComTipos(t)
	cliente := int64(778)
	n := c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "5")
	exige(t, c.aplicar(t, e, n))
	_, err := contabil(c).EstornarEstoqueDaSaida(c.ctx, n, c.ator)
	exige(t, err)
	igualQ(t, "origem", saldo(t, c, c.almox), "10")
	igualQ(t, "consignação", saldo(t, c, consig), "0")
	lista, err := stockRepo.NewStockRepositorySQLC(c.pool).ListarConsignacoes(c.ctx, "SAIDA", true)
	exige(t, err)
	if len(lista) != 0 {
		t.Fatalf("a remessa cancelada segue aberta: %+v", lista)
	}
}

// Material do fornecedor: fora do estoque até ser consumido, e aí pelo valor combinado.
func TestConsignacaoRecebidaSoEntraNoEstoqueAoConsumir(t *testing.T) {
	c := novaNotaAutorizada(t)
	novoAlmoxConsignacao(t, c)
	fornecedor := testutil.UniqueCode() % 1_000_000
	testutil.Exec(t, c.pool, `INSERT INTO suppliers(code,name,person_type,document_type,document_number,created_by,enterprise_id,register_date) VALUES($1,'Forn','JURIDICA','CNPJ','11222333000181',$2,$3,CURRENT_DATE)`, fornecedor, c.ator, c.empresa)
	t.Cleanup(func() {
		testutil.Exec(t, c.pool, `DELETE FROM suppliers WHERE code=$1 AND enterprise_id=$2`, fornecedor, c.empresa)
	})
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	r, err := repo.ReceberConsignacao(c.ctx, stockentity.Consignacao{Parceiro: fornecedor, ItemCode: c.item, Quantidade: dq("8"), Valor: dq("30"), Emissao: mesmoDia(), CreatedBy: c.ator})
	exige(t, err)
	igualQ(t, "não entra no estoque", saldo(t, c, c.almox), "10")
	out, err := repo.ConsumirConsignacao(c.ctx, stockentity.BaixaConsignacao{ID: r.ID, Quantidade: dq("2"), Almox: c.almox, Ator: c.ator})
	exige(t, err)
	igualQ(t, "consumo entra", saldo(t, c, c.almox), "12")
	igualQ(t, "saldo do fornecedor", out.Saldo(), "6")
	out, err = repo.DevolverConsignacao(c.ctx, stockentity.BaixaConsignacao{ID: r.ID, Quantidade: dq("6"), Ator: c.ator})
	exige(t, err)
	igualQ(t, "devolver não mexe no estoque", saldo(t, c, c.almox), "12")
	if out.Status != stockentity.ConsignacaoEncerrada {
		t.Errorf("%+v", out)
	}
}

type irmas struct {
	origem  cenario
	destino cenario
	chave   string
	transit int64
}

// duasFiliais monta dois estabelecimentos da mesma raiz, cada um com seu item.
func duasFiliais(t *testing.T, raizDestino string) irmas {
	t.Helper()
	o := novaNotaAutorizada(t)
	d := novaNotaAutorizada(t)
	testutil.Exec(t, o.pool, `INSERT INTO fiscal_configs(enterprise_id,cnpj_empresa,razao_social,updated_by) VALUES($1,'11222333000181','Matriz',$2)`, o.empresa, o.ator)
	testutil.Exec(t, o.pool, `INSERT INTO fiscal_configs(enterprise_id,cnpj_empresa,razao_social,updated_by) VALUES($1,$2,'Filial',$3)`, d.empresa, raizDestino+"000262", d.ator)
	transito := testutil.NovoAlmoxarifado(t, o.pool, o.empresa)
	testutil.Exec(t, o.pool, `UPDATE warehouse SET location='TRANSITO' WHERE id=$1`, transito)
	chave := "41261111222333000181550010000000011000000011"
	t.Cleanup(func() {
		testutil.Exec(t, o.pool, `DELETE FROM stock_cnpj_transfers WHERE origin_id=$1`, o.empresa)
		for _, e := range []int64{o.empresa, d.empresa} {
			testutil.Exec(t, o.pool, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, e)
			testutil.Exec(t, o.pool, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, e)
		}
		testutil.Exec(t, o.pool, `DELETE FROM fiscal_exit_pendencias WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM stock_movements WHERE enterprise_id=$1`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM stock_balances WHERE enterprise_id=$1`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM invoice_types WHERE enterprise_id=$1`, o.empresa)
		testutil.Exec(t, o.pool, `DELETE FROM warehouse WHERE id=$1`, transito)
	})
	return irmas{origem: o, destino: d, chave: chave, transit: transito}
}

func (i irmas) notaDeTransferencia(t *testing.T, cnpj string) int64 {
	t.Helper()
	n := i.origem.notaDoTipo(t, i.origem.tipoDeNota(t, "TRANSFERENCIA", "ATUALIZA"), nil, "4")
	testutil.Exec(t, i.origem.pool, `UPDATE fiscal_exits SET cnpj_destinatario=$2, chave_acesso=$3 WHERE id=$1`, n, cnpj, i.chave)
	return n
}

func emTx(t *testing.T, c cenario, fn func(tx pgx.Tx) error) {
	t.Helper()
	tx, err := c.pool.Begin(context.Background())
	exige(t, err)
	defer tx.Rollback(context.Background())
	exige(t, fn(tx))
	exige(t, tx.Commit(context.Background()))
}

func TestTransferenciaEntreCNPJsFicaEmTransitoAteAEntradaDoDestino(t *testing.T) {
	i := duasFiliais(t, "11222333")
	o := i.origem
	e := o.efeitosComTipos(t)
	n := i.notaDeTransferencia(t, "11222333000262")
	exige(t, o.aplicar(t, e, n))
	igualQ(t, "origem", saldo(t, o, o.almox), "6")
	igualQ(t, "trânsito segue no estoque da origem", saldo(t, o, i.transit), "4")

	// Entrada de outra nota não mexe.
	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.ReceberEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, "outra", 1, i.destino.ator)
	})
	igualQ(t, "trânsito intacto", saldo(t, o, i.transit), "4")

	// Empresa que não é o destino não baixa, mesmo com a chave certa.
	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.ReceberEntreCNPJsTx(o.ctx, tx, o.empresa, i.chave, 1, o.ator)
	})
	igualQ(t, "só o destino recebe", saldo(t, o, i.transit), "4")

	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.ReceberEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, i.chave, 555, i.destino.ator)
	})
	igualQ(t, "trânsito baixado", saldo(t, o, i.transit), "0")
	repo := stockRepo.NewStockRepositorySQLC(o.pool)
	if err := repo.PodeCancelarNotaDeEstoque(o.ctx, n); err == nil {
		t.Fatal("cancelou nota já recebida")
	}
	// Repetir a entrada não baixa de novo.
	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.ReceberEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, i.chave, 555, i.destino.ator)
	})
	igualQ(t, "sem baixa dupla", saldo(t, o, i.transit), "0")

	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.DesfazerRecebimentoEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, 555, i.destino.ator)
	})
	igualQ(t, "entrada cancelada volta ao trânsito", saldo(t, o, i.transit), "4")

	_, err := contabil(o).EstornarEstoqueDaSaida(o.ctx, n, o.ator)
	exige(t, err)
	igualQ(t, "nota cancelada volta à origem", saldo(t, o, o.almox), "10")
	igualQ(t, "trânsito vazio", saldo(t, o, i.transit), "0")

	vistas, err := repo.TransferenciasEntreCNPJs(i.destino.ctx)
	exige(t, err)
	if len(vistas) != 1 || vistas[0].Status != stockentity.TransferenciaCancelada {
		t.Fatalf("%+v", vistas)
	}
}

// Raiz diferente é outra pessoa jurídica: a nota baixa o estoque como qualquer saída.
func TestTransferenciaParaOutraRaizNaoCruzaEmpresas(t *testing.T) {
	i := duasFiliais(t, "99888777")
	o := i.origem
	n := i.notaDeTransferencia(t, "99888777000262")
	exige(t, o.aplicar(t, o.efeitosComTipos(t), n))
	igualQ(t, "baixa comum", saldo(t, o, o.almox), "6")
	igualQ(t, "sem trânsito", saldo(t, o, i.transit), "0")
	vistas, err := stockRepo.NewStockRepositorySQLC(o.pool).TransferenciasEntreCNPJs(o.ctx)
	exige(t, err)
	if len(vistas) != 0 {
		t.Fatalf("%+v", vistas)
	}
}

func mesmoDia() time.Time { return time.Now() }

func contabil(c cenario) fiscal_uc.SaidaContabil {
	return fiscalRepo.NewFiscalRepositoryPG(c.pool).(fiscal_uc.SaidaContabil)
}
