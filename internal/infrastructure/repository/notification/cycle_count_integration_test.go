//go:build integration

package notification_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/notification_uc"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	httphandler "github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestCycleCountUsesBusinessCodeAndWritesAuditTransactionally(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	actor := uuid.New()
	code := int64(1_500_000_000 + testutil.UniqueCode()%500_000_000)
	_, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Operador contagem',$2,'x')`, actor, actor.String()+"@test.local")
	if err != nil {
		t.Fatal(err)
	}
	var enterpriseID, warehouseID int64
	if err = pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Tenant contagem',$2) RETURNING id`, code, actor).Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed) VALUES($1,'Contagem',$2,'NORMAL','INTERNO',TRUE,TRUE) RETURNING id`, fmt.Sprintf("WC-%d", code), actor).Scan(&warehouseID); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER')`, actor, enterpriseID)
	if err != nil {
		t.Fatal(err)
	}
	legacy, legacyAlpha := testutil.UniqueCode(), testutil.UniqueCode()
	_, err = pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code) VALUES($1,$2,'Item contagem',$4,$5,'0007'),($1,$3,'Item alfa',$4,$5,'TEA452-0')`, warehouseID, legacy, legacyAlpha, actor, enterpriseID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO stock_balances(enterprise_id,item_code,warehouse_id,mask,quantity) VALUES($1,$2,$3,'',5)`, enterpriseID, legacy, warehouseID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_count_audit WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_counts WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_balances WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_enterprises WHERE enterprise_id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM warehouse WHERE id=$1`, warehouseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise WHERE id=$1`, enterpriseID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, actor)
	})
	repo := notificationrepo.New(pool)
	h := httphandler.NewNotificationHandler(notification_uc.New(repo))
	request := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		user := &security.AuthUser{ID: actor.String(), EnterpriseID: enterpriseID, Role: "USER"}
		return req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, user))
	}
	rec := httptest.NewRecorder()
	h.CreateCycleCount(rec, request(http.MethodPost, "/api/stock/cycle-counts", fmt.Sprintf(`{"warehouse_id":%d,"item_code":"0007","scheduled_for":%q}`, warehouseID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created notificationentity.CycleCount
	if err = json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ItemCode != "0007" || created.LegacyItemCode != legacy {
		t.Fatalf("referência incorreta: %+v", created)
	}
	if created.Origin != notificationentity.CycleOriginManual || created.PolicyDays != nil {
		t.Fatalf("origem manual incorreta: %+v", created)
	}
	rec = httptest.NewRecorder()
	h.CreateCycleCount(rec, request(http.MethodPost, "/api/stock/cycle-counts", fmt.Sprintf(`{"warehouse_id":%d,"item_code":"TEA452-0","scheduled_for":%q}`, warehouseID, time.Now().Add(2*time.Hour).UTC().Format(time.RFC3339))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar alfa HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	var createdAlpha notificationentity.CycleCount
	if err = json.Unmarshal(rec.Body.Bytes(), &createdAlpha); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	detailReq := request(http.MethodGet, "/api/stock/cycle-counts/"+createdAlpha.ID.String(), "")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", createdAlpha.ID.String())
	detailReq = detailReq.WithContext(context.WithValue(detailReq.Context(), chi.RouteCtxKey, rc))
	h.GetCycleCount(rec, detailReq)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"item_code":"TEA452-0"`) {
		t.Fatalf("detalhe HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ListCycleCounts(rec, request(http.MethodGet, "/api/stock/cycle-counts", ""))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"item_code":"0007"`) || !strings.Contains(rec.Body.String(), `"item_code":"TEA452-0"`) {
		t.Fatalf("lista HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err = repo.TransitionCycleCount(ctx, enterpriseID, createdAlpha.ID, actor, notificationentity.CycleCancelled, nil, "substituir por ocorrência automática de teste"); err != nil {
		t.Fatal(err)
	}
	policyDays := 9
	var automaticID uuid.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO stock_cycle_counts(enterprise_id,warehouse_id,item_code,scheduled_for,origin,policy_days) VALUES($1,$2,$3,NOW(),'POLITICA_ITEM',$4) RETURNING id`, enterpriseID, warehouseID, legacyAlpha, policyDays).Scan(&automaticID); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	automaticReq := request(http.MethodGet, "/api/stock/cycle-counts/"+automaticID.String(), "")
	automaticRC := chi.NewRouteContext()
	automaticRC.URLParams.Add("id", automaticID.String())
	automaticReq = automaticReq.WithContext(context.WithValue(automaticReq.Context(), chi.RouteCtxKey, automaticRC))
	h.GetCycleCount(rec, automaticReq)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"origin":"POLITICA_ITEM"`) || !strings.Contains(rec.Body.String(), `"policy_days":9`) || !strings.Contains(rec.Body.String(), `"item_code":"TEA452-0"`) {
		t.Fatalf("origem automática no detalhe HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	started, err := repo.TransitionCycleCount(ctx, enterpriseID, created.ID, actor, notificationentity.CycleCounting, nil, "início")
	if err != nil || started.State != notificationentity.CycleCounting {
		t.Fatalf("iniciar: state=%s err=%v", started.State, err)
	}
	counted := "5"
	completed, err := repo.TransitionCycleCount(ctx, enterpriseID, created.ID, actor, notificationentity.CycleCompleted, &counted, "conferido")
	if err != nil || completed.State != notificationentity.CycleCompleted {
		t.Fatalf("concluir: state=%s err=%v", completed.State, err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM stock_cycle_count_audit WHERE enterprise_id=$1 AND cycle_count_id=$2`, enterpriseID, created.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 3 {
		t.Fatalf("auditorias=%d, esperado=3", audits)
	}
}

func TestEligibleRecipientsAreTenantIsolatedAndExposeActivity(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	userA, userB := uuid.New(), uuid.New()
	for id, name := range map[uuid.UUID]string{userA: "Usuário A", userB: "Usuário B"} {
		if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password,is_active) VALUES($1,$2,$3,'x',$4)`, id, name, id.String()+"@test.local", id == userA); err != nil {
			t.Fatal(err)
		}
	}
	base := int64(1_500_000_000 + testutil.UniqueCode()%400_000_000)
	var enterpriseA, enterpriseB int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Tenant A',$2) RETURNING id`, base, userA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Tenant B',$2) RETURNING id`, base+1, userB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$3,'ADMIN'),($2,$4,'USER')`, userA, userB, enterpriseA, enterpriseB)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO enterprise_departments(enterprise_id,code,name,active) VALUES($1,'A','Departamento A',TRUE),($2,'B','Departamento B',FALSE)`, enterpriseA, enterpriseB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise_departments WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_enterprises WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise WHERE id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1,$2)`, userA, userB)
	})
	repo := notificationrepo.New(pool)
	users, err := repo.ListEligibleUsers(ctx, enterpriseA)
	if err != nil || len(users) != 1 || users[0].ID != userA || !users[0].Active {
		t.Fatalf("users A=%+v err=%v", users, err)
	}
	departments, err := repo.ListEligibleDepartments(ctx, enterpriseB)
	if err != nil || len(departments) != 1 || departments[0].Code != "B" || departments[0].Active {
		t.Fatalf("departments B=%+v err=%v", departments, err)
	}
}

