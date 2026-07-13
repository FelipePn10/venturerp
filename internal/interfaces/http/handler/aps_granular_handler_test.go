package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
	"github.com/go-chi/chi/v5"
)

type granularAPSRepo struct {
	apsrepo.APSRepository
	apsrepo.ConfigurationRepository
	calls []string
}

func (r *granularAPSRepo) UpdateEmployeeContact(context.Context, int64, int64, apsrepo.EmployeeContact) error {
	r.calls = append(r.calls, "patch-contact")
	return nil
}
func (r *granularAPSRepo) DeleteEmployeeContact(context.Context, int64, int64) error {
	r.calls = append(r.calls, "delete-contact")
	return nil
}
func (r *granularAPSRepo) UpdateEmployeeFunction(context.Context, int64, int64, apsrepo.EmployeeFunction) error {
	r.calls = append(r.calls, "patch-function")
	return nil
}
func (r *granularAPSRepo) DeleteEmployeeFunction(context.Context, int64, int64) error {
	r.calls = append(r.calls, "delete-function")
	return nil
}
func (r *granularAPSRepo) UpdateMachineService(context.Context, int64, int64, apsrepo.MachineService) error {
	r.calls = append(r.calls, "patch-service")
	return nil
}
func (r *granularAPSRepo) DeleteMachineService(context.Context, int64, int64) error {
	r.calls = append(r.calls, "delete-service")
	return nil
}
func (r *granularAPSRepo) UpdateMachineServiceItem(context.Context, int64, int64, int64, apsrepo.ServiceItem) error {
	r.calls = append(r.calls, "patch-item")
	return nil
}
func (r *granularAPSRepo) DeleteMachineServiceItem(context.Context, int64, int64, int64) error {
	r.calls = append(r.calls, "delete-item")
	return nil
}
func (r *granularAPSRepo) UpdateMachineSpecialValue(context.Context, int64, int64, apsrepo.SpecialValue) error {
	r.calls = append(r.calls, "patch-special")
	return nil
}
func (r *granularAPSRepo) DeleteMachineSpecialValue(context.Context, int64, int64) error {
	r.calls = append(r.calls, "delete-special")
	return nil
}

func granularRouter(repo *granularAPSRepo, role string) http.Handler {
	h := NewAPSHandler(aps_uc.New(repo), nil)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), contextkey.UserKey, &security.AuthUser{Role: role, EnterpriseID: 1})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.With(httpmw.RequireRole("ADMIN")).Patch("/employees/{employeeID}/contacts/{contactID}", h.UpdateEmployeeContact)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/employees/{employeeID}/contacts/{contactID}", h.DeleteEmployeeContact)
	r.With(httpmw.RequireRole("ADMIN")).Patch("/employees/{employeeID}/functions/{functionID}", h.UpdateEmployeeFunction)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/employees/{employeeID}/functions/{functionID}", h.DeleteEmployeeFunction)
	r.With(httpmw.RequireRole("ADMIN")).Patch("/resources/{machineID}/services/{serviceID}", h.UpdateMachineService)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/resources/{machineID}/services/{serviceID}", h.DeleteMachineService)
	r.With(httpmw.RequireRole("ADMIN")).Patch("/resources/{machineID}/services/{serviceID}/items/{itemID}", h.UpdateMachineServiceItem)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/resources/{machineID}/services/{serviceID}/items/{itemID}", h.DeleteMachineServiceItem)
	r.With(httpmw.RequireRole("ADMIN")).Patch("/resources/{machineID}/special-values/{fieldID}", h.UpdateMachineSpecialValue)
	r.With(httpmw.RequireRole("ADMIN")).Delete("/resources/{machineID}/special-values/{fieldID}", h.DeleteMachineSpecialValue)
	return r
}

func TestAPSGranularRoutesAdmin(t *testing.T) {
	tests := []struct{ name, method, path, body, call string }{
		{"contact patch", "PATCH", "/employees/1/contacts/2", `{"contact_type":"email","value":"planner@example.com","is_primary":true}`, "patch-contact"},
		{"contact delete", "DELETE", "/employees/1/contacts/2", "", "delete-contact"},
		{"function patch", "PATCH", "/employees/1/functions/2", `{"function_name":"Planner","is_supervisor":true}`, "patch-function"},
		{"function delete", "DELETE", "/employees/1/functions/2", "", "delete-function"},
		{"service patch", "PATCH", "/resources/1/services/2", `{"service_code":"PM-1","description":"Inspection","service_type":"mechanical","frequency_value":1,"frequency_unit":"month","implemented_on":"2026-07-12T00:00:00Z","responsible_employee_ids":[]}`, "patch-service"},
		{"service delete", "DELETE", "/resources/1/services/2", "", "delete-service"},
		{"item patch", "PATCH", "/resources/1/services/2/items/3", `{"item_code":10,"quantity":"2.500000"}`, "patch-item"},
		{"item delete", "DELETE", "/resources/1/services/2/items/3", "", "delete-item"},
		{"special patch", "PATCH", "/resources/1/special-values/2", `{"name":"Voltage","value_type":"number","numeric_value":"220"}`, "patch-special"},
		{"special delete", "DELETE", "/resources/1/special-values/2", "", "delete-special"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &granularAPSRepo{}
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			granularRouter(repo, "ADMIN").ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if len(repo.calls) != 1 || repo.calls[0] != tt.call {
				t.Fatalf("calls=%v", repo.calls)
			}
		})
	}
}

func TestAPSGranularRoutesRejectUserAndBadPayload(t *testing.T) {
	repo := &granularAPSRepo{}
	req := httptest.NewRequest(http.MethodDelete, "/employees/1/contacts/2", nil)
	rec := httptest.NewRecorder()
	granularRouter(repo, "USER").ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || len(repo.calls) != 0 {
		t.Fatalf("status=%d calls=%v", rec.Code, repo.calls)
	}
	req = httptest.NewRequest(http.MethodPatch, "/employees/1/contacts/2", strings.NewReader(`{"contact_type":"fax"}`))
	rec = httptest.NewRecorder()
	granularRouter(repo, "ADMIN").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || len(repo.calls) != 0 {
		t.Fatalf("status=%d calls=%v body=%s", rec.Code, repo.calls, rec.Body.String())
	}
}
