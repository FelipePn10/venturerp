//go:build integration

// Efeitos da NF-e autorizada como pendência (migração 000401): títulos, baixa
// de estoque e pedido entram juntos numa transação ou nenhum entra, e a
// pendência fica com o motivo até o reprocesso dar certo.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run EfeitosDaNFe ./internal/infrastructure/fiscalefeitos/
package fiscalefeitos_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/fiscalefeitos"
	financialRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	itemRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	stockRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// estoqueQueFalha simula a baixa recusada (saldo bloqueado, período fechado…)
// DEPOIS de os títulos já terem sido gravados na mesma transação.
type estoqueQueFalha struct{ stockrepo.StockRepository }

func (estoqueQueFalha) CreateMovement(context.Context, *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	return nil, errors.New("período de estoque fechado")
}

type cenario struct {
	pool    *pgxpool.Pool
	ctx     context.Context
	ator    uuid.UUID
	empresa int64
	nota    int64
	item    int64
}

// novaNotaAutorizada monta uma empresa NOVA (id ≠ código, para o agendador
// provar que monta a sessão certa), um item com depósito padrão e uma nota.
func novaNotaAutorizada(t *testing.T) cenario {
	t.Helper()
	pool := testutil.Pool(t)
	c := cenario{pool: pool, ator: testutil.Actor(t, pool), item: testutil.UniqueCode()}
	code := 1_950_000_000 + testutil.UniqueCode()%40_000_000
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO enterprise(code, name, created_by) VALUES ($1, 'EFEITOS NFE', $2) RETURNING id`, code, c.ator).Scan(&c.empresa); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.ApagarEmpresasPorCodigo(t, pool, int64(code)) })
	c.ctx = context.WithValue(context.Background(), contextkey.UserKey,
		&security.AuthUser{ID: c.ator.String(), Role: "ADMIN", EnterpriseID: c.empresa, EnterpriseCode: code})

	testutil.Exec(t, pool, `INSERT INTO items(code, business_code, warehouse_code, created_by, enterprise_id)
		VALUES ($1, ($1::bigint)::text, 7, $2, $3)`, c.item, c.ator, c.empresa)
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO fiscal_exits(numero_nf, data_emissao, cfop, natureza_operacao, valor_total, status, created_by, enterprise_id)
		 VALUES (77, $3::date, '5102', 'Venda', 150, 'AUTHORIZED', $1, $2) RETURNING id`,
		c.ator, c.empresa, time.Now()).Scan(&c.nota); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_exit_items(fiscal_exit_id, item_code, cfop, quantity, unit_price, total_price)
		VALUES ($1, $2, '5102', 3, 50, 150)`, c.nota, c.item)
	// Saldo para a baixa: a saída sem estoque é recusada (saldo negativo bloqueado).
	if _, err := stockRepo.NewStockRepositorySQLC(pool).CreateMovement(c.ctx, &stockentity.StockMovement{
		ItemCode: c.item, WarehouseID: 7, MovementType: stockentity.MovementTypeIn, Quantity: decimal.NewFromFloat(10), UnitPrice: decimal.NewFromFloat(20), CreatedBy: c.ator,
	}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM fiscal_exit_pendencias WHERE fiscal_exit_id = $1`, c.nota)
		testutil.Exec(t, pool, `DELETE FROM contas_receber WHERE fiscal_exit_id = $1`, c.nota)
		testutil.Exec(t, pool, `DELETE FROM stock_movements WHERE enterprise_id = $1`, c.empresa)
		testutil.Exec(t, pool, `DELETE FROM stock_balances WHERE enterprise_id = $1`, c.empresa)
		testutil.Exec(t, pool, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id = $1`, c.nota)
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE id = $1`, c.nota)
		testutil.Exec(t, pool, `DELETE FROM items WHERE code = $1`, c.item)
		testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id = $1`, c.empresa)
	})
	return c
}

func (c cenario) efeitos(t *testing.T, falhar *bool) *fiscal_uc.EfeitosNFe {
	t.Helper()
	q, _ := testutil.Queries(t)
	return &fiscal_uc.EfeitosNFe{
		Fiscal:      fiscalRepo.NewFiscalRepositoryPG(c.pool),
		Itens:       itemRepo.NewRepositoryItemSQLC(q),
		Pendencias:  fiscalRepo.NewPendenciasPG(c.pool),
		GeraTitulos: true,
		Transacao: func(ctx context.Context, fn func(fiscal_uc.EfeitosRepos) error) error {
			tx, err := c.pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(ctx) }()
			var estoque stockrepo.StockRepository = stockRepo.NewStockRepositoryTx(tx)
			if *falhar {
				estoque = estoqueQueFalha{estoque}
			}
			if err := fn(fiscal_uc.EfeitosRepos{
				Financeiro:  financialRepo.NewFinancialRepositoryTx(tx),
				Estoque:     estoque,
				Faturamento: fiscalRepo.NewFiscalRepositoryTx(tx),
				Pendencias:  fiscalRepo.NewPendenciasTx(tx),
			}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
	}
}

func (c cenario) contar(t *testing.T, sql string) int {
	t.Helper()
	var n int
	if err := c.pool.QueryRow(context.Background(), sql, c.nota).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEfeitosDaNFeSaoTudoOuNadaEReprocessam(t *testing.T) {
	c := novaNotaAutorizada(t)
	falhar := true
	ef := c.efeitos(t, &falhar)

	if err := ef.Registrar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	err := ef.Executar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator)
	if err == nil || !strings.Contains(err.Error(), "período de estoque fechado") {
		t.Fatalf("erro = %v, quer a falha da baixa", err)
	}
	// Tudo ou nada: o título gravado antes da baixa foi desfeito junto.
	if n := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); n != 0 {
		t.Fatalf("títulos = %d depois da falha, quer 0 (a transação não desfez)", n)
	}
	pend, err := ef.DaNota(c.ctx, c.nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].Situacao != "PENDENTE" || pend[0].Tentativas != 1 ||
		!strings.Contains(pend[0].UltimoErro, "período de estoque fechado") {
		t.Fatalf("pendência = %+v, quer PENDENTE com 1 tentativa e o motivo", pend)
	}

	// Registrar de novo não duplica a pendência aberta.
	if err := ef.Registrar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	if n := c.contar(t, `SELECT count(*) FROM fiscal_exit_pendencias WHERE fiscal_exit_id = $1`); n != 1 {
		t.Fatalf("pendências = %d, quer 1", n)
	}

	// Reprocesso automático: o agendador pega a vencida de qualquer empresa e
	// roda na sessão da empresa dona.
	falhar = false
	testutil.Exec(t, c.pool, `UPDATE fiscal_exit_pendencias SET proxima_tentativa = now() - interval '1 minute' WHERE fiscal_exit_id = $1`, c.nota)
	resolvidas, _, err := ef.ProcessarVencidas(context.Background(), 1000, fiscalefeitos.Sessao)
	if err != nil {
		t.Fatal(err)
	}
	if resolvidas < 1 {
		t.Fatalf("resolvidas = %d, quer ao menos a desta nota", resolvidas)
	}
	pend, err = ef.DaNota(c.ctx, c.nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].Situacao != "RESOLVIDA" {
		t.Fatalf("pendência = %+v, quer RESOLVIDA", pend)
	}
	if n := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); n != 1 {
		t.Fatalf("títulos = %d, quer 1", n)
	}
	var deposito int64
	if err := c.pool.QueryRow(context.Background(),
		`SELECT warehouse_id FROM stock_movements WHERE reference_type = 'NF_SAIDA' AND reference_code = $1`, c.nota).Scan(&deposito); err != nil {
		t.Fatalf("baixa da nota: %v", err)
	}
	if deposito != 7 {
		t.Fatalf("depósito da baixa = %d, quer o padrão do item (7)", deposito)
	}

	// Reprocessar com tudo resolvido não aplica de novo.
	if _, err := ef.Reprocessar(c.ctx, c.nota, c.ator); err != nil {
		t.Fatal(err)
	}
	if n := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); n != 1 {
		t.Fatalf("títulos = %d depois do reprocesso, quer 1 (aplicou duas vezes)", n)
	}
}

// A pendência é da empresa: outra empresa não a vê nem a reprocessa.
func TestEfeitosDaNFePendenciaIsolaEmpresa(t *testing.T) {
	c := novaNotaAutorizada(t)
	falhar := true
	ef := c.efeitos(t, &falhar)
	if err := ef.Registrar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	outra := testutil.TenantContext(t, c.pool)
	pend, err := ef.DaNota(outra, c.nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 0 {
		t.Fatalf("outra empresa viu %d pendência(s) da nota", len(pend))
	}
	falhar = false
	if _, err := ef.Reprocessar(outra, c.nota, c.ator); err != nil {
		t.Fatal(err)
	}
	if n := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); n != 0 {
		t.Fatalf("outra empresa aplicou os efeitos da nota (%d títulos)", n)
	}
}

// O agendador lê a fila (pendência ainda aberta) enquanto a autorização aplica
// os efeitos; quando ele executa, a pendência já foi resolvida. Antes a segunda
// execução baixava o estoque e gerava os títulos de novo (visto no clone da
// Tecnofer: duas saídas de 1 M para uma NF-e de 1 M).
func TestEfeitosDaNFeNaoReaplicamDepoisDeResolvidos(t *testing.T) {
	c := novaNotaAutorizada(t)
	falhar := false
	ef := c.efeitos(t, &falhar)
	if err := ef.Registrar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := ef.Executar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
			t.Fatalf("execução %d: %v", i+1, err)
		}
	}
	if n := c.contar(t, `SELECT count(*) FROM stock_movements WHERE reference_type = 'NF_SAIDA' AND reference_code = $1`); n != 1 {
		t.Fatalf("baixas = %d, quer 1", n)
	}
	if n := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); n != 1 {
		t.Fatalf("títulos = %d, quer 1", n)
	}
}

// Execuções simultâneas (autorização, reprocesso automático e consultas de
// status ao mesmo tempo) aplicam os efeitos uma vez só.
func TestEfeitosDaNFeSimultaneosAplicamUmaVez(t *testing.T) {
	c := novaNotaAutorizada(t)
	falhar := false
	ef := c.efeitos(t, &falhar)
	if err := ef.Registrar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator); err != nil {
		t.Fatal(err)
	}
	const n = 5
	erros := make(chan error, n)
	inicio := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			<-inicio
			erros <- ef.Executar(c.ctx, c.nota, fiscalentity.PendenciaAutorizacao, c.ator)
		}()
	}
	close(inicio)
	for i := 0; i < n; i++ {
		if err := <-erros; err != nil {
			t.Fatalf("execução simultânea: %v", err)
		}
	}
	if q := c.contar(t, `SELECT count(*) FROM stock_movements WHERE reference_type = 'NF_SAIDA' AND reference_code = $1`); q != 1 {
		t.Fatalf("baixas = %d com %d execuções simultâneas, quer 1", q, n)
	}
	if q := c.contar(t, `SELECT count(*) FROM contas_receber WHERE fiscal_exit_id = $1`); q != 1 {
		t.Fatalf("títulos = %d com %d execuções simultâneas, quer 1", q, n)
	}
	var saldo decimal.Decimal
	if err := c.pool.QueryRow(context.Background(), `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id = $1 AND item_code = $2`, c.empresa, c.item).Scan(&saldo); err != nil {
		t.Fatal(err)
	}
	if !saldo.Equal(decimal.NewFromInt(7)) {
		t.Fatalf("saldo = %s, quer 7 (10 − 3 uma vez)", saldo)
	}
}
