//go:build integration

// Conferência de empresa nas referências (migração 000402): a FK diz que o
// registro apontado existe; o gatilho diz que ele é da MESMA empresa.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run ConferenciaDeEmpresa ./internal/infrastructure/database/
package database_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Migration que cria tabela ou FK nova precisa rodar de novo
// `SELECT instalar_conferencia_de_empresa();`. Este teste é o que avisa.
func TestConferenciaDeEmpresaCobreTodosOsVinculos(t *testing.T) {
	pool := testutil.Pool(t)
	rows, err := pool.Query(context.Background(), `
		SELECT v.filho || '.' || v.coluna
		  FROM conferencia_empresa_vinculos() v
		 WHERE NOT EXISTS (
		        SELECT 1
		          FROM pg_trigger g
		          JOIN pg_class c ON c.oid = g.tgrelid AND c.relname = v.filho
		          JOIN pg_proc p ON p.oid = g.tgfoid
		         WHERE g.tgname = 'zz_conferir_empresa_ref'
		           AND p.prosrc LIKE '%NEW.' || v.coluna || ' IS NOT NULL%')
		 ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var faltando []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		faltando = append(faltando, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(faltando) > 0 {
		t.Fatalf("vínculos sem conferência de empresa (a migration nova precisa chamar instalar_conferencia_de_empresa()): %s",
			strings.Join(faltando, ", "))
	}
}

type empresaDeTeste struct {
	id, code int64
}

func novaEmpresaDeTeste(t *testing.T, pool *pgxpool.Pool, nome string) empresaDeTeste {
	t.Helper()
	e := empresaDeTeste{code: 1_900_000_000 + testutil.UniqueCode()%90_000_000}
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO enterprise(code, name) VALUES ($1, $2) RETURNING id`, e.code, nome).Scan(&e.id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id = $1`, e.id) })
	return e
}

func exigeVinculoRecusado(t *testing.T, err error, restricao string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" || pgErr.ConstraintName != restricao {
		t.Fatalf("erro = %v, quer foreign_key_violation da restrição %s", err, restricao)
	}
}

func TestConferenciaDeEmpresaRecusaReferenciaCruzada(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	ator := testutil.Actor(t, pool)
	a := novaEmpresaDeTeste(t, pool, "CONFERENCIA A")
	b := novaEmpresaDeTeste(t, pool, "CONFERENCIA B")

	// Mesma convenção (enterprise_id dos dois lados), pela coluna item_code.
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code, business_code, warehouse_code, created_by, enterprise_id)
		VALUES ($1, ($1::bigint)::text, $1, $2, $3)`, item, ator, a.id)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM item_planning_extras WHERE item_code = $1`, item)
		testutil.Exec(t, pool, `DELETE FROM items WHERE code = $1`, item)
	})

	_, err := pool.Exec(ctx, `INSERT INTO item_planning_extras(item_code, enterprise_id) VALUES ($1, $2)`, item, b.id)
	exigeVinculoRecusado(t, err, "conferencia_empresa_item_planning_extras_item_code")

	if _, err := pool.Exec(ctx, `INSERT INTO item_planning_extras(item_code, enterprise_id) VALUES ($1, $2)`, item, a.id); err != nil {
		t.Fatalf("referência da própria empresa foi recusada: %v", err)
	}
	// Mover a linha para outra empresa também é cruzar a referência.
	_, err = pool.Exec(ctx, `UPDATE item_planning_extras SET enterprise_id = $2 WHERE item_code = $1`, item, b.id)
	exigeVinculoRecusado(t, err, "conferencia_empresa_item_planning_extras_item_code")
	// UPDATE que não mexe na referência nem na empresa segue livre.
	if _, err := pool.Exec(ctx, `UPDATE item_planning_extras SET coverage = 3 WHERE item_code = $1`, item); err != nil {
		t.Fatalf("UPDATE sem mudar a referência foi recusado: %v", err)
	}

	// Convenções diferentes: a divergência de recebimento é da empresa pelo
	// CÓDIGO (enterprise_code) e o item pelo id. Com id ≠ código — como em toda
	// empresa criada depois da primeira —, comparar as colunas direto erraria.
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM receiving_divergences WHERE item_code = $1`, item) })
	divergencia := `INSERT INTO receiving_divergences(divergence_type, enterprise_code, item_code) VALUES ('OTHER', $1, $2)`
	_, err = pool.Exec(ctx, divergencia, b.code, item)
	exigeVinculoRecusado(t, err, "conferencia_empresa_receiving_divergences_item_code")
	if _, err := pool.Exec(ctx, divergencia, a.code, item); err != nil {
		t.Fatalf("divergência da empresa do item foi recusada: %v", err)
	}
}

