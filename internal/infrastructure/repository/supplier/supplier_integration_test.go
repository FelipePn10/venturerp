//go:build integration

package supplier_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func newSupplierRepo(t *testing.T) (domainrepo.SupplierRepository, func(code int64)) {
	q, pool := testutil.Queries(t)
	repo := supplierrepo.New(q, pool)
	cleanup := func(code int64) {
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE code = $1", code)
	}
	return repo, cleanup
}

func makeSupplier(t *testing.T, code int64, doc string) *entity.Supplier {
	t.Helper()
	ie := "1234567890"
	// DocumentEstrangeiro bypasses CNPJ/CPF check-digit validation so each test can
	// use a unique document and avoid cross-package collisions on GetByDocument.
	s, err := entity.NewSupplier(code, entity.SupplierInput{
		Name:              "Fornecedor Integração",
		PersonType:        entity.PersonJuridica,
		DocumentType:      entity.DocumentEstrangeiro,
		DocumentNumber:    doc,
		StateRegistration: &ie,
	}, uuid.New())
	if err != nil {
		t.Fatalf("NewSupplier: %v", err)
	}
	return s
}

func TestIntegration_Supplier_CRUD(t *testing.T) {
	repo, cleanup := newSupplierRepo(t)
	ctx := context.Background()
	code := testutil.UniqueCode()
	doc := fmt.Sprintf("EXT%d", code)
	defer cleanup(code)

	created, err := repo.CreateSupplier(ctx, makeSupplier(t, code, doc))
	if err != nil {
		t.Fatalf("CreateSupplier: %v", err)
	}
	if created.ID == 0 || created.Code != code {
		t.Fatalf("unexpected created supplier: %+v", created)
	}

	got, err := repo.GetSupplierByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetSupplierByCode: %v", err)
	}
	if got.Name != "Fornecedor Integração" || !got.IsActive {
		t.Errorf("unexpected supplier read: %+v", got)
	}

	byDoc, err := repo.GetSupplierByDocument(ctx, doc)
	if err != nil {
		t.Fatalf("GetSupplierByDocument: %v", err)
	}
	if byDoc.Code != code {
		t.Errorf("GetSupplierByDocument returned code %d, want %d", byDoc.Code, code)
	}

	// Folders: address + phone.
	city := "Curitiba"
	uf := "PR"
	if _, err := repo.AddAddress(ctx, &entity.SupplierAddress{
		SupplierID: created.ID, AddressType: entity.AddressComercial,
		City: &city, UF: &uf, Country: "Brasil", IsDefault: true,
	}); err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	addrs, err := repo.ListAddresses(ctx, created.ID)
	if err != nil || len(addrs) != 1 || addrs[0].UF == nil || *addrs[0].UF != "PR" {
		t.Fatalf("ListAddresses unexpected: %+v err=%v", addrs, err)
	}
	if _, err := repo.AddPhone(ctx, &entity.SupplierPhone{SupplierID: created.ID, Number: "4133221100", Ranking: 1}); err != nil {
		t.Fatalf("AddPhone: %v", err)
	}
	if ph, err := repo.ListPhones(ctx, created.ID); err != nil || len(ph) != 1 {
		t.Fatalf("ListPhones unexpected: %+v err=%v", ph, err)
	}

	// SEFAZ snapshot.
	if err := repo.UpdateSefazSnapshot(ctx, code, "LIBERADO", "tester"); err != nil {
		t.Fatalf("UpdateSefazSnapshot: %v", err)
	}
	got, _ = repo.GetSupplierByCode(ctx, code)
	if got.BillingReceiptStatus == nil || *got.BillingReceiptStatus != "LIBERADO" {
		t.Errorf("SEFAZ snapshot not persisted: %+v", got.BillingReceiptStatus)
	}

	// Block / unblock.
	if err := repo.BlockSupplier(ctx, code, "inadimplência"); err != nil {
		t.Fatalf("BlockSupplier: %v", err)
	}
	got, _ = repo.GetSupplierByCode(ctx, code)
	if !got.Blocked {
		t.Error("expected supplier blocked")
	}
	if err := repo.UnblockSupplier(ctx, code); err != nil {
		t.Fatalf("UnblockSupplier: %v", err)
	}
	got, _ = repo.GetSupplierByCode(ctx, code)
	if got.Blocked {
		t.Error("expected supplier unblocked")
	}

	// Delete (no purchase orders reference it → should succeed).
	if err := repo.DeleteSupplier(ctx, code); err != nil {
		t.Fatalf("DeleteSupplier: %v", err)
	}
	if _, err := repo.GetSupplierByCode(ctx, code); err == nil {
		t.Error("expected supplier to be gone after delete")
	}
}

func TestIntegration_Supplier_PropagateStateRegistration(t *testing.T) {
	repo, cleanup := newSupplierRepo(t)
	ctx := context.Background()
	c1 := testutil.UniqueCode()
	c2 := testutil.UniqueCode() + 1
	doc := fmt.Sprintf("EXT%d", c1) // shared, unique to this test
	defer cleanup(c1)
	defer cleanup(c2)

	// Two establishments sharing the same document.
	if _, err := repo.CreateSupplier(ctx, makeSupplier(t, c1, doc)); err != nil {
		t.Fatalf("create c1: %v", err)
	}
	if _, err := repo.CreateSupplier(ctx, makeSupplier(t, c2, doc)); err != nil {
		t.Fatalf("create c2: %v", err)
	}

	newIE := "9998887776"
	if err := repo.PropagateStateRegistration(ctx, doc, &newIE, c1); err != nil {
		t.Fatalf("PropagateStateRegistration: %v", err)
	}
	// c2 (same doc, not the excepted code) must receive the new IE.
	got2, _ := repo.GetSupplierByCode(ctx, c2)
	if got2.StateRegistration == nil || *got2.StateRegistration != newIE {
		t.Errorf("c2 IE not propagated: %v", got2.StateRegistration)
	}
}
