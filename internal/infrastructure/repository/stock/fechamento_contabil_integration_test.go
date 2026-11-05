//go:build integration

package stock_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type contasTeste struct{ plano, estoque, cmv, processo, absorvidos, provisao, despesa int64 }

func (e ambienteEstoque) configurarContas(t *testing.T, producao bool) contasTeste {
	t.Helper()
	pool := testutil.Pool(t)
	bg := context.Background()
	var c contasTeste
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans(plan_number,description,valid_from,status,enterprise_id) VALUES($1,'Fech','2020-01-01','A',$2) RETURNING id`,
		testutil.UniqueCode()%1_000_000+500_000, e.empresa).Scan(&c.plano); err != nil {
		t.Fatal(err)
	}
	conta := func(n string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO accounting_accounts(plan_id,account_number,description,nature_code,valid_from,is_analytic) VALUES($1,$2,$2,'D','2020-01-01',TRUE) RETURNING id`, c.plano, n).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	c.estoque, c.cmv, c.processo, c.absorvidos, c.provisao, c.despesa = conta("1"), conta("2"), conta("3"), conta("4"), conta("5"), conta("6")
	forn := conta("7")
	testutil.Exec(t, pool, `INSERT INTO accounting_posting_params(enterprise_id,plan_id,fornecedores_account_id,estoque_account_id,cmv_account_id,producao_processo_account_id,custos_absorvidos_account_id,provisao_obsoletos_account_id,despesa_provisao_account_id,contabilizar_producao)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, e.empresa, c.plano, forn, c.estoque, c.cmv, c.processo, c.absorvidos, c.provisao, c.despesa, producao)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM accounting_posting_params WHERE enterprise_id=$1`, e.empresa)
		testutil.Exec(t, pool, `DELETE FROM accounting_journal_entries WHERE empresa_id=$1`, e.empresa)
		testutil.Exec(t, pool, `DELETE FROM accounting_accounts WHERE plan_id=$1`, c.plano)
		testutil.Exec(t, pool, `DELETE FROM accounting_plans WHERE id=$1`, c.plano)
	})
	return c
}

func (e ambienteEstoque) novaOrdem(t *testing.T, status string) int64 {
	t.Helper()
	pool := testutil.Pool(t)
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id) VALUES($1,$2,10,$3,$4,$5) RETURNING id`,
		testutil.UniqueCode(), e.outro, status, e.ator, e.empresa).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM production_orders WHERE id=$1`, id) })
	return id
}

func (e ambienteEstoque) moverDaOrdem(t *testing.T, ordem, item int64, data time.Time, tipo string, qtd, preco int64) *stockentity.StockMovement {
	t.Helper()
	ref := stockentity.ReferenceTypeProductionOrder
	m := &stockentity.StockMovement{ItemCode: item, WarehouseID: e.almox, MovementType: tipo, Quantity: decimal.NewFromInt(qtd),
		UnitPrice: decimal.NewFromInt(preco), MovementDate: &data, ReferenceType: &ref, ReferenceCode: &ordem, CreatedBy: e.ator}
	got, err := e.repo.CreateMovement(e.ctx, m)
	exigir(t, err)
	return got
}

// saldoConta: débitos menos créditos, estornos incluídos.
func (e ambienteEstoque) saldoConta(t *testing.T, conta int64) decimal.Decimal {
	t.Helper()
	return e.somar(t, `SELECT COALESCE(SUM(CASE WHEN debit_account_id=$2 THEN value ELSE 0 END) - SUM(CASE WHEN credit_account_id=$2 THEN value ELSE 0 END),0)
		FROM accounting_journal_entries WHERE empresa_id=$1`, e.empresa, conta)
}

func TestFechamento_ProducaoLancaConsumoEntradaEAbsorcao(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	c := e.configurarContas(t, true)
	inicio, _ := mesPassado()
	dia := func(n int) time.Time { return inicio.AddDate(0, 0, n-1) }
	exigir(t, e.moverEm(t, dia(2), "IN", 10, 10))
	ordem := e.novaOrdem(t, "CLOSED")
	e.moverDaOrdem(t, ordem, e.item, dia(3), "OUT", 5, 0)  // material: 50
	e.moverDaOrdem(t, ordem, e.outro, dia(4), "EP", 1, 80) // produto: 80, 30 de mão de obra

	sim, err := e.repo.FecharMes(e.ctx, inicio, e.ator, true)
	exigir(t, err)
	igual(t, "consumo simulado", sim.ProducaoConsumo, "50")
	igual(t, "entrada simulada", sim.ProducaoEntrega, "80")
	igual(t, "absorvido simulado", sim.ProducaoAbsorvida, "30")
	igual(t, "simulação não lança", e.somar(t, `SELECT COUNT(*) FROM accounting_journal_entries WHERE empresa_id=$1`, e.empresa), "0")

	res, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	if !res.Contabilizado {
		t.Fatalf("não contabilizou: %+v", res)
	}
	igual(t, "processo zera", e.saldoConta(t, c.processo), "0")
	igual(t, "estoque sobe a mão de obra", e.saldoConta(t, c.estoque), "30")
	igual(t, "absorvido", e.saldoConta(t, c.absorvidos), "-30")

	f, err := e.repo.Fechamentos(e.ctx)
	exigir(t, err)
	exigir(t, e.repo.ReabrirMes(e.ctx, f[0].ID, "teste", e.ator))
	igual(t, "reabrir estorna o estoque", e.saldoConta(t, c.estoque), "0")
	igual(t, "reabrir estorna o absorvido", e.saldoConta(t, c.absorvidos), "0")

	// Fechar de novo não soma o que a reabertura desfez.
	_, err = e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	igual(t, "refechar lança uma vez só", e.saldoConta(t, c.estoque), "30")
}