// O fluxo interfábricas é cruzado de propósito e fica fora da conferência.
func TestConferenciaDeEmpresaRespeitaExcecoes(t *testing.T) {
	pool := testutil.Pool(t)
	var excecoes []string
	rows, err := pool.Query(context.Background(),
		`SELECT DISTINCT filho FROM conferencia_empresa_vinculos()
		  WHERE filho LIKE 'mrp\_%' OR filho LIKE 'planning\_%' OR filho LIKE '%\_enterprises'
		     OR filho IN ('planned_orders', 'production_plan_inter_factories')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			t.Fatal(err)
		}
		excecoes = append(excecoes, f)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(excecoes) > 0 {
		t.Fatalf("tabelas de exceção entraram na conferência: %v", excecoes)
	}
}

// Colunas de código sem FK (migração 000403): o código pode repetir entre
// empresas, então vale "existe na minha empresa", não "o pai é da minha".
func TestConferenciaDeEmpresaPorCodigo(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	ator := testutil.Actor(t, pool)
	a := novaEmpresaDeTeste(t, pool, "POR CODIGO A")
	b := novaEmpresaDeTeste(t, pool, "POR CODIGO B")

	var pedidoA int64
	if err := pool.QueryRow(ctx, `INSERT INTO sales_orders(order_number, enterprise_code, created_by)
		VALUES (1, $1, $2) RETURNING code`, a.code, ator).Scan(&pedidoA); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id = ANY($1)`, []int64{a.id, b.id})
		testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE enterprise_code = ANY($1)`, []int64{a.code, b.code})
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, []int64{a.id, b.id})
		testutil.Exec(t, pool, `DELETE FROM notification_outbox WHERE enterprise_id = ANY($1)`, []int64{a.id, b.id})
	})
	nota := `INSERT INTO fiscal_exits(numero_nf, data_emissao, cfop, natureza_operacao, valor_total, sales_order_code, customer_code, status, created_by, enterprise_id)
		VALUES (1, $5::date, '5102', 'Venda', 10, $1, $2, 'DRAFT', $3, $4)`
	hoje := time.Now()

	// Pedido de código único no banco: o da A não serve para a nota da B.
	_, err := pool.Exec(ctx, nota, pedidoA, nil, ator, b.id, hoje)
	exigeVinculoRecusado(t, err, "conferencia_empresa_fiscal_exits_sales_order_code")
	if _, err := pool.Exec(ctx, nota, pedidoA, nil, ator, a.id, hoje); err != nil {
		t.Fatalf("nota da empresa do pedido foi recusada: %v", err)
	}
	// Código que não existe em lugar nenhum segue livre (como sem FK).
	if _, err := pool.Exec(ctx, nota, int64(987_654_321_000), nil, ator, b.id, hoje); err != nil {
		t.Fatalf("pedido inexistente foi tratado como de outra empresa: %v", err)
	}

	// Cliente com o MESMO código nas duas empresas (cada uma tem a sua cópia).
	cliente := 1_800_000_000 + testutil.UniqueCode()%90_000_000
	testutil.Exec(t, pool, `INSERT INTO customers(code, name, document_number, created_by, enterprise_id) VALUES ($1, 'CLIENTE A', '11111111000191', $2, $3)`, cliente, ator, a.id)
	if _, err := pool.Exec(ctx, nota, nil, cliente, ator, b.id, hoje); err == nil {
		t.Fatal("B usou o cliente que só existe na A")
	} else {
		exigeVinculoRecusado(t, err, "conferencia_empresa_fiscal_exits_customer_code")
	}
	testutil.Exec(t, pool, `INSERT INTO customers(code, name, document_number, created_by, enterprise_id) VALUES ($1, 'CLIENTE B', '22222222000191', $2, $3)`, cliente, ator, b.id)
	if _, err := pool.Exec(ctx, nota, nil, cliente, ator, b.id, hoje); err != nil {
		t.Fatalf("B com a própria cópia do cliente foi recusada: %v", err)
	}
}