func TestCatalogProducerStatusMatchesConnectedProducers(t *testing.T) {
	pool := testutil.Pool(t)
	rows, err := pool.Query(context.Background(), `SELECT event_key,producer_description FROM notification_event_catalog WHERE active AND producer_status='ATIVO'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	expected := map[string]bool{
		"COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO": true, "CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS": true, "MRP_EXCECAO": true,
		"FISCAL_NFE_SAIDA_CRIADA": true, "FISCAL_NFE_SAIDA_AGUARDANDO_AUTORIZACAO": true, "FISCAL_NFE_SAIDA_AUTORIZADA": true, "FISCAL_NFE_SAIDA_REJEITADA": true, "FISCAL_NFE_SAIDA_CANCELADA": true,
		"FISCAL_NFE_ENTRADA_IMPORTADA": true, "FISCAL_NFE_ENTRADA_APROVADA": true, "FISCAL_NFE_ENTRADA_ITEM_NAO_IDENTIFICADO": true, "FISCAL_NFE_ENTRADA_DIVERGENCIA_FISCAL": true, "FISCAL_NFE_ENTRADA_DIVERGENCIA_QUANTIDADE": true, "FISCAL_NFE_ENTRADA_CANCELADA": true,
		"ESTOQUE_CONTAGEM_PROXIMA_VENCIMENTO": true, "ESTOQUE_CONTAGEM_VENCIDA": true, "ESTOQUE_CONTAGEM_DIVERGENCIA": true, "ESTOQUE_CONTAGEM_CONCLUIDA": true, "ESTOQUE_CONTAGEM_APROVADA": true, "ESTOQUE_ABAIXO_MINIMO": true, "ESTOQUE_NEGATIVO": true, "ESTOQUE_LOTE_PROXIMO_VENCIMENTO": true, "ESTOQUE_MOVIMENTACAO_INCOMUM": true,
	}
	seen := map[string]bool{}
	for rows.Next() {
		var key, description string
		if err = rows.Scan(&key, &description); err != nil {
			t.Fatal(err)
		}
		if !expected[key] || description == "" {
			t.Fatalf("produtor ativo divergente: %s", key)
		}
		seen[key] = true
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != len(expected) {
		t.Fatalf("produtores ativos=%d esperados=%d", len(seen), len(expected))
	}
}
