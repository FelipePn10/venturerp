package purchase_requisition_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
)

func toRequisitionResponse(r *entity.PurchaseRequisition) *response.PurchaseRequisitionResponse {
	if r == nil {
		return nil
	}
	return &response.PurchaseRequisitionResponse{
		ID:                    r.ID,
		Code:                  r.Code,
		EnterpriseCode:        r.EnterpriseCode,
		RequestTypeCode:       r.RequestTypeCode,
		RequesterEmployeeCode: r.RequesterEmployeeCode,
		EmissionDate:          r.EmissionDate,
		Status:                string(r.Status),
		Notes:                 r.Notes,
		IsActive:              r.IsActive,
		CreatedAt:             r.CreatedAt,
		CreatedBy:             r.CreatedBy,
		UpdatedAt:             r.UpdatedAt,
		Items:                 toRequisitionItemValues(r.Items),
	}
}

func toRequisitionResponses(reqs []*entity.PurchaseRequisition) []*response.PurchaseRequisitionResponse {
	out := make([]*response.PurchaseRequisitionResponse, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, toRequisitionResponse(r))
	}
	return out
}

func toRequisitionItemResponse(it *entity.PurchaseRequisitionItem) *response.PurchaseRequisitionItemResponse {
	if it == nil {
		return nil
	}
	return &response.PurchaseRequisitionItemResponse{
		ID:                it.ID,
		RequisitionCode:   it.RequisitionCode,
		Sequence:          it.Sequence,
		ItemCode:          it.ItemCode,
		Quantity:          it.Quantity,
		AttendedQty:       it.AttendedQty,
		CancelledQty:      it.CancelledQty,
		Balance:           it.Balance(),
		UOM:               it.UOM,
		CostCenterCode:    it.CostCenterCode,
		AccountingAccount: it.AccountingAccount,
		SuggestedPrice:    it.SuggestedPrice,
		DeliveryDate:      it.DeliveryDate,
		Application:       it.Application,
		UtilizationType:   it.UtilizationType,
		Status:            string(it.Status),
		IsActive:          it.IsActive,
		CreatedAt:         it.CreatedAt,
	}
}

func toRequisitionItemValues(items []*entity.PurchaseRequisitionItem) []response.PurchaseRequisitionItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.PurchaseRequisitionItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, *toRequisitionItemResponse(it))
	}
	return out
}

func toRequisitionItemResponses(items []*entity.PurchaseRequisitionItem) []*response.PurchaseRequisitionItemResponse {
	out := make([]*response.PurchaseRequisitionItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toRequisitionItemResponse(it))
	}
	return out
}
