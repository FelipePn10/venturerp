//go:build integration

package structure_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestStructureIsIsolatedByTenant: a estrutura (BOM) é da empresa do item pai.
// Antes, a criação só conferia se o item existia no banco: a empresa B criava
// componente na estrutura do item da A e lia/alterava/removia a dela.
func TestStructureIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	pai := int64(1_750_000_000 + testutil.UniqueCode()%10_000_000)
	filho := pai + 1
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM item_structure_history WHERE parent_code=$1`, pai)
		_, _ = pool.Exec(base, `DELETE FROM item_structures WHERE parent_code=$1`, pai)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code = ANY($1)`, []int64{pai, filho, filho + 1})
	})
	actor := testutil.Actor(t, pool)
	testutil.SeedItem(t, pool, a.Ctx, pai, actor)
	testutil.SeedItem(t, pool, a.Ctx, filho, actor)
	testutil.SeedItem(t, pool, b.Ctx, filho+1, actor)

	q := sqlc.New(pool)
	repo := structure.NewItemStructureRepository(q).WithHistory(pool)
	comp := &entity.ItemStructure{ParentCode: pai, ChildCode: filho, Quantity: 2, UnitOfMeasurement: "UN", Health: "ATIVO", Sequence: 10, CreatedBy: actor}

	if _, err := repo.Create(b.Ctx, comp); err == nil {
		t.Fatal("empresa B criou componente na estrutura do item da A")
	}
	if ok, err := repo.ItemExists(b.Ctx, pai); err != nil || ok {
		t.Fatalf("empresa B enxergou o item da A (%v)", err)
	}
	if _, err := repo.Create(a.Ctx, &entity.ItemStructure{ParentCode: pai, ChildCode: filho + 1, Quantity: 1, UnitOfMeasurement: "UN", Health: "ATIVO", Sequence: 20, CreatedBy: actor}); err == nil {
		t.Fatal("empresa A pôs item da B na própria estrutura")
	}
	criado, err := repo.Create(a.Ctx, comp)
	if err != nil {
		t.Fatalf("empresa A não criou o componente: %v", err)
	}
	if l, err := repo.GetAllDirectChildren(b.Ctx, pai); err != nil || len(l) != 0 {
		t.Fatalf("empresa B leu %d componente(s) da A (%v)", len(l), err)
	}
	if _, err := repo.GetByID(b.Ctx, criado.ID); err == nil {
		t.Fatal("empresa B leu o componente da A pelo id")
	}
	comp.Quantity = 99
	if _, err := repo.Update(b.Ctx, comp); err == nil {
		t.Fatal("empresa B alterou a estrutura da A")
	}
	if err := repo.Delete(b.Ctx, criado.ID); err == nil {
		t.Fatal("empresa B removeu componente da A")
	}
	if err := repo.DeleteByCodes(b.Ctx, pai, filho, nil); err == nil {
		t.Fatal("empresa B removeu componente da A pelos códigos")
	}
	if h, err := repo.ListByParent(b.Ctx, pai, 50); err != nil || len(h) != 0 {
		t.Fatalf("empresa B leu %d registro(s) do histórico da A (%v)", len(h), err)
	}
	consulta := structure_query.NewStructureQueryRepository(q)
	if l, err := consulta.GetWhereUsed(b.Ctx, filho, 0); err != nil || len(l) != 0 {
		t.Fatalf("empresa B viu onde o item da A é usado: %d (%v)", len(l), err)
	}
	if l, err := consulta.ConsultChildren(b.Ctx, pai, "", nil); err != nil || len(l) != 0 {
		t.Fatalf("empresa B consultou a estrutura da A: %d (%v)", len(l), err)
	}
	if l, err := repo.GetAllDirectChildren(a.Ctx, pai); err != nil || len(l) != 1 || l[0].Quantity != 2 {
		t.Fatalf("a estrutura da A mudou: %+v (%v)", l, err)
	}
	if l, err := consulta.GetWhereUsed(a.Ctx, filho, 0); err != nil || len(l) != 1 {
		t.Fatalf("empresa A não achou onde o próprio item é usado: %d (%v)", len(l), err)
	}
}
