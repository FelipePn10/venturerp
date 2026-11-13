//go:build integration

// Conciliação bancária contra o banco de verdade.
//
// Existe porque a conciliação nunca tinha funcionado de ponta a ponta: a
// importação gravava o extrato e nenhuma rota o lia; e o saldo da conta somava o
// movimento conciliado por cima de `saldo_inicial`, que a baixa já tinha
// atualizado — o saldo dobraria no primeiro extrato conciliado.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run Conciliacao ./internal/infrastructure/repository/financial/
package financial_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bancada struct {
	t     *testing.T
	ctx   context.Context
	pool  *pgxpool.Pool
	repo  domrepo.FinancialRepository
	conta int64
	tag   string
}

func montarBancada(t *testing.T) *bancada {
	t.Helper()
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	ctx := testutil.TenantContext(t, pool)
	cb, err := repo.CreateContaBancaria(ctx, &entity.ContaBancaria{
		Banco: "748", Agencia: "0723", Conta: "CONC", Descricao: "Conciliação teste",
		SaldoInicial: decimal.NewFromInt(1000), IsActive: true, CreatedBy: testutil.Actor(t, pool),
	})
	if err != nil {
		t.Fatalf("criando a conta: %v", err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM extrato_bancario WHERE conta_bancaria_id = $1`, cb.ID)
		testutil.Exec(t, pool, `DELETE FROM fluxo_caixa WHERE conta_bancaria_id = $1`, cb.ID)
		testutil.Exec(t, pool, `DELETE FROM contas_bancarias WHERE id = $1`, cb.ID)
	})
	return &bancada{t: t, ctx: ctx, pool: pool, repo: repo, conta: cb.ID, tag: fmt.Sprint(testutil.UniqueCode())}
}

// extrato grava uma linha do extrato e devolve o id.
func (b *bancada) extrato(data time.Time, valor string, tipo, memo string) int64 {
	b.t.Helper()
	hash := fmt.Sprintf("conc-%s-%s-%s-%s", b.tag, data.Format("20060102"), valor, memo)
	ok, err := b.repo.SaveExtratoItem(b.ctx, b.conta, data, decimal.RequireFromString(valor), tipo, memo, "F", hash)
	if err != nil || !ok {
		b.t.Fatalf("gravando extrato %s: ok=%v err=%v", memo, ok, err)
	}
	var id int64
	if err := b.pool.QueryRow(context.Background(),
		`SELECT id FROM extrato_bancario WHERE extrato_hash = $1`, hash).Scan(&id); err != nil {
		b.t.Fatal(err)
	}
	return id
}

// baixa simula o que a baixa de título grava: o movimento NÃO conciliado e o
// saldo corrente da conta atualizado no mesmo gesto.
func (b *bancada) baixa(data time.Time, tipo entity.FluxoCaixaTipo, valor string) int64 {
	b.t.Helper()
	v := decimal.RequireFromString(valor)
	var id int64
	if err := b.pool.QueryRow(context.Background(),
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, descricao, conciliado, enterprise_id)
		 SELECT $1::date, $2, $3, $4, 'baixa teste', false, enterprise_id FROM contas_bancarias WHERE id = $4
		 RETURNING id`, data, string(tipo), v, b.conta).Scan(&id); err != nil {
		b.t.Fatalf("gravando movimento: %v", err)
	}
	if tipo == entity.FluxoCaixaTipoSaida {
		v = v.Neg()
	}
	testutil.Exec(b.t, b.pool, `UPDATE contas_bancarias SET saldo_inicial = saldo_inicial + $1 WHERE id = $2`, v, b.conta)
	return id
}

func (b *bancada) saldo() decimal.Decimal {
	b.t.Helper()
	s, err := b.repo.GetSaldoConta(b.ctx, b.conta)
	if err != nil {
		b.t.Fatal(err)
	}
	return s
}

func (b *bancada) linhas(sit domrepo.ExtratoSituacao) map[int64]*domrepo.ExtratoLinha {
	b.t.Helper()
	ls, err := b.repo.ListExtrato(b.ctx, b.conta, domrepo.ExtratoFiltro{Situacao: sit})
	if err != nil {
		b.t.Fatal(err)
	}
	out := map[int64]*domrepo.ExtratoLinha{}
	for _, l := range ls {
		out[l.ID] = l
	}
	return out
}

func querValidacao(t *testing.T, err error, oque string) {
	t.Helper()
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("%s: esperava recusa de validação, veio %v", oque, err)
	}
}

