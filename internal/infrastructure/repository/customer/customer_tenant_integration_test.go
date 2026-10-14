//go:build integration

package customer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// TestCustomersAreIsolatedByTenant: a "sonda cruzada" que achou o defeito —
// GET /api/customers/{code} devolvia o mesmo cadastro para duas empresas.
// Cada empresa tem o seu cadastro; o mesmo código pode existir nas duas, cada
// uma enxergando só o seu; nada se lê, altera, bloqueia ou apaga de outra.
func TestCustomersAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	codeA := int64(1_670_000_000 + testutil.UniqueCode()%10_000_000)
	codeB := codeA + 1
	var enterpriseA, enterpriseB int64
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Clientes A') RETURNING id`, codeA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Clientes B') RETURNING id`, codeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, []int64{enterpriseA, enterpriseB})
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id = ANY($1)`, []int64{enterpriseA, enterpriseB})
	})
	repo := customerpg.New(sqlc.New(pool), pool)
	ctxA := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseA, EnterpriseCode: codeA})
	ctxB := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseB, EnterpriseCode: codeB})

	code, err := repo.NextCustomerCode(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	novo := func(nome string) *entity.Customer {
		return &entity.Customer{Code: code, Name: nome, DocumentType: "CNPJ", DocumentNumber: "11222333000181",
			PaymentCondVisibility: "TODOS", IsActive: true, CreatedBy: uuid.UUID(actor)}
	}
	a, err := repo.CreateCustomer(ctxA, novo("CLIENTE DA EMPRESA A"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetCustomerByCode(ctxB, code); err == nil {
		t.Fatal("empresa B leu o cliente da empresa A")
	}
	if _, err := repo.GetCustomerByDocument(ctxB, "11222333000181"); err == nil {
		t.Fatal("empresa B achou o cliente da empresa A pelo CNPJ")
	}
	if lista, _ := repo.ListCustomers(ctxB, false); len(lista) != 0 {
		t.Fatalf("empresa B listou %d cliente(s) da A", len(lista))
	}
	if err := repo.BlockCustomer(ctxB, code, "tentativa cruzada"); err == nil {
		t.Fatal("empresa B bloqueou o cliente da A")
	}
	a.Name = "ALTERADO POR B"
	if _, err := repo.UpdateCustomer(ctxB, a); err == nil {
		t.Fatal("empresa B alterou o cliente da A")
	}
	if _, err := repo.AddAddress(ctxB, &entity.CustomerAddress{CustomerID: a.ID, AddressType: "COBRANCA", Country: "BR"}); err == nil {
		t.Fatal("empresa B incluiu endereço no cliente da A")
	}
	end, err := repo.AddAddress(ctxA, &entity.CustomerAddress{CustomerID: a.ID, AddressType: "COBRANCA", Country: "BR"})
	if err != nil {
		t.Fatal(err)
	}
	if l, _ := repo.ListAddresses(ctxB, a.ID); len(l) != 0 {
		t.Fatal("empresa B leu endereço do cliente da A")
	}
	if err := repo.DeleteAddress(ctxB, end.ID); err == nil {
		t.Fatal("empresa B apagou endereço do cliente da A")
	}
	contato, err := repo.AddContact(ctxA, &entity.CustomerContact{CustomerID: a.ID, Name: "Ana"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteContact(ctxB, contato.ID); err == nil {
		t.Fatal("empresa B desativou contato do cliente da A")
	}
	contato.Name = "Ana Maria"
	if got, err := repo.UpdateContact(ctxA, contato); err != nil || got.Name != "Ana Maria" {
		t.Fatalf("alterar contato não gravou: %+v %v", got, err)
	}
	end.City = strPtr("Curitiba")
	if got, err := repo.UpdateAddress(ctxA, end); err != nil || got.City == nil || *got.City != "Curitiba" {
		t.Fatalf("alterar endereço não gravou: %+v %v", got, err)
	}

	// O mesmo código na empresa B é OUTRO cadastro.
	b, err := repo.CreateCustomer(ctxB, novo("CLIENTE DA EMPRESA B"))
	if err != nil {
		t.Fatalf("o mesmo código deveria poder existir na empresa B: %v", err)
	}
	if b.ID == a.ID {
		t.Fatal("as empresas compartilham o registro")
	}
	gotA, err := repo.GetCustomerByCode(ctxA, code)
	if err != nil || gotA.Name != "CLIENTE DA EMPRESA A" {
		t.Fatalf("empresa A: %+v %v", gotA, err)
	}
	gotB, err := repo.GetCustomerByCode(ctxB, code)
	if err != nil || gotB.Name != "CLIENTE DA EMPRESA B" {
		t.Fatalf("empresa B: %+v %v", gotB, err)
	}
	if l, _ := repo.ListAddresses(ctxA, a.ID); len(l) != 1 {
		t.Fatalf("endereço da A sumiu: %d", len(l))
	}
	if _, err := repo.CreateCustomer(ctxA, novo("DUPLICADO NA A")); err == nil || !strings.Contains(err.Error(), "já existe cliente com o código") {
		t.Fatalf("código repetido dentro da mesma empresa deveria ser recusado com mensagem clara: %v", err)
	}
}

func strPtr(s string) *string { return &s }
