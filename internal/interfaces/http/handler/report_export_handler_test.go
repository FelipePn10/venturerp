package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type reportFiscalReader struct {
	config *fiscalentity.FiscalConfig
	err    error
}

func (r reportFiscalReader) GetFiscalConfig(context.Context) (*fiscalentity.FiscalConfig, error) {
	return r.config, r.err
}

func TestReportExportRejectsBlankLetterhead(t *testing.T) {
	for _, reader := range []reportFiscalReader{{err: errors.New("ausente")}, {config: &fiscalentity.FiscalConfig{}}} {
		h := NewReportExportHandler(reader)
		req := httptest.NewRequest(http.MethodPost, "/api/reports/export?format=csv", strings.NewReader(`{"title":"Teste","columns":["Código"],"rows":[["1"]]}`))
		rec := httptest.NewRecorder()
		h.Export(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, corpo=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestReportExportUsesAuthenticatedCompanyData(t *testing.T) {
	h := NewReportExportHandler(reportFiscalReader{config: &fiscalentity.FiscalConfig{RazaoSocial: "Venture ERP Ltda", CnpjEmpresa: "12345678000199"}})
	req := httptest.NewRequest(http.MethodPost, "/api/reports/export?format=csv", strings.NewReader(`{"title":"Teste","columns":["Código"],"rows":[["1"]]}`))
	rec := httptest.NewRecorder()
	h.Export(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, corpo=%s", rec.Code, rec.Body.String())
	}
}
