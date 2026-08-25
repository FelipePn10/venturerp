package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeliveryRescheduleRejectsTextualItemCodeInPortuguese(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/delivery-reschedule/create", strings.NewReader(`{"item_code":"4853"}`))
	rec := httptest.NewRecorder()

	(&DeliveryRescheduleHandler{}).Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusBadRequest)
	}
	if body := rec.Body.String(); !strings.Contains(body, "O código do item deve ser informado como número") {
		t.Fatalf("resposta de validação inesperada: %s", body)
	}
}
