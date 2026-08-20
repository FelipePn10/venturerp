package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/notification_uc"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type notificationValidationRepo struct {
	notification_uc.Repository
	retryErr        error
	transitionErr   error
	requestedTenant *int64
}

func (r notificationValidationRepo) ListEligibleUsers(_ context.Context, tenant int64) ([]notificationentity.EligibleUser, error) {
	if r.requestedTenant != nil {
		*r.requestedTenant = tenant
	}
	return []notificationentity.EligibleUser{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000009"), Name: "Usuário seguro", Role: "USER", Active: false}}, nil
}
func (r notificationValidationRepo) ListEligibleDepartments(_ context.Context, tenant int64) ([]notificationentity.EligibleDepartment, error) {
	if r.requestedTenant != nil {
		*r.requestedTenant = tenant
	}
	return []notificationentity.EligibleDepartment{{Code: "COMERCIAL", Description: "Comercial", Active: true}}, nil
}

func (r notificationValidationRepo) RetryDelivery(context.Context, int64, uuid.UUID, uuid.UUID) error {
	return r.retryErr
}
func (r notificationValidationRepo) TransitionCycleCount(context.Context, int64, uuid.UUID, uuid.UUID, notificationentity.CycleCountState, *string, string) (notificationentity.CycleCount, error) {
	return notificationentity.CycleCount{}, r.transitionErr
}

func notificationRequest(method, path, body, role string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	user := &security.AuthUser{ID: "00000000-0000-0000-0000-000000000001", EnterpriseID: 1, Role: role}
	return req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, user))
}

func withNotificationParam(req *http.Request, key, value string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func TestNotificationRoutesReturnSemanticErrors(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name string
		want int
		call func(*NotificationHandler, http.ResponseWriter, *http.Request)
		req  *http.Request
		repo notificationValidationRepo
	}{
		{"settings inválidos", 422, (*NotificationHandler).SaveSettings, notificationRequest("PUT", "/settings", `{"digest_time":"99:99","timezone":"Inválido","retention_days":1,"max_emails_per_minute":0}`, "ADMIN"), notificationValidationRepo{}},
		{"assinatura inválida", 422, (*NotificationHandler).CreateSubscription, notificationRequest("POST", "/subscriptions", `{"event_key":"","cadence":"INVALIDA","thresholds":{},"recipients":[]}`, "ADMIN"), notificationValidationRepo{}},
		{"retry inexistente", 404, (*NotificationHandler).RetryDelivery, withNotificationParam(notificationRequest("POST", "/deliveries/"+id.String()+"/retry", `{}`, "ADMIN"), "id", id.String()), notificationValidationRepo{retryErr: fmt.Errorf("%w: entrega", notificationentity.ErrNotFound)}},
		{"contagem sem item", 422, (*NotificationHandler).CreateCycleCount, notificationRequest("POST", "/cycle-counts", `{"warehouse_id":1,"item_code":"","scheduled_for":"2026-08-20T12:00:00Z"}`, "USER"), notificationValidationRepo{}},
		{"transição concorrente", 409, (*NotificationHandler).TransitionCycleCount, withNotificationParam(notificationRequest("POST", "/cycle-counts/"+id.String()+"/transition", `{"state":"CONCLUIDA","counted_quantity":"1"}`, "USER"), "id", id.String()), notificationValidationRepo{transitionErr: fmt.Errorf("%w: estado alterado", notificationentity.ErrConflict)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewNotificationHandler(notification_uc.New(tt.repo))
			rec := httptest.NewRecorder()
			tt.call(h, rec, tt.req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "error") {
				t.Fatalf("mensagem ausente: %s", rec.Body.String())
			}
		})
	}
}

func TestNotificationEligibleRecipientsUseAuthenticatedTenantForAdminAndUser(t *testing.T) {
	for _, role := range []string{"ADMIN", "USER"} {
		for _, endpoint := range []string{"users", "departments"} {
			t.Run(role+"/"+endpoint, func(t *testing.T) {
				var tenant int64
				h := NewNotificationHandler(notification_uc.New(notificationValidationRepo{requestedTenant: &tenant}))
				req := notificationRequest(http.MethodGet, "/recipients/"+endpoint, "", role)
				rec := httptest.NewRecorder()
				if endpoint == "users" {
					h.ListEligibleUsers(rec, req)
				} else {
					h.ListEligibleDepartments(rec, req)
				}
				if rec.Code != http.StatusOK {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
				}
				if tenant != 1 {
					t.Fatalf("tenant consultado=%d", tenant)
				}
				if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "token") || strings.Contains(rec.Body.String(), "email") {
					t.Fatalf("dado sensível exposto: %s", rec.Body.String())
				}
				if !strings.Contains(rec.Body.String(), `"active"`) {
					t.Fatalf("estado ativo ausente: %s", rec.Body.String())
				}
			})
		}
	}
}
