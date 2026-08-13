//go:build integration

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

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
		_, _ = pool.Exec(context.Background(), `DELETE FROM items WHERE enterprise_id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM enterprise WHERE id IN ($1,$2)`, enterpriseA, enterpriseB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM warehouse WHERE id=$1`, warehouseCode)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	})

	legacyA, legacyB, leadingLegacy := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	_, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code)
		VALUES($1,$2,'Item A',$5,$6,'TEA452-0'),($1,$3,'Item B',$5,$7,'TEA452-0'),($1,$4,'Zeros',$5,$6,'0007-A')`,
		warehouseCode, legacyA, legacyB, leadingLegacy, userID, enterpriseA, enterpriseB)
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

	tests := []struct {
		name, tenant, code string
		legacy             int64
	}{
		{"tenant A", "A", "TEA452-0", legacyA},
		{"tenant B same code", "B", "TEA452-0", legacyB},
		{"leading zeros", "A", "0007-A", leadingLegacy},
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
