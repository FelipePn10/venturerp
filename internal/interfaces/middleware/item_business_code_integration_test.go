//go:build integration

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type realRouterItemAuth struct{ ports.AuthService }

func (realRouterItemAuth) CanCreateItem(context.Context) bool  { return true }
func (realRouterItemAuth) FindItemByCode(context.Context) bool { return true }

func TestItemBusinessCodeHTTPAcrossOperationalDomains(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	userID := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Teste compatibilidade',$2,'x')`, userID, userID+"@test.local"); err != nil {
		t.Fatal(err)
	}
	var warehouseCode int64
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed)
		VALUES($1,'Teste compatibilidade',$2,'NORMAL','INTERNO',TRUE,TRUE) RETURNING id`, fmt.Sprintf("W-%d", testutil.UniqueCode()), userID).Scan(&warehouseCode); err != nil {
		t.Fatal(err)
	}
	enterpriseCodeA := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	enterpriseCodeB := enterpriseCodeA + 1
	var enterpriseA, enterpriseB int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Compat A',$3),($2,'Compat B',$3) RETURNING id`, enterpriseCodeA, enterpriseCodeB, userID).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM enterprise WHERE code=$1`, enterpriseCodeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_count_audit WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_cycle_counts WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise WHERE id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM warehouse WHERE id=$1`, warehouseCode)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	})

	legacyA, legacyB, leadingLegacy, numericLegacy, numericZeroLegacy := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	_, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,engineering_weight,engineering_dimensions,planning_reorder_point,engineering_type,commercial_description,accounting_cest)
		VALUES($1,$2,'Item A',$7,$8,'TEA452-0','{"gross":1,"net":1,"unit":"KG"}','{"length":10,"width":20,"height":30}',NULL,2,'Descrição preservada','1234567'),($1,$3,'Item B',$7,$9,'TEA452-0','{"gross":1,"net":1,"unit":"KG"}','{"length":10,"width":20,"height":30}',NULL,2,'Descrição preservada','1234567'),($1,$4,'Zeros',$7,$8,'0007-A','{"gross":1,"net":1,"unit":"KG"}','{"length":10,"width":20,"height":30}',NULL,2,'Descrição preservada','1234567'),($1,$5,'Numérico',$7,$8,'4853','{"gross":1,"net":1,"unit":"KG"}','{}','{}',2,'Descrição preservada','1234567'),($1,$6,'Zeros numérico',$7,$8,'0007','{"gross":1,"net":1,"unit":"KG"}','{"length":10,"width":20,"height":30}',NULL,2,'Descrição preservada','1234567')`,
		warehouseCode, legacyA, legacyB, leadingLegacy, numericLegacy, numericZeroLegacy, userID, enterpriseA, enterpriseB)
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := enterpriseA
			if r.Header.Get("X-Test-Tenant") == "B" {
				tenantID = enterpriseB
			}
			user := &security.AuthUser{EnterpriseID: tenantID, Role: "ADMIN"}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextkey.UserKey, user)))
		})
	})
	router.Use(ItemBusinessCodeCompatibility(pool))
	echo := func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}
	for _, path := range []string{
		"/api/item-suppliers", "/api/stock/movements", "/api/sales-orders",
		"/api/mrp-calculation", "/api/items", "/api/items/structure", "/api/production-order",
	} {
		router.Post(path, echo)
	}
	pathEcho := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		raw := chi.URLParam(r, "code")
		if raw == "" {
			raw = chi.URLParam(r, "itemCode")
		}
		if raw == "" {
			raw = chi.URLParam(r, "item_code")
		}
		legacy, _ := strconv.ParseInt(raw, 10, 64)
		_ = json.NewEncoder(w).Encode(map[string]any{"item_code": legacy})
	}
	realItemRepo := itemrepo.NewRepositoryItemSQLC(sqlc.New(pool))
	realItemHandler := handler.NewCreateItemHandler(nil, item_uc.NewUpdateItemUseCase(realItemRepo, realRouterItemAuth{}), nil, nil, nil)
	realActivationHandler := handler.NewItemActivationHandler(&item_uc.ValidateItemActivationUseCase{ItemRepo: realItemRepo, Auth: realRouterItemAuth{}})
	groupedRouter := chi.NewRouter()
	groupedRouter.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				user := &security.AuthUser{EnterpriseID: enterpriseA, EnterpriseCode: enterpriseCodeA, Role: "ADMIN"}
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, user)))
			})
		})
		r.Use(TenantBodyGuard)
		r.Use(ItemBusinessCodeCompatibility(pool))
		r.Route("/api/items", func(r chi.Router) {
			r.With(RequireRole("ADMIN", "USER")).Put("/{code}", realItemHandler.UpdateItem)
			r.With(RequireRole("ADMIN", "USER")).Get("/{code}/activation-readiness", realActivationHandler.ValidateActivation)
		})
	})
	groupedPut := httptest.NewRequest(http.MethodPut, "/api/items/4853", strings.NewReader(`{"commercial":{"warranty_days":731},"accounting":{"origin":1,"calculate_pis_cofins":true}}`))
	groupedPut.Header.Set("Content-Type", "application/json")
	groupedPutRec := httptest.NewRecorder()
	groupedRouter.ServeHTTP(groupedPutRec, groupedPut)
	if groupedPutRec.Code != http.StatusOK {
		t.Fatalf("router real PUT numérico status=%d body=%s", groupedPutRec.Code, groupedPutRec.Body.String())
	}
	groupedReadyRec := httptest.NewRecorder()
	groupedRouter.ServeHTTP(groupedReadyRec, httptest.NewRequest(http.MethodGet, "/api/items/4853/activation-readiness", nil))
	if groupedReadyRec.Code != http.StatusOK || !strings.Contains(groupedReadyRec.Body.String(), `"item_code":"4853"`) {
		t.Fatalf("router real readiness numérico status=%d body=%s", groupedReadyRec.Code, groupedReadyRec.Body.String())
	}
	router.Put("/api/items/{code}", realItemHandler.UpdateItem)
	router.Get("/api/items/{code}/activation-readiness", realActivationHandler.ValidateActivation)
	for _, route := range []string{"/api/stock/movements/item/{itemCode}", "/api/configurator/items/{itemCode}/rules", "/api/items/structure/resolve/{itemCode}", "/api/quality/plans/by-item/{itemCode}", "/api/standard-cost/items/{itemCode}", "/api/mrp-calculation/profile/{item_code}/1", "/api/item-calendar-promise/{item_code}/x/2026/8", "/api/financial/relatorios/ficha-tecnica/{item_code}"} {
		router.Get(route, pathEcho)
	}

	tests := []struct {
		name, tenant, code string
		legacy             int64
	}{
		{"tenant A", "A", "TEA452-0", legacyA},
		{"tenant B same code", "B", "TEA452-0", legacyB},
		{"leading zeros", "A", "0007-A", leadingLegacy},
		{"numeric business code", "A", "4853", numericLegacy},
		{"numeric leading zeros", "A", "0007", numericZeroLegacy},
	}
	for _, path := range []string{"/api/stock/movements/item/4853", "/api/configurator/items/4853/rules", "/api/items/structure/resolve/4853", "/api/quality/plans/by-item/4853", "/api/standard-cost/items/4853", "/api/mrp-calculation/profile/4853/1", "/api/item-calendar-promise/4853/x/2026/8", "/api/financial/relatorios/ficha-tecnica/4853"} {
		t.Run("numeric path "+path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"item_code":"4853"`) {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}

	for _, code := range []string{"4853", "0007", "TEA452-0"} {
		for _, methodPath := range []struct{ method, path string }{{http.MethodPut, "/api/items/" + code}, {http.MethodGet, "/api/items/" + code + "/activation-readiness"}} {
			t.Run(methodPath.method+" "+code, func(t *testing.T) {
				body := ""
				if methodPath.method == http.MethodPut {
					body = `{"commercial":{"warranty_days":730},"accounting":{"origin":1,"calculate_pis_cofins":true}}`
				}
				req := httptest.NewRequest(methodPath.method, methodPath.path, strings.NewReader(body))
				if body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
				}
				if methodPath.method == http.MethodPut {
					return // status 200 came from the real handler/use case/repository chain.
				}
				var response struct {
					ItemCode string `json:"item_code"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.ItemCode != code {
					t.Fatalf("código perdido: %q", response.ItemCode)
				}
			})
		}
	}
	var warranty, origin int
	var calculate bool
	var description, cest string
	var dimensionsAbsent, reorderPointAbsent bool
	if err = pool.QueryRow(ctx, `SELECT commercial_warranty_days,accounting_origin,accounting_calculate_pis_cofins,commercial_description,accounting_cest,engineering_dimensions = '{}'::jsonb,planning_reorder_point = '{}'::jsonb FROM items WHERE enterprise_id=$1 AND code=$2`, enterpriseA, numericLegacy).Scan(&warranty, &origin, &calculate, &description, &cest, &dimensionsAbsent, &reorderPointAbsent); err != nil {
		t.Fatal(err)
	}
	if warranty != 730 || origin != 1 || !calculate || description != "Descrição preservada" || cest != "1234567" || !dimensionsAbsent || !reorderPointAbsent {
		t.Fatalf("update parcial não preservou estado: warranty=%d origin=%d calculate=%t description=%q cest=%q dimensions_absent=%t reorder_point_absent=%t", warranty, origin, calculate, description, cest, dimensionsAbsent, reorderPointAbsent)
	}
	policyReq := httptest.NewRequest(http.MethodPut, "/api/items/4853", strings.NewReader(`{"warehouse":{"cyclical_count_config":{"days_interval":1}}}`))
	policyReq.Header.Set("Content-Type", "application/json")
	policyRec := httptest.NewRecorder()
	router.ServeHTTP(policyRec, policyReq)
	if policyRec.Code != http.StatusOK || !strings.Contains(policyRec.Body.String(), `"cyclical_count_config":{"days_interval":1}`) {
		t.Fatalf("política snake_case status=%d body=%s", policyRec.Code, policyRec.Body.String())
	}
	if err = notificationrepo.New(pool).SchedulePolicyCycleCounts(ctx); err != nil {
		t.Fatal(err)
	}
	var scheduledOrigin string
	var scheduledDays int
	if err = pool.QueryRow(ctx, `SELECT origin,policy_days FROM stock_cycle_counts WHERE enterprise_id=$1 AND item_code=$2 AND state='PROGRAMADA'`, enterpriseA, numericLegacy).Scan(&scheduledOrigin, &scheduledDays); err != nil {
		t.Fatal(err)
	}
	if scheduledOrigin != "POLITICA_ITEM" || scheduledDays != 1 {
		t.Fatalf("scheduler não ativado pelo HTTP: origin=%s days=%d", scheduledOrigin, scheduledDays)
	}
	for _, invalid := range []string{`{"warehouse":{"cyclical_count_config":{"days_interval":0}}}`, `{"warehouse":{"cyclical_count_config":{"days_interval":-1}}}`, `{"warehouse":{"cyclical_count_config":{"days":1}}}`} {
		req := httptest.NewRequest(http.MethodPut, "/api/items/4853", strings.NewReader(invalid))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("política inválida aceita status=%d body=%s request=%s", rec.Code, rec.Body.String(), invalid)
		}
	}
	legacyPolicyReq := httptest.NewRequest(http.MethodPut, "/api/items/4853", strings.NewReader(`{"warehouse":{"cyclical_count_config":{"DaysInterval":2}}}`))
	legacyPolicyReq.Header.Set("Content-Type", "application/json")
	legacyPolicyRec := httptest.NewRecorder()
	router.ServeHTTP(legacyPolicyRec, legacyPolicyReq)
	if legacyPolicyRec.Code != http.StatusOK || !strings.Contains(legacyPolicyRec.Body.String(), `"cyclical_count_config":{"days_interval":2}`) {
		t.Fatalf("compatibilidade DaysInterval status=%d body=%s", legacyPolicyRec.Code, legacyPolicyRec.Body.String())
	}
	for _, tc := range tests {
		for _, path := range []string{
			"/api/item-suppliers", "/api/stock/movements", "/api/sales-orders",
			"/api/mrp-calculation", "/api/items/structure", "/api/production-order",
		} {
			t.Run(tc.name+" "+path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(fmt.Sprintf(`{"item_code":%q}`, tc.code)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Test-Tenant", tc.tenant)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
				}
				var body struct {
					ItemCode       string `json:"item_code"`
					LegacyItemCode int64  `json:"legacy_item_code"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.ItemCode != tc.code || body.LegacyItemCode != tc.legacy {
					t.Fatalf("resposta incorreta: %+v", body)
				}
			})
		}
	}

	t.Run("item base nested reference", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"engineering":{"item_base_cod":"0007-A"}}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body struct {
			Engineering struct {
				ItemBaseCod       string `json:"item_base_cod"`
				LegacyItemBaseCod int64  `json:"legacy_item_base_cod"`
			} `json:"engineering"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Engineering.ItemBaseCod != "0007-A" || body.Engineering.LegacyItemBaseCod != leadingLegacy {
			t.Fatalf("item_base_cod incorreto: %+v", body.Engineering)
		}
	})
}
