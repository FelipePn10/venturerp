package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	thirdpartyuc "github.com/FelipePn10/panossoerp/internal/application/usecase/third_party_service_uc"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type thirdPartyHTTPRepo struct {
	domain.Repository
	created         *domain.Price
	deleted         int64
	statusUpdated   string
	movementCreated *domain.Movement
	globalCreated   *domain.GlobalConversion
}

func (r *thirdPartyHTTPRepo) CreatePrice(_ context.Context, p *domain.Price, _ string) (*domain.Price, error) {
	p.ID = 7
	r.created = p
	return p, nil
}
func (r *thirdPartyHTTPRepo) UpdatePrice(_ context.Context, p *domain.Price, _ string) (*domain.Price, error) {
	r.created = p
	return p, nil
}
func (r *thirdPartyHTTPRepo) DeletePrice(_ context.Context, id int64, _ string, _ uuid.UUID) error {
	r.deleted = id
	return nil
}
func (r *thirdPartyHTTPRepo) ListPrices(context.Context, domain.PriceFilter) ([]domain.Price, error) {
	return []domain.Price{{ID: 7, UnitPrice: decimal.RequireFromString("12.5")}}, nil
}
func (r *thirdPartyHTTPRepo) ListOrders(context.Context, domain.OrderFilter) ([]domain.ServiceOrder, error) {
	return []domain.ServiceOrder{{Code: 9, ProductionOrderID: 4, ItemCode: 100, OperationID: 3, Quantity: decimal.NewFromInt(10), FulfilledQuantity: decimal.NewFromInt(4), DueDate: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Status: "FIRM"}}, nil
}
func (r *thirdPartyHTTPRepo) UpdateOrderStatus(_ context.Context, id int64, status string, _, _ *int64, _ uuid.UUID) (*domain.ServiceOrder, error) {
	r.statusUpdated = status
	return &domain.ServiceOrder{ID: id, Code: 9, Status: status, Quantity: decimal.NewFromInt(10)}, nil
}
func (r *thirdPartyHTTPRepo) AddMovement(_ context.Context, id int64, movement domain.Movement) (*domain.Movement, error) {
	movement.ID, movement.ServiceOrderID = 13, id
	r.movementCreated = &movement
	return &movement, nil
}
func (r *thirdPartyHTTPRepo) ListMovements(context.Context, int64) ([]domain.Movement, error) {
	return []domain.Movement{{ID: 13, ServiceOrderID: 7, MovementType: "REMITTANCE", Quantity: decimal.NewFromInt(2), OccurredAt: time.Now(), IdempotencyKey: "http-movement"}}, nil
}
func (r *thirdPartyHTTPRepo) OrderHistory(context.Context, int64) ([]domain.OrderHistory, error) {
	return []domain.OrderHistory{{ID: 1, ServiceOrderID: 7, EventType: "CREATE", ActorID: uuid.New(), OccurredAt: time.Now()}}, nil
}
func (r *thirdPartyHTTPRepo) UpsertGlobalConversion(_ context.Context, conversion domain.GlobalConversion) (*domain.GlobalConversion, error) {
	conversion.ID, conversion.IsActive = 5, true
	r.globalCreated = &conversion
	return &conversion, nil
}
func (r *thirdPartyHTTPRepo) ListGlobalConversions(context.Context) ([]domain.GlobalConversion, error) {
	return []domain.GlobalConversion{{ID: 5, FromUOM: "CX", ToUOM: "UN", Factor: decimal.NewFromInt(12), IsActive: true}}, nil
}

func thirdPartyRouter(repo *thirdPartyHTTPRepo, role string) http.Handler {
	h := NewThirdPartyServiceHandler(thirdpartyuc.New(repo))
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			u := &security.AuthUser{ID: uuid.NewString(), Role: role, EnterpriseID: 1}
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, u)))
		})
	})
	r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/prices", h.ListPrices)
	r.With(httpmw.RequireRole("ADMIN")).Post("/prices", h.CreatePrice)
	r.With(httpmw.RequireRole("ADMIN")).Put("/prices/{id}", h.UpdatePrice)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/prices/{id}", h.DeletePrice)
	r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/orders/report", h.ListOrders)
	r.With(httpmw.RequireRole("ADMIN")).Patch("/orders/{id}/status", h.UpdateOrderStatus)
	r.With(httpmw.RequireRole("ADMIN")).Post("/orders/{id}/movements", h.AddMovement)
	r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/orders/{id}/movements", h.ListMovements)
	r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/orders/{id}/history", h.OrderHistory)
	r.With(httpmw.RequireRole("ADMIN")).Post("/global-conversions", h.UpsertGlobalConversion)
	r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/global-conversions", h.ListGlobalConversions)
	return r
}

