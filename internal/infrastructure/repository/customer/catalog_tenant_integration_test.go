//go:build integration

package customer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	eoentity "github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	smtentity "github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	eorepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation"
	smtrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock_movement"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestCatalogsAreIsolatedByTenant: os cadastros de apoio eram do banco
// inteiro. Cada empresa tem os seus, com o MESMO código podendo existir nas
// duas; nada se lê ou altera da outra, e o banco recusa registro apontando
// para catálogo de outra empresa.
func TestCatalogsAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		for _, q := range []string{
			`DELETE FROM customers WHERE enterprise_id = ANY($1)`,
			`DELETE FROM state_group_ufs WHERE enterprise_id = ANY($1)`,
			`DELETE FROM entry_operation_types WHERE enterprise_id = ANY($1)`,
			`DELETE FROM state_groups WHERE enterprise_id = ANY($1)`,
			`DELETE FROM stock_movement_types WHERE enterprise_id = ANY($1)`,
			`DELETE FROM regions WHERE enterprise_id = ANY($1)`,
			`DELETE FROM tax_types WHERE enterprise_id = ANY($1)`,
		} {
			if _, err := pool.Exec(base, q, ids); err != nil {
				t.Errorf("limpeza: %v", err)
			}
		}
	})
	repo := customerpg.New(sqlc.New(pool), pool)

	// Região: o mesmo código nas duas empresas, cada uma vendo a sua.
	regA, err := repo.CreateRegion(a.Ctx, &entity.Region{Code: 10, Description: "SUL A", UF: "PR", City: "CURITIBA", CreatedBy: uuid.UUID(actor)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetRegionByCode(b.Ctx, 10); err == nil {
		t.Fatal("empresa B leu a região da A")
	}
	if l, err := repo.ListRegions(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d região(ões) da A (%v)", len(l), err)
	}
	regA.Description = "ALTERADA POR B"
	if _, err := repo.UpdateRegion(b.Ctx, regA); err == nil {
		t.Fatal("empresa B alterou a região da A")
	}
	if _, err := repo.CreateRegion(b.Ctx, &entity.Region{Code: 10, Description: "SUL B", UF: "SC", City: "JOINVILLE", CreatedBy: uuid.UUID(actor)}); err != nil {
		t.Fatalf("o mesmo código de região tem de poder existir na empresa B: %v", err)
	}
	if got, err := repo.GetRegionByCode(a.Ctx, 10); err != nil || got.Description != "SUL A" {
		t.Fatalf("a região da A mudou: %+v (%v)", got, err)
	}

	// Tributação: alterar grava (antes era um stub) e só na própria empresa.
	tt, err := repo.CreateTaxType(a.Ctx, &entity.TaxType{Code: 5, Description: "TRIB A"})
	if err != nil {
		t.Fatal(err)
	}
	tt.Description = "TRIB A ALTERADA"
	if _, err := repo.UpdateTaxType(b.Ctx, tt); err == nil {
		t.Fatal("empresa B alterou a tributação da A")
	}
	if got, err := repo.UpdateTaxType(a.Ctx, tt); err != nil || got.Description != "TRIB A ALTERADA" {
		t.Fatalf("alterar a tributação não gravou: %+v (%v)", got, err)
	}

	// Garantia no banco: cliente da B não aponta para região da A.
	if _, err := pool.Exec(base, `INSERT INTO customers (code, name, document_number, enterprise_id, created_by, region_id) VALUES (77,'X','11222333000181',$1,$2,$3)`,
		b.ID, actor, regA.ID); err == nil {
		t.Fatal("o banco aceitou cliente da B com região da A")
	}
	if _, err := pool.Exec(base, `INSERT INTO customers (code, name, document_number, enterprise_id, created_by, region_id) VALUES (77,'X','11222333000181',$1,$2,$3)`,
		a.ID, actor, regA.ID); err != nil {
		t.Fatalf("cliente da A com a própria região foi recusado: %v", err)
	}

	// TES e grupo de UF.
	eo := eorepo.New(sqlc.New(pool), pool)
	if _, err := eo.CreateStateGroup(a.Ctx, &eoentity.StateGroup{Code: 3, Description: "SUL", CreatedBy: uuid.UUID(actor)}); err != nil {
		t.Fatal(err)
	}
	if err := eo.AddStateGroupUF(b.Ctx, 3, "PR"); err == nil {
		t.Fatal("empresa B incluiu UF no grupo da A")
	}
	if err := eo.AddStateGroupUF(a.Ctx, 3, "PR"); err != nil {
		t.Fatal(err)
	}
	if ok, err := eo.UFInGroup(b.Ctx, 3, "PR"); err != nil || ok {
		t.Fatalf("empresa B enxergou o grupo de UF da A (%v)", err)
	}
	if _, err := eo.CreateEntryOperation(a.Ctx, &eoentity.EntryOperationType{Code: 1102, Description: "COMPRA A", NatureOperation: "COMPRA", CreatedBy: uuid.UUID(actor)}); err != nil {
		t.Fatal(err)
	}
	if _, err := eo.GetEntryOperationByCode(b.Ctx, 1102); err == nil {
		t.Fatal("empresa B leu a TES da A")
	}
	if l, err := eo.ListEntryOperations(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d TES da A (%v)", len(l), err)
	}

	// Tipo de movimento de estoque.
	smt := smtrepo.New(pool)
	tipo, err := smt.Create(a.Ctx, &smtentity.StockMovementType{Sigla: "ENT", Description: "ENTRADA A", UsageType: smtentity.UsageGeral, EntryExit: smtentity.DirEntrada})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := smt.GetBySigla(b.Ctx, "ENT"); err == nil {
		t.Fatal("empresa B leu o tipo de movimento da A")
	}
	if _, err := smt.GetByID(b.Ctx, tipo.ID); err == nil {
		t.Fatal("empresa B leu o tipo de movimento da A pelo id")
	}
	if _, err := smt.Update(b.Ctx, tipo); err == nil {
		t.Fatal("empresa B alterou o tipo de movimento da A")
	}
}
