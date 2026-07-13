package item_supplier_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
)

func toItemPreferredSupplierResponse(s *entity.ItemPreferredSupplier) *response.ItemPreferredSupplierResponse {
	if s == nil {
		return nil
	}
	return &response.ItemPreferredSupplierResponse{ID: s.ID, EnterpriseID: s.EnterpriseID, ItemCode: s.ItemCode, SupplierCode: s.SupplierCode, Mask: s.Mask, Ranking: s.Ranking, SupplierItemCode: s.SupplierItemCode, SupplierDescription: s.SupplierDescription, UOM: s.UOM, XMLUOM: s.XMLUOM, ConversionFactor: s.ConversionFactor, PackageQuantity: s.PackageQuantity, IsPreferred: s.IsPreferred, SupplierUF: s.SupplierUF, ClassificationID: s.ClassificationID, ClassificationDate: s.ClassificationDate, ClassificationGrade: s.ClassificationGrade, DirectBilling: s.DirectBilling, ThirdPartyOrder: s.ThirdPartyOrder, IgnoreAvgCostAddition: s.IgnoreAvgCostAddition, Ecommerce: s.Ecommerce, Barcode: s.Barcode, Notes: s.Notes, ValidUntil: s.ValidUntil, LeadTimeDays: s.LeadTimeDays, IsActive: s.IsActive, CreatedAt: s.CreatedAt, CreatedBy: s.CreatedBy, UpdatedAt: s.UpdatedAt}
}
func toItemPreferredSupplierResponses(x []*entity.ItemPreferredSupplier) []*response.ItemPreferredSupplierResponse {
	out := make([]*response.ItemPreferredSupplierResponse, 0, len(x))
	for _, s := range x {
		out = append(out, toItemPreferredSupplierResponse(s))
	}
	return out
}
func toQualityResponse(q *entity.QualityReport) *response.ItemSupplierQualityReportResponse {
	if q == nil {
		return nil
	}
	return &response.ItemSupplierQualityReportResponse{ID: q.ID, ItemSupplierID: q.ItemSupplierID, RegisteredOn: q.RegisteredOn, Status: q.Status, FileName: q.FileName, ContentType: q.ContentType, HasAttachment: len(q.Content) > 0, Notes: q.Notes, CreatedAt: q.CreatedAt, CreatedBy: q.CreatedBy}
}
