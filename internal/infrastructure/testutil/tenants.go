//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Empresa é uma empresa criada para o teste, com o contexto autenticado nela.
type Empresa struct {
	ID   int64
	Code int64
	Ctx  context.Context
}

// DuasEmpresas cria duas empresas descartáveis para testes de isolamento.
// O código é bem diferente do id de propósito: na empresa 1 os dois valem 1 e
// a troca de tenant.ID por tenant.Code passa despercebida. `limpar` roda antes
// de apagar as empresas, para o teste remover o que criou nelas.
func DuasEmpresas(t *testing.T, pool *pgxpool.Pool, limpar func(a, b Empresa)) (Empresa, Empresa) {
	t.Helper()
	base := context.Background()
	actor := Actor(t, pool).String()
	nova := func(code int64, nome string) Empresa {
		e := Empresa{Code: code}
		if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id`, code, nome).Scan(&e.ID); err != nil {
			t.Fatal(err)
		}
		e.Ctx = context.WithValue(base, contextkey.UserKey, &security.AuthUser{ID: actor, Role: "ADMIN", EnterpriseID: e.ID, EnterpriseCode: e.Code})
		return e
	}
	code := int64(1_680_000_000 + UniqueCode()%10_000_000)
	a := nova(code, "Isolamento A")
	b := nova(code+1, "Isolamento B")
	t.Cleanup(func() {
		if limpar != nil {
			limpar(a, b)
		}
		_, _ = pool.Exec(base, `DELETE FROM user_enterprises WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id = ANY($1)`, []int64{a.ID, b.ID})
	})
	return a, b
}

// DadoLegadoCruzado grava em `tabela` uma linha que aponta para registro de
// OUTRA empresa — o que a conferência de empresa (migração 000402) recusa
// desde então, mas que dado antigo ainda pode ter. Serve aos testes que provam
// que uma consulta não vaza esse legado.
//
// O gatilho é desligado e religado dentro da mesma transação: DDL no Postgres
// é transacional, então nenhuma outra sessão o vê desligado.
func DadoLegadoCruzado(t *testing.T, pool *pgxpool.Pool, tabela, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ident := pgx.Identifier{tabela}.Sanitize()
	if _, err := tx.Exec(ctx, "ALTER TABLE "+ident+" DISABLE TRIGGER zz_conferir_empresa_ref"); err != nil {
		t.Fatalf("desligando a conferência de empresa de %s: %v", tabela, err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("gravando dado legado em %s: %v", tabela, err)
	}
	if _, err := tx.Exec(ctx, "ALTER TABLE "+ident+" ENABLE TRIGGER zz_conferir_empresa_ref"); err != nil {
		t.Fatalf("religando a conferência de empresa de %s: %v", tabela, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