func TestConciliacaoCicloCompleto(t *testing.T) {
	b := montarBancada(t)
	d := dia(2026, time.November, 12)

	recebimento := b.extrato(d, "500.00", "CREDIT", "PIX RECEBIDO")
	boleto := b.extrato(d, "200.00", "DEBIT", "LIQUIDACAO BOLETO")
	tarifa := b.extrato(d, "8.50", "DEBIT", "TARIFA")

	// Recebimento baixado 2 dias antes: casa sozinho (janela de 3 dias).
	movReceb := b.baixa(d.AddDate(0, 0, -2), entity.FluxoCaixaTipoEntrada, "500")
	// Boleto baixado 10 dias depois: fora da janela automática, só à mão.
	movBoleto := b.baixa(d.AddDate(0, 0, 10), entity.FluxoCaixaTipoSaida, "200")
	// Mesmo valor mas ENTRADA: nunca pode casar com o débito do boleto.
	b.baixa(d, entity.FluxoCaixaTipoEntrada, "200")
	saldoAntes := b.saldo() // 1000 + 500 - 200 + 200 = 1500

	n, err := b.repo.AutoMatchExtrato(b.ctx, b.conta)
	if err != nil || n != 1 {
		t.Fatalf("automático: conciliou %d (err %v), esperava 1", n, err)
	}
	if l := b.linhas(domrepo.ExtratoConciliados)[recebimento]; l == nil || l.FluxoCaixaID == nil || *l.FluxoCaixaID != movReceb {
		t.Fatalf("recebimento não casou com o movimento certo: %+v", l)
	}
	// O defeito do saldo: conciliar não pode somar o movimento de novo.
	if s := b.saldo(); !s.Equal(saldoAntes) {
		t.Fatalf("conciliar mudou o saldo: %s → %s", saldoAntes, s)
	}

	pend := b.linhas(domrepo.ExtratoPendentes)
	if len(pend) != 2 || pend[boleto] == nil || pend[tarifa] == nil {
		t.Fatalf("pendentes = %v, esperava boleto e tarifa", pend)
	}

	// Candidatos do boleto: só a SAÍDA de 200, não a entrada de mesmo valor.
	cands, err := b.repo.CandidatosConciliacao(b.ctx, boleto)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].FluxoCaixaID != movBoleto || cands[0].DiferencaDias != 10 {
		t.Fatalf("candidatos do boleto = %+v", cands)
	}

	// Valor diferente é recusado; o certo passa; casar de novo é recusado.
	movErrado := b.baixa(d, entity.FluxoCaixaTipoSaida, "199.99")
	querValidacao(t, b.repo.ConciliarExtrato(b.ctx, boleto, movErrado), "valor diferente")
	if err := b.repo.ConciliarExtrato(b.ctx, boleto, movBoleto); err != nil {
		t.Fatalf("conciliação manual: %v", err)
	}
	querValidacao(t, b.repo.ConciliarExtrato(b.ctx, tarifa, movBoleto), "movimento já conciliado")

	// Tarifa: não tem título, entra no caixa pelo extrato e mexe no saldo.
	saldoAntes = b.saldo()
	fluxoTarifa, err := b.repo.LancarExtratoNoCaixa(b.ctx, tarifa, "")
	if err != nil {
		t.Fatalf("lançar tarifa: %v", err)
	}
	if s := b.saldo(); !s.Equal(saldoAntes.Sub(decimal.RequireFromString("8.50"))) {
		t.Fatalf("saldo depois da tarifa = %s, esperava %s - 8.50", s, saldoAntes)
	}
	l := b.linhas(domrepo.ExtratoTodos)[tarifa]
	if !l.Conciliado || !l.LancadoPeloExtrato || l.FluxoCaixaID == nil || *l.FluxoCaixaID != fluxoTarifa {
		t.Fatalf("tarifa lançada: %+v", l)
	}
	_, err = b.repo.LancarExtratoNoCaixa(b.ctx, tarifa, "")
	querValidacao(t, err, "lançar duas vezes")

	// Desfazer a tarifa apaga o movimento e devolve o saldo.
	if err := b.repo.DesfazerConciliacao(b.ctx, tarifa); err != nil {
		t.Fatal(err)
	}
	if s := b.saldo(); !s.Equal(saldoAntes) {
		t.Fatalf("desfazer a tarifa não devolveu o saldo: %s, esperava %s", s, saldoAntes)
	}
	var existe bool
	_ = b.pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM fluxo_caixa WHERE id = $1)`, fluxoTarifa).Scan(&existe)
	if existe {
		t.Fatal("o movimento criado pelo extrato continuou no caixa depois de desfazer")
	}

	// Desfazer o recebimento solta o par, mas a baixa do título fica.
	if err := b.repo.DesfazerConciliacao(b.ctx, recebimento); err != nil {
		t.Fatal(err)
	}
	var conc bool
	if err := b.pool.QueryRow(context.Background(), `SELECT conciliado FROM fluxo_caixa WHERE id = $1`, movReceb).Scan(&conc); err != nil {
		t.Fatalf("a baixa sumiu ao desfazer a conciliação: %v", err)
	}
	if conc {
		t.Fatal("a baixa continuou marcada como conciliada")
	}
	if s := b.saldo(); !s.Equal(saldoAntes) {
		t.Fatalf("desfazer a conciliação de uma baixa mexeu no saldo: %s", s)
	}
	querValidacao(t, b.repo.DesfazerConciliacao(b.ctx, recebimento), "desfazer o que não está conciliado")
}

// A outra empresa não vê nem mexe no extrato desta. Com a empresa 1 nos dois
// contextos este teste passaria sem provar nada — por isso a empresa própria.
func TestConciliacaoIsolaAEmpresa(t *testing.T) {
	b := montarBancada(t)
	d := dia(2026, time.November, 12)
	linha := b.extrato(d, "50.00", "DEBIT", "TARIFA")
	mov := b.baixa(d, entity.FluxoCaixaTipoSaida, "50")
	vizinha := empresaPropria(t, b.pool)

	quer404 := func(oque string, err error) {
		t.Helper()
		if _, ok := errorsuc.AsNotFound(err); !ok {
			t.Errorf("%s pela outra empresa: esperava não encontrado, veio %v", oque, err)
		}
	}
	_, err := b.repo.ListExtrato(vizinha, b.conta, domrepo.ExtratoFiltro{})
	quer404("listar", err)
	_, err = b.repo.CandidatosConciliacao(vizinha, linha)
	quer404("candidatos", err)
	quer404("conciliar", b.repo.ConciliarExtrato(vizinha, linha, mov))
	_, err = b.repo.LancarExtratoNoCaixa(vizinha, linha, "")
	quer404("lançar", err)
	quer404("desfazer", b.repo.DesfazerConciliacao(vizinha, linha))
	_, err = b.repo.AutoMatchExtrato(vizinha, b.conta)
	quer404("automático", err)

	if l := b.linhas(domrepo.ExtratoTodos)[linha]; l == nil || l.Conciliado {
		t.Fatalf("a outra empresa alterou a linha: %+v", l)
	}
}

func TestConciliacaoFiltraPorPeriodo(t *testing.T) {
	b := montarBancada(t)
	dentro := b.extrato(dia(2026, time.November, 10), "10.00", "DEBIT", "DENTRO")
	b.extrato(dia(2026, time.November, 1), "11.00", "DEBIT", "ANTES")
	b.extrato(dia(2026, time.November, 20), "12.00", "DEBIT", "DEPOIS")
	ini, fim := dia(2026, time.November, 5), dia(2026, time.November, 15)
	ls, err := b.repo.ListExtrato(b.ctx, b.conta, domrepo.ExtratoFiltro{Inicio: &ini, Fim: &fim})
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 1 || ls[0].ID != dentro {
		t.Fatalf("período devolveu %d linhas, esperava só a de 10/11", len(ls))
	}
}
