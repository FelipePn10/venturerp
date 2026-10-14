//go:build integration

package technical_assistance_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	csentity "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/consumer_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestServiceReferencesAreIsolatedByTenant: responsáveis de garantia não tinham
// empresa, e os atendimentos (assistência técnica e SAC) aceitavam cliente,
// responsável, consumidor e pedido de outra empresa.
func TestServiceReferencesAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	cliente := int64(1_770_000_000 + testutil.UniqueCode()%10_000_000)
	consumidor := cliente
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM technical_assistance_calls WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		_, _ = pool.Exec(base, `DELETE FROM technical_assistance_warranty_responsibles WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		_, _ = pool.Exec(base, `DELETE FROM consumer_service_consumer_enterprises WHERE consumer_code=$1`, consumidor)
		_, _ = pool.Exec(base, `DELETE FROM consumer_service_consumers WHERE code=$1`, consumidor)
		_, _ = pool.Exec(base, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	for _, e := range []testutil.Empresa{a, b} {
		if _, err := pool.Exec(base, `INSERT INTO customers (code, name, document_number, enterprise_id, created_by) VALUES ($1,'CLIENTE',$2,$3,$4)`,
			cliente, "11222333000181", e.ID, actor); err != nil {
			t.Fatal(err)
		}
	}
	ta := technical_assistance.New(pool)
	resp, err := ta.CreateWarrantyResponsible(a.Ctx, &entity.WarrantyResponsible{Name: "RESP A", CustomerCode: &cliente, CreatedBy: uuid.UUID(actor)})
	if err != nil {
		t.Fatal(err)
	}
	if l, err := ta.ListWarrantyResponsibles(b.Ctx, false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d responsável(eis) da A (%v)", len(l), err)
	}
	if _, err := ta.CreateCall(b.Ctx, b.ID, &entity.Call{CallNumber: 1, EnterpriseCode: b.Code, CustomerCode: cliente, WarrantyResponsibleCode: &resp.Code,
		Status: "PENDING", Priority: "NORMAL", OpenedAt: time.Now(), Subject: "x", CreatedBy: uuid.UUID(actor)}); err == nil {
		t.Fatal("o atendimento da B aceitou o responsável de garantia da A")
	}

	// SAC: consumidor da A não serve para atendimento da B.
	if _, err := pool.Exec(base, `INSERT INTO consumer_service_consumers (code, name, created_by) VALUES ($1,'CONSUMIDOR A',$2)`, consumidor, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(base, `INSERT INTO consumer_service_consumer_enterprises (consumer_code, enterprise_id) VALUES ($1,$2)`, consumidor, a.ID); err != nil {
		t.Fatal(err)
	}
	sac := consumer_service.New(pool)
	if _, err := sac.CreateCall(b.Ctx, b.ID, &csentity.Call{CallNumber: 1, ConsumerCode: consumidor, CallTypeCode: 1, Subject: "x", CreatedBy: uuid.UUID(actor)}); err == nil {
		t.Fatal("o SAC da B abriu atendimento para o consumidor da A")
	}
}
