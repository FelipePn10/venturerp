package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	cuttinguc "github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc"
	employeeuc "github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	priorityuc "github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	quotationuc "github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc"
	cuttingentity "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	cuttingrepo "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	employeeentity "github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
	priorityentity "github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	quotationentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quotationrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type desktopValidationAuth struct{ ports.AuthService }

func (desktopValidationAuth) CanCreateEmployee(context.Context) bool      { return true }
func (desktopValidationAuth) CanCreateOrderPriority(context.Context) bool { return true }
func (desktopValidationAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
}

type quotationTenantAuth struct{ ports.AuthService }

func (quotationTenantAuth) CanCreateSalesOrder(context.Context) bool    { return true }
func (quotationTenantAuth) CanUpdateSalesOrder(context.Context) bool    { return true }
func (quotationTenantAuth) EnterpriseID(context.Context) (int64, error) { return 1, nil }

type quotationTenantRepo struct {
	quotationrepo.SalesQuotationRepository
	created                   bool
	cancellationReasonMissing bool
}

func (r *quotationTenantRepo) GetCancellationReason(context.Context, int64) (*quotationentity.CancellationReason, error) {
	if r.cancellationReasonMissing {
		return nil, quotationrepo.ErrCancellationReasonNotFound
	}
	panic("unused")
}

type cuttingSettingsRepo struct {
	cuttingrepo.CuttingPlanRepository
	settings *cuttingentity.CuttingSettings
}

func (r cuttingSettingsRepo) GetSettings(context.Context) (*cuttingentity.CuttingSettings, error) {
	return r.settings, nil
}

func TestCuttingSettingsReturns200WithoutAndWithConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name     string
		settings *cuttingentity.CuttingSettings
		mode     string
	}{{"tenant sem configuração", &cuttingentity.CuttingSettings{DefaultConsumptionMode: cuttingentity.ConsumptionAutomatic}, "AUTOMATIC"}, {"tenant configurado", &cuttingentity.CuttingSettings{DefaultConsumptionMode: cuttingentity.ConsumptionManual, DefaultMinRemnantMM: 125}, "MANUAL"}} {
		t.Run(tt.name, func(t *testing.T) {
			uc := cuttinguc.NewCuttingPlanUseCase(cuttingSettingsRepo{settings: tt.settings}, nil, nil)
			h := NewCuttingPlanHandler(uc, nil)
			rec := httptest.NewRecorder()
			h.GetSettings(rec, httptest.NewRequest(http.MethodGet, "/api/cutting-settings", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.mode) {
				t.Fatalf("defaults/configuração ausentes: %s", rec.Body.String())
			}
		})
	}
}

func TestSalesQuotationDomainValidationErrorsAreNot500(t *testing.T) {
	h := NewSalesQuotationHandler(&quotationuc.UseCase{Repo: &quotationTenantRepo{}, Auth: quotationTenantAuth{}}, nil)
	cases := []struct {
		name, path, body string
		call             func(*SalesQuotationHandler, http.ResponseWriter, *http.Request)
	}{
		{"empty parameter labels", "/parameters", `{"purchase_order_prompt":"","delivery_authorization_prompt":""}`, func(h *SalesQuotationHandler, w http.ResponseWriter, r *http.Request) { h.SaveParameters(w, r) }},
		{"unbalanced commission", "/commission-patterns", `{"description":"Padrão","commission_pct":"10","invoice_pct":"7","payment_pct":"2"}`, func(h *SalesQuotationHandler, w http.ResponseWriter, r *http.Request) { h.SaveCommissionPattern(w, r) }},
		{"empty cancellation reason", "/cancellation-reasons", `{"description":""}`, func(h *SalesQuotationHandler, w http.ResponseWriter, r *http.Request) { h.SaveCancellationReason(w, r) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.call(h, rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "error") {
				t.Fatalf("mensagem ausente: %s", rec.Body.String())
			}
		})
	}
}

