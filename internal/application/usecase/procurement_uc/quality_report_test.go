package procurement_uc

import (
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/google/uuid"
)

func TestToReceivingInspectionQualityReportResponse(t *testing.T) {
	linkedBy := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	registeredOn := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	linkedAt := time.Date(2026, 8, 12, 15, 30, 0, 0, time.UTC)
	fileName := "certificado-lote-42.pdf"
	contentType := "application/pdf"
	link := &entity.ReceivingInspectionQualityReport{
		EnterpriseID: 7, InspectionOrderID: 42, QualityReportID: 81, ItemSupplierID: 13,
		RegisteredOn: registeredOn, Status: "APROVADO", FileName: &fileName,
		ContentType: &contentType, LinkedAt: linkedAt, LinkedBy: linkedBy,
	}

	got := toReceivingInspectionQualityReportResponse(link)
	if got.InspectionOrderID != 42 || got.QualityReportID != 81 || got.ItemSupplierID != 13 {
		t.Fatalf("identificadores incorretos: %+v", got)
	}
	if got.Status != "APROVADO" || got.FileName == nil || *got.FileName != fileName {
		t.Fatalf("metadados do laudo incorretos: %+v", got)
	}
	if !got.RegisteredOn.Equal(registeredOn) || !got.LinkedAt.Equal(linkedAt) || got.LinkedBy != linkedBy {
		t.Fatalf("rastreabilidade incorreta: %+v", got)
	}
}
