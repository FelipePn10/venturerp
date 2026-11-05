//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NovoAlmoxarifado cria um almoxarifado da empresa e o apaga no fim do teste. O
// estoque recusa movimento em almoxarifado que não é da empresa: teste não pode
// usar um id fixo.
func NovoAlmoxarifado(t *testing.T, pool *pgxpool.Pool, empresa int64) int64 {
	t.Helper()
	ator := Actor(t, pool)
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		 VALUES($1,$1,$2,'INTERNO','NORMAL',true,true,$3) RETURNING id`, "T-"+uuid.NewString()[:10], ator, empresa).Scan(&id); err != nil {
		t.Fatalf("criando o almoxarifado de teste: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM warehouse WHERE id=$1`, id) })
	return id
}

// NovosItens garante os códigos como itens da empresa, apagados no fim do teste.
func NovosItens(t *testing.T, pool *pgxpool.Pool, empresa int64, ator uuid.UUID, codigos ...int64) {
	t.Helper()
	for _, c := range codigos {
		codigo := c
		Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3) ON CONFLICT DO NOTHING`, codigo, ator, empresa)
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE code=$1`, codigo) })
	}
}
