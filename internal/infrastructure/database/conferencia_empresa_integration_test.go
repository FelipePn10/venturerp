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
