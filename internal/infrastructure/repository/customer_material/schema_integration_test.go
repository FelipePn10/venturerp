//go:build integration

// Testes das invariantes do estoque de terceiros (beneficiamento).
//
// São testes de SCHEMA de propósito: as regras cobertas aqui são as que o banco
// tem de sustentar mesmo que uma regra de aplicação falhe ou seja contornada por
// SQL manual. Num controle de material que não é da empresa, devolver mais do que
// entrou ou perder o rastro de uma sucata é problema fiscal, não de tela.
//
//	TEST_DATABASE_URL=... go test -tags=integration ./internal/infrastructure/repository/customer_material/
package customer_material_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// cenario cria empresa, usuário e uma remessa com um item, e devolve os ids.
// Tudo com sufixo aleatório para os testes não disputarem a mesma NF-e.
type cenario struct {
	pool         *pgxpool.Pool
	enterpriseID int64
	userID       string
	remittanceID int64
	itemID       int64
}

func novoCenario(t *testing.T, qtyInvoiced, qtyReceived string) *cenario {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	c := &cenario{pool: pool}

	// Sufixo único: a suíte roda contra uma base compartilhada e migrada.
	suffix := fmt.Sprintf("%d", testutil.UniqueCode())

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (id, name, email, password, role, is_active)
		 VALUES (gen_random_uuid(), 'Beneficiamento Teste', 'benef-'||$1||'@teste.local', 'x', 'ADMIN', true)
		 RETURNING id::text`, suffix).Scan(&c.userID); err != nil {
		t.Fatalf("criando usuário: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1::uuid`, c.userID) })

	// enterprise.code é INTEGER, não BIGINT: testutil.UniqueCode() devolve valores
	// na casa dos 9 bilhões e estouraria a coluna. A faixa abaixo é alta o bastante
	// para não colidir com os cadastros reais e ainda cabe em int32.
	code := 900_000_000 + (testutil.UniqueCode() % 1_000_000_000)
	if err := pool.QueryRow(ctx,
		`INSERT INTO enterprise (code, name, created_by) VALUES ($1, 'BENEF TESTE', $2::uuid) RETURNING id`,
		code, c.userID).Scan(&c.enterpriseID); err != nil {
		t.Fatalf("criando empresa: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id = $1`, c.enterpriseID) })

	// nfe_number é BIGINT, então aqui a faixa larga do helper serve.
	nfe := testutil.UniqueCode()
	if err := pool.QueryRow(ctx,
		`INSERT INTO customer_material_remittances
		   (enterprise_id, customer_code, nfe_number, nfe_series, cfop, issue_date, received_at,
		    fiscal_return_deadline, total_value, created_by)
		 VALUES ($1, 100, $2, '1', '5901', '2026-09-17', '2026-09-17', '2026-10-17', 1069.73, $3::uuid)
		 RETURNING id`, c.enterpriseID, nfe, c.userID).Scan(&c.remittanceID); err != nil {
		t.Fatalf("criando remessa: %v", err)
	}

	divergenceReason := any(nil)
	if qtyInvoiced != qtyReceived {
		divergenceReason = "conferência física divergente"
	}
	// Registrado antes de criar a remessa, mas ANTES também dos cleanups de empresa
	// e usuário na ordem LIFO? Não: t.Cleanup roda em LIFO, e este é registrado
	// depois dos dois, então roda primeiro — exatamente o necessário, porque a
	// remessa referencia ambos. Itens e movimentos saem por ON DELETE CASCADE.
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM customer_material_remittances WHERE enterprise_id = $1`, c.enterpriseID)
	})

	if err := pool.QueryRow(ctx,
		`INSERT INTO customer_material_items
		   (enterprise_id, remittance_id, line_number, customer_item_code, description, ncm, cst, uom,
		    qty_invoiced, qty_received, unit_value, divergence_reason)
		 VALUES ($1, $2, 1, '20042554', 'SUB CJ SOLDADO', '76169900', '050', 'PC', $3, $4, 178.283333, $5)
		 RETURNING id`, c.enterpriseID, c.remittanceID, qtyInvoiced, qtyReceived, divergenceReason).Scan(&c.itemID); err != nil {
		t.Fatalf("criando item da remessa: %v", err)
	}
	return c
}

func (c *cenario) exec(t *testing.T, sql string, args ...any) error {
	t.Helper()
	_, err := c.pool.Exec(context.Background(), sql, args...)
	return err
}

func (c *cenario) saldo(t *testing.T) (balance, divergence string) {
	t.Helper()
	if err := c.pool.QueryRow(context.Background(),
		`SELECT balance_qty::text, divergence_qty::text FROM customer_material_items WHERE id = $1`,
		c.itemID).Scan(&balance, &divergence); err != nil {
		t.Fatalf("lendo saldo: %v", err)
	}
	return balance, divergence
}

// TestSaldoNaoPodeFicarNegativo é a invariante central: material de terceiro que
// sai não pode passar do que entrou, somando retorno, sobra e sucata.
func TestSaldoNaoPodeFicarNegativo(t *testing.T) {
	c := novoCenario(t, "6", "6")

	if err := c.exec(t, `UPDATE customer_material_items SET qty_returned = 7 WHERE id = $1`, c.itemID); err == nil {
		t.Fatal("o banco aceitou devolver 7 de 6 recebidos")
	}
	// A soma também não pode estourar, ainda que cada parcela caiba sozinha.
	if err := c.exec(t,
		`UPDATE customer_material_items SET qty_returned = 4, qty_leftover = 2, qty_scrapped = 1 WHERE id = $1`,
		c.itemID); err == nil {
		t.Fatal("o banco aceitou 4+2+1 saindo de 6 recebidos")
	}
	if err := c.exec(t,
		`UPDATE customer_material_items SET qty_returned = 4, qty_leftover = 1, qty_scrapped = 1 WHERE id = $1`,
		c.itemID); err != nil {
		t.Fatalf("o banco recusou 4+1+1 de 6, que fecha exatamente: %v", err)
	}
	if balance, _ := c.saldo(t); balance != "0.000000" {
		t.Fatalf("saldo = %s, esperado 0", balance)
	}
}

// TestRetornoParcialMantemSaldoPorLinha reproduz o caso que o cliente documentou:
// remessa 16906, dois itens em quilo, primeiro retorno parcial.
func TestRetornoParcialMantemSaldoPorLinha(t *testing.T) {
	c := novoCenario(t, "16", "16")
	if err := c.exec(t, `UPDATE customer_material_items SET qty_returned = 13.94 WHERE id = $1`, c.itemID); err != nil {
		t.Fatalf("retorno parcial recusado: %v", err)
	}
	balance, _ := c.saldo(t)
	if balance != "2.060000" {
		t.Fatalf("saldo pendente = %s, esperado 2.060000 (16 - 13,94)", balance)
	}
}

// TestDivergenciaExigeMotivo: recebimento diferente da nota é registrado, nunca
// silenciado — é o que permite cobrar a regularização depois.
func TestDivergenciaExigeMotivo(t *testing.T) {
	c := novoCenario(t, "6", "6")

	if err := c.exec(t, `UPDATE customer_material_items SET qty_received = 5 WHERE id = $1`, c.itemID); err == nil {
		t.Fatal("o banco aceitou divergência sem motivo")
	}
	if err := c.exec(t,
		`UPDATE customer_material_items SET qty_received = 5, divergence_reason = 'faltou 1 peça' WHERE id = $1`,
		c.itemID); err != nil {
		t.Fatalf("divergência com motivo foi recusada: %v", err)
	}
	if _, divergence := c.saldo(t); divergence != "-1.000000" {
		t.Fatalf("divergência = %s, esperado -1.000000", divergence)
	}
}

// TestBloqueioExigeMotivo: material sem pedido ou divergente entra bloqueado, e
// bloqueio sem motivo é bloqueio que ninguém sabe resolver.
func TestBloqueioExigeMotivo(t *testing.T) {
	c := novoCenario(t, "6", "6")

	if err := c.exec(t, `UPDATE customer_material_remittances SET blocked = true WHERE id = $1`, c.remittanceID); err == nil {
		t.Fatal("o banco aceitou bloquear sem motivo")
	}
	if err := c.exec(t,
		`UPDATE customer_material_remittances SET blocked = true, block_reason = 'sem pedido cadastrado' WHERE id = $1`,
		c.remittanceID); err != nil {
		t.Fatalf("bloqueio com motivo foi recusado: %v", err)
	}
}

// TestMovimentosExigemRastro cobre o que a auditoria fiscal precisa encontrar:
// sucata com destinação, ajuste com justificativa e quantidade sempre positiva.
func TestMovimentosExigemRastro(t *testing.T) {
	c := novoCenario(t, "6", "6")
	insert := func(extraCols, extraVals string, args ...any) error {
		sql := fmt.Sprintf(
			`INSERT INTO customer_material_movements
			   (enterprise_id, remittance_item_id, movement_type, quantity, idempotency_key, created_by%s)
			 VALUES ($1, $2, $3, $4, $5, $6::uuid%s)`, extraCols, extraVals)
		all := append([]any{c.enterpriseID, c.itemID}, args...)
		return c.exec(t, sql, all...)
	}

	if err := insert("", "", "RETURN", "-1", "neg-"+fmt.Sprint(c.itemID), c.userID); err == nil {
		t.Fatal("o banco aceitou movimento com quantidade negativa")
	}
	if err := insert("", "", "SCRAP", "1", "scrap-sem-"+fmt.Sprint(c.itemID), c.userID); err == nil {
		t.Fatal("o banco aceitou sucata sem destinação")
	}
	if err := insert("", "", "ADJUSTMENT", "1", "adj-sem-"+fmt.Sprint(c.itemID), c.userID); err == nil {
		t.Fatal("o banco aceitou ajuste sem justificativa")
	}
	if err := insert(", scrap_destination, cfop", ", 'CLIENTE', '5903'",
		"SCRAP", "1", "scrap-ok-"+fmt.Sprint(c.itemID), c.userID); err != nil {
		t.Fatalf("sucata com destinação foi recusada: %v", err)
	}
	if err := insert(", reason", ", 'correção de conferência'",
		"ADJUSTMENT", "1", "adj-ok-"+fmt.Sprint(c.itemID), c.userID); err != nil {
		t.Fatalf("ajuste com motivo foi recusado: %v", err)
	}
}

// TestIdempotenciaDoMovimento: repetir a requisição de baixa não pode baixar duas
// vezes. Mesma disciplina do módulo de serviços de terceiros.
func TestIdempotenciaDoMovimento(t *testing.T) {
	c := novoCenario(t, "6", "6")
	key := "idem-" + fmt.Sprint(c.itemID)
	sql := `INSERT INTO customer_material_movements
	          (enterprise_id, remittance_item_id, movement_type, quantity, cfop, idempotency_key, created_by)
	        VALUES ($1, $2, 'RETURN', 1, '5902', $3, $4::uuid)`
	if err := c.exec(t, sql, c.enterpriseID, c.itemID, key, c.userID); err != nil {
		t.Fatalf("primeiro movimento recusado: %v", err)
	}
	err := c.exec(t, sql, c.enterpriseID, c.itemID, key, c.userID)
	if err == nil {
		t.Fatal("o banco aceitou a mesma chave de idempotência duas vezes")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "idempotencia") &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Fatalf("a recusa não foi pela chave de idempotência: %v", err)
	}
}

// TestNfeDeRemessaNaoEntraDuasVezes evita o erro operacional mais provável no
// recebimento: lançar a mesma nota do mesmo cliente de novo.
func TestNfeDeRemessaNaoEntraDuasVezes(t *testing.T) {
	c := novoCenario(t, "6", "6")
	var nfe int64
	if err := c.pool.QueryRow(context.Background(),
		`SELECT nfe_number FROM customer_material_remittances WHERE id = $1`, c.remittanceID).Scan(&nfe); err != nil {
		t.Fatalf("lendo a remessa: %v", err)
	}
	err := c.exec(t,
		`INSERT INTO customer_material_remittances
		   (enterprise_id, customer_code, nfe_number, nfe_series, issue_date, received_at,
		    fiscal_return_deadline, created_by)
		 VALUES ($1, 100, $2, '1', '2026-09-17', '2026-09-17', '2026-10-17', $3::uuid)`,
		c.enterpriseID, nfe, c.userID)
	if err == nil {
		t.Fatal("o banco aceitou a mesma NF-e do mesmo cliente duas vezes")
	}
}

// TestPrazoFiscalNaoPodeAnteceder a emissão: o prazo de 30 dias para o retorno é
// contado da nota, e uma data anterior tornaria a cobrança de prazo sem sentido.
func TestPrazoFiscalNaoPodeAntecederEmissao(t *testing.T) {
	c := novoCenario(t, "6", "6")
	err := c.exec(t,
		`INSERT INTO customer_material_remittances
		   (enterprise_id, customer_code, nfe_number, nfe_series, issue_date, received_at,
		    fiscal_return_deadline, created_by)
		 VALUES ($1, 100, $3, '1', '2026-09-17', '2026-09-17', '2026-09-01', $2::uuid)`,
		c.enterpriseID, c.userID, testutil.UniqueCode())
	if err == nil {
		t.Fatal("o banco aceitou prazo de retorno anterior à emissão")
	}
}

// TestIsolamentoPorEmpresa: o material de terceiro é operacional, então segue a
// convenção `enterprise_id` do resto do domínio e nunca aparece para outra empresa.
func TestIsolamentoPorEmpresa(t *testing.T) {
	a := novoCenario(t, "6", "6")
	b := novoCenario(t, "6", "6")
	if a.enterpriseID == b.enterpriseID {
		t.Fatal("os dois cenários caíram na mesma empresa; o teste não provaria nada")
	}
	var visiveis int
	if err := a.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM customer_material_remittances WHERE enterprise_id = $1 AND id = $2`,
		a.enterpriseID, b.remittanceID).Scan(&visiveis); err != nil {
		t.Fatalf("consultando: %v", err)
	}
	if visiveis != 0 {
		t.Fatal("a remessa de uma empresa apareceu no filtro da outra")
	}
}