func TestSalesQuotationInvalidCancellationReasonIsNot500(t *testing.T) {
	repo := &quotationTenantRepo{cancellationReasonMissing: true}
	h := NewSalesQuotationHandler(&quotationuc.UseCase{Repo: repo, Auth: quotationTenantAuth{}}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/sales-quotation/12/cancel", strings.NewReader(`{"reason_code":999999}`))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("code", "12")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()
	h.Cancel(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cancellation reason") {
		t.Fatalf("mensagem de domínio ausente: %s", rec.Body.String())
	}
}

type employeeValidationRepo struct{ duplicate bool }

func (r employeeValidationRepo) Create(_ context.Context, e *employeeentity.Employee) (*employeeentity.Employee, error) {
	if r.duplicate {
		return nil, fmt.Errorf("create employee: %w", &pgconn.PgError{Code: "23505"})
	}
	return e, nil
}
func (employeeValidationRepo) Update(context.Context, *employeeentity.Employee) (*employeeentity.Employee, error) {
	panic("unused")
}
func (employeeValidationRepo) GetByCode(context.Context, int64) (*employeeentity.Employee, error) {
	panic("unused")
}
func (employeeValidationRepo) List(context.Context) ([]*employeeentity.Employee, error) {
	panic("unused")
}
func (employeeValidationRepo) ListByRole(context.Context, string) ([]*employeeentity.Employee, error) {
	panic("unused")
}
func (employeeValidationRepo) Deactivate(context.Context, int64) error { panic("unused") }

func TestEmployeeValidationErrorsAreNot500(t *testing.T) {
	cases := []struct {
		name, body string
		repo       employeeValidationRepo
		want       int
	}{{"zero code", `{"code":0,"name":"Ana"}`, employeeValidationRepo{}, http.StatusUnprocessableEntity}, {"negative code", `{"code":-1,"name":"Ana"}`, employeeValidationRepo{}, http.StatusUnprocessableEntity}, {"empty name", `{"code":1,"name":""}`, employeeValidationRepo{}, http.StatusUnprocessableEntity}, {"duplicate", `{"code":1,"name":"Ana"}`, employeeValidationRepo{duplicate: true}, http.StatusConflict}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			uc := &employeeuc.CreateEmployeeUseCase{Repo: tt.repo, Auth: desktopValidationAuth{}}
			h := NewEmployeeHandler(uc, nil, nil, nil, nil)
			rec := httptest.NewRecorder()
			h.CreateEmployee(rec, httptest.NewRequest(http.MethodPost, "/api/employee/create", strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "error") {
				t.Fatalf("domain message missing: %s", rec.Body.String())
			}
		})
	}
}

func TestCreateItemRejectsUnknownUnitOfMeasurement(t *testing.T) {
	h := &ItemHandler{}
	rec := httptest.NewRecorder()
	h.CreateItem(rec, httptest.NewRequest(http.MethodPost, "/api/items/create", strings.NewReader(`{"warehouse":{"unit_of_measurement":"INEXISTENTE"}}`)))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unidade de medida") {
		t.Fatalf("mensagem incompreensível: %s", rec.Body.String())
	}
}

type priorityValidationRepo struct {
	rows []*priorityentity.OrderPriority
}

func (r priorityValidationRepo) List(context.Context) ([]*priorityentity.OrderPriority, error) {
	return r.rows, nil
}
func (priorityValidationRepo) Create(_ context.Context, p *priorityentity.OrderPriority) (*priorityentity.OrderPriority, error) {
	return p, nil
}
func (priorityValidationRepo) Update(context.Context, *priorityentity.OrderPriority) (*priorityentity.OrderPriority, error) {
	panic("unused")
}
func (priorityValidationRepo) GetByCode(context.Context, int64) (*priorityentity.OrderPriority, error) {
	panic("unused")
}
func (priorityValidationRepo) FindByValue(context.Context, float64) (*priorityentity.OrderPriority, error) {
	panic("unused")
}
func (priorityValidationRepo) Delete(context.Context, int64) error { panic("unused") }

func TestPriorityValidationErrorsAreNot500(t *testing.T) {
	cases := []struct {
		name, body string
		repo       priorityValidationRepo
		want       int
	}{{"invalid interval", `{"interval_start":10,"interval_end":10,"priority":"A"}`, priorityValidationRepo{}, http.StatusUnprocessableEntity}, {"overlap", `{"interval_start":5,"interval_end":15,"priority":"A"}`, priorityValidationRepo{rows: []*priorityentity.OrderPriority{{Code: 7, IntervalStart: 1, IntervalEnd: 10}}}, http.StatusConflict}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			uc := &priorityuc.CreateOrderPriorityUseCase{Repo: tt.repo, Auth: desktopValidationAuth{}}
			h := NewOrderPriorityHandler(uc, nil, nil)
			rec := httptest.NewRecorder()
			h.Create(rec, httptest.NewRequest(http.MethodPost, "/api/order-priority", strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "error") {
				t.Fatalf("domain message missing: %s", rec.Body.String())
			}
		})
	}
}
