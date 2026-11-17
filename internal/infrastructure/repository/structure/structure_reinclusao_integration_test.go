//go:build integration

package structure_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestComponenteExcluidoPodeVoltarParaAEstrutura: a exclusão desativa a linha
// (is_active = FALSE) e os índices únicos contavam a linha inativa, então o
// componente excluído nunca mais voltava para o mesmo pai — a tela respondia
// "este componente já está na estrutura deste item" sem mostrá-lo
// (Tecnofer, TP-0120/50-E no BU-120/50, 17/11/2026).
func TestComponenteExcluidoPodeVoltarParaAEstrutura(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	pai := int64(1_760_000_000 + testutil.UniqueCode()%10_000_000)
	filho := pai + 1
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM item_structure_history WHERE parent_code=$1`, pai)
		_, _ = pool.Exec(base, `DELETE FROM item_structures WHERE parent_code=$1`, pai)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code = ANY($1)`, []int64{pai, filho})
	})
	actor := testutil.Actor(t, pool)
	testutil.SeedItem(t, pool, a.Ctx, pai, actor)
	testutil.SeedItem(t, pool, a.Ctx, filho, actor)

	repo := structure.NewItemStructureRepository(sqlc.New(pool)).WithHistory(pool)
	mascara := "AZUL"
	for _, caso := range []struct {
		nome    string
		mascara *string
		seq     int
	}{
		{"sem máscara", nil, 10},
		{"com máscara", &mascara, 20},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			novo := func(qtd float64) *entity.ItemStructure {
				return &entity.ItemStructure{ParentCode: pai, ChildCode: filho, ParentMask: caso.mascara,
					Quantity: decimal.NewFromFloat(qtd), UnitOfMeasurement: "UN", Health: "ATIVO",
					Sequence: caso.seq, CreatedBy: actor}
			}
			if _, err := repo.Create(a.Ctx, novo(1)); err != nil {
				t.Fatalf("inclusão inicial: %v", err)
			}
			// A duplicata ATIVA continua proibida.
			if _, err := repo.Create(a.Ctx, novo(1)); err == nil {
				t.Fatal("o mesmo componente entrou duas vezes ativo")
			}
			if err := repo.DeleteByCodes(a.Ctx, pai, filho, caso.mascara); err != nil {
				t.Fatalf("exclusão: %v", err)
			}
			// A empresa B continua sem acesso à estrutura da A.
			if _, err := repo.Create(b.Ctx, novo(5)); err == nil {
				t.Fatal("empresa B reincluiu componente na estrutura da A")
			}
			if _, err := repo.Create(a.Ctx, novo(3)); err != nil {
				t.Fatalf("o componente excluído não pôde voltar: %v", err)
			}
			var ativas, total int
			if err := pool.QueryRow(base, `
				SELECT count(*) FILTER (WHERE is_active), count(*) FROM item_structures
				 WHERE parent_code=$1 AND child_code=$2 AND parent_mask IS NOT DISTINCT FROM $3`,
				pai, filho, caso.mascara).Scan(&ativas, &total); err != nil {
				t.Fatal(err)
			}
			if ativas != 1 || total != 2 {
				t.Fatalf("esperava 1 ativa (a nova) e a inativa preservada: ativas=%d total=%d", ativas, total)
			}
		})
	}

	// A tela lista só as ativas: um componente por máscara, com a quantidade nova.
	l, err := repo.GetAllDirectChildren(a.Ctx, pai)
	if err != nil || len(l) != 2 {
		t.Fatalf("estrutura listada: %d linha(s) (%v)", len(l), err)
	}
	for _, c := range l {
		if !c.Quantity.Equal(decimal.NewFromFloat(3)) {
			t.Fatalf("listou a linha antiga: %+v", c)
		}
	}
	if l, err := repo.GetAllDirectChildren(b.Ctx, pai); err != nil || len(l) != 0 {
		t.Fatalf("empresa B leu %d componente(s) da A (%v)", len(l), err)
	}
}