// Devolução de material à ordem reduz o consumo do mês.
func TestFechamento_DevolucaoReduzOConsumo(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	c := e.configurarContas(t, true)
	inicio, _ := mesPassado()
	dia := func(n int) time.Time { return inicio.AddDate(0, 0, n-1) }
	exigir(t, e.moverEm(t, dia(2), "IN", 10, 10))
	ordem := e.novaOrdem(t, "IN_PROGRESS")
	e.moverDaOrdem(t, ordem, e.item, dia(3), "OUT", 5, 0)
	e.moverDaOrdem(t, ordem, e.item, dia(4), "IN", 3, 10)
	res, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	igual(t, "consumo líquido", res.ProducaoConsumo, "20")
	igual(t, "ordem aberta fica em processo", e.saldoConta(t, c.processo), "20")
}

func TestFechamento_ContaFaltandoImpedeFecharComProducaoLigada(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	e.configurarContas(t, true)
	testutil.Exec(t, testutil.Pool(t), `UPDATE accounting_posting_params SET producao_processo_account_id=NULL WHERE enterprise_id=$1`, e.empresa)
	inicio, _ := mesPassado()
	exigir(t, e.moverEm(t, inicio.AddDate(0, 0, 1), "IN", 10, 10))
	e.moverDaOrdem(t, e.novaOrdem(t, "IN_PROGRESS"), e.item, inicio.AddDate(0, 0, 2), "OUT", 1, 0)
	sim, err := e.repo.FecharMes(e.ctx, inicio, e.ator, true)
	exigir(t, err)
	if len(sim.Pendencias) != 1 {
		t.Fatalf("a simulação aponta a conta: %+v", sim.Pendencias)
	}
	if _, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false); !ehValidacao(err) {
		t.Fatalf("esperava recusa, veio %v", err)
	}
}

func TestFechamento_ProducaoDesligadaNaoLanca(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	e.configurarContas(t, false)
	inicio, _ := mesPassado()
	exigir(t, e.moverEm(t, inicio.AddDate(0, 0, 1), "IN", 10, 10))
	e.moverDaOrdem(t, e.novaOrdem(t, "IN_PROGRESS"), e.item, inicio.AddDate(0, 0, 2), "OUT", 1, 0)
	_, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	igual(t, "nada lançado", e.somar(t, `SELECT COUNT(*) FROM accounting_journal_entries WHERE empresa_id=$1`, e.empresa), "0")
}

func TestFechamento_ProvisaoLancaAVariacaoDoMes(t *testing.T) {
	e := novoAmbienteEstoque(t)
	e.limparFaseC(t)
	c := e.configurarContas(t, false)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `INSERT INTO stock_parameters(enterprise_id,expired_lot_action,obsolete_days,obsolete_provision_pct) VALUES($1,'BLOQUEAR',30,50)`, e.empresa)
	inicio, _ := mesPassado()
	antes := inicio.AddDate(0, -1, 0)
	exigir(t, e.moverEm(t, antes.AddDate(0, 0, 1), "IN", 10, 10)) // 100, parado há mais de 30 dias no fim

	r1, err := e.repo.FecharMes(e.ctx, antes, e.ator, false)
	exigir(t, err)
	igual(t, "no primeiro mês ainda gira", r1.ProvisaoObsoletos, "0")

	r2, err := e.repo.FecharMes(e.ctx, inicio, e.ator, false)
	exigir(t, err)
	igual(t, "provisão 50% de 100", r2.ProvisaoObsoletos, "50")
	igual(t, "variação", r2.ProvisaoVariacao, "50")
	igual(t, "despesa", e.saldoConta(t, c.despesa), "50")
	igual(t, "provisão", e.saldoConta(t, c.provisao), "-50")
}
