//go:build integration

package technical_assistance_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
)

func TestRMAIsTransactionalIdempotentAndTenantSafe(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	enterpriseCodeA := int64(1_500_000_000 + testutil.UniqueCode()%100_000_000)
	enterpriseCodeB := enterpriseCodeA + 1
	var enterpriseA, enterpriseB, callCode, itemCode int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'RMA A') RETURNING id`, enterpriseCodeA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'RMA B') RETURNING id`, enterpriseCodeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'RMA Evidence',$2,'x')`, actor, actor.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO technical_assistance_calls(call_number,enterprise_code,customer_code,subject,created_by) VALUES(1,$1,10,'Teste RMA',$2) RETURNING code`, enterpriseCodeA, actor).Scan(&callCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO technical_assistance_call_items(call_code,sequence,item_code,quantity) VALUES($1,1,99,1) RETURNING code`, callCode).Scan(&itemCode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM technical_assistance_rma_evidences WHERE enterprise_id=$1`, enterpriseA)
		_, _ = pool.Exec(ctx, `DELETE FROM technical_assistance_rma_events WHERE enterprise_id=$1`, enterpriseA)
		_, _ = pool.Exec(ctx, `DELETE FROM technical_assistance_rmas WHERE enterprise_id=$1`, enterpriseA)
		_, _ = pool.Exec(ctx, `DELETE FROM technical_assistance_calls WHERE code=$1`, callCode)
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{enterpriseA, enterpriseB})
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, actor)
	})
	repo := repository.New(pool)
	rma := &entity.RMA{CallCode: callCode, Status: "SOLICITADO", ReasonCode: "DEFEITO", EligibilityStatus: "ELEGIVEL", EligibilityReason: "garantia vigente", SLADueAt: time.Now().Add(48 * time.Hour), IdempotencyKey: "integration-rma", CreatedBy: actor, Items: []*entity.RMAItem{{CallItemCode: itemCode, ItemCode: 99, Quantity: 1}}}
	created, err := repo.CreateRMA(ctx, enterpriseA, rma)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := repo.CreateRMA(ctx, enterpriseA, rma)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Code != created.Code {
		t.Fatalf("idempotência criou códigos distintos: %d/%d", created.Code, repeated.Code)
	}
	if _, err = repo.GetRMA(ctx, enterpriseB, created.Code); err == nil {
		t.Fatal("tenant B consultou RMA do tenant A")
	}
	loaded, err := repo.GetRMA(ctx, enterpriseA, created.Code)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Items) != 1 || len(loaded.Events) != 1 {
		t.Fatalf("rastreabilidade incompleta: itens=%d eventos=%d", len(loaded.Items), len(loaded.Events))
	}
	evidence, err := repo.CreateRMAEvidence(ctx, enterpriseA, &entity.RMAEvidence{RMACode: created.Code, FileName: "falha.pdf", ContentType: "application/pdf", Content: []byte("%PDF-1.4 evidence"), SizeBytes: 17, SHA256: "evidence-hash", UploadedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetRMAEvidence(ctx, enterpriseB, created.Code, evidence.ID); err == nil {
		t.Fatal("tenant B baixou evidência do RMA do tenant A")
	}
	listed, err := repo.ListRMAEvidences(ctx, enterpriseA, created.Code)
	if err != nil || len(listed) != 1 || listed[0].FileName != "falha.pdf" {
		t.Fatalf("pesquisa de evidências inválida: rows=%d err=%v", len(listed), err)
	}
}