func TestThirdPartyOrderReportCSV(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders/report?format=csv", nil)
	rec := httptest.NewRecorder()
	thirdPartyRouter(&thirdPartyHTTPRepo{}, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") || !strings.Contains(rec.Body.String(), "pending_quantity") || !strings.Contains(rec.Body.String(), ";9;;;4;0;3;100;") {
		t.Fatalf("status=%d content-type=%s body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestThirdPartyOrderReportPDF(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders/report?format=pdf", nil)
	rec := httptest.NewRecorder()
	thirdPartyRouter(&thirdPartyHTTPRepo{}, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/pdf" || !strings.HasPrefix(rec.Body.String(), "%PDF") {
		t.Fatalf("status=%d content-type=%s prefix=%q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()[:min(8, rec.Body.Len())])
	}
}
func TestThirdPartyPriceHTTPCreate(t *testing.T) {
	repo := &thirdPartyHTTPRepo{}
	h := NewThirdPartyServiceHandler(thirdpartyuc.New(repo))
	payload := `{"item_code":100,"supplier_code":200,"operation_id":3,"uom":"UN","reference_date":"2026-07-13T00:00:00Z","unit_price":"12.500000","freight_type":"FIXED","freight_value":"2","tax_percent":"5","reason":"initial"}`
	req := httptest.NewRequest(http.MethodPost, "/api/third-party-services/prices", strings.NewReader(payload))
	userID := uuid.New()
	req = req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, &security.AuthUser{ID: userID.String(), Role: "ADMIN", EnterpriseID: 1}))
	rec := httptest.NewRecorder()
	h.CreatePrice(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if repo.created == nil || repo.created.UnitPrice.String() != "12.5" || !strings.Contains(rec.Body.String(), `"id":7`) {
		t.Fatalf("created=%+v body=%s", repo.created, rec.Body.String())
	}
}
func TestThirdPartyPriceHTTPRejectsInvalidActorAndBody(t *testing.T) {
	h := NewThirdPartyServiceHandler(thirdpartyuc.New(&thirdPartyHTTPRepo{}))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.CreatePrice(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
	rec = httptest.NewRecorder()
	h.CreatePrice(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestThirdPartyPriceRoutesAuthorizeMutationsAndAllowConsultation(t *testing.T) {
	payload := `{"item_code":100,"supplier_code":200,"operation_id":3,"uom":"UN","reference_date":"2026-07-13T00:00:00Z","unit_price":"12.5","freight_type":"FIXED","freight_value":"0","tax_percent":"0","reason":"contract"}`
	repo := &thirdPartyHTTPRepo{}
	req := httptest.NewRequest(http.MethodPost, "/prices", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	thirdPartyRouter(repo, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || repo.created != nil {
		t.Fatalf("USER mutation status=%d created=%+v", rec.Code, repo.created)
	}
	req = httptest.NewRequest(http.MethodGet, "/prices?price_type=WITH_PRICE", nil)
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":7`) {
		t.Fatalf("USER query status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, "/prices/7", strings.NewReader(payload))
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || repo.created == nil || repo.created.ID != 7 {
		t.Fatalf("ADMIN update status=%d created=%+v body=%s", rec.Code, repo.created, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/prices/7?reason=expired", nil)
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || repo.deleted != 7 {
		t.Fatalf("ADMIN delete status=%d deleted=%d", rec.Code, repo.deleted)
	}
}

func TestThirdPartyOperationalRoutesEnforceRolesAndContracts(t *testing.T) {
	repo := &thirdPartyHTTPRepo{}
	statusPayload := `{"status":"RELEASED_WITHOUT_PO"}`
	req := httptest.NewRequest(http.MethodPatch, "/orders/7/status", strings.NewReader(statusPayload))
	rec := httptest.NewRecorder()
	thirdPartyRouter(repo, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || repo.statusUpdated != "" {
		t.Fatalf("USER status mutation=%d status=%s", rec.Code, repo.statusUpdated)
	}
	req = httptest.NewRequest(http.MethodPatch, "/orders/7/status", strings.NewReader(statusPayload))
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || repo.statusUpdated != "RELEASED_WITHOUT_PO" {
		t.Fatalf("ADMIN status mutation=%d body=%s", rec.Code, rec.Body.String())
	}
	movementPayload := `{"movement_type":"REMITTANCE","quantity":"2.000000","occurred_at":"2026-07-13T12:00:00Z","idempotency_key":"http-movement"}`
	req = httptest.NewRequest(http.MethodPost, "/orders/7/movements", strings.NewReader(movementPayload))
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || repo.movementCreated == nil || repo.movementCreated.IdempotencyKey != "http-movement" {
		t.Fatalf("movement status=%d body=%s created=%+v", rec.Code, rec.Body.String(), repo.movementCreated)
	}
	for _, path := range []string{"/orders/7/movements", "/orders/7/history", "/global-conversions"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		thirdPartyRouter(repo, "USER").ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("USER GET %s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	conversionPayload := `{"from_uom":"CX","to_uom":"UN","factor":"12"}`
	req = httptest.NewRequest(http.MethodPost, "/global-conversions", strings.NewReader(conversionPayload))
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("USER conversion mutation status=%d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/global-conversions", strings.NewReader(conversionPayload))
	rec = httptest.NewRecorder()
	thirdPartyRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || repo.globalCreated == nil || !repo.globalCreated.Factor.Equal(decimal.NewFromInt(12)) {
		t.Fatalf("ADMIN conversion status=%d body=%s conversion=%+v", rec.Code, rec.Body.String(), repo.globalCreated)
	}
}
