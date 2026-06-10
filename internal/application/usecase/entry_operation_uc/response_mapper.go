package entry_operation_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
)

func toStateGroupResponse(g *entity.StateGroup) *response.StateGroupResponse {
	if g == nil {
		return nil
	}
	return &response.StateGroupResponse{
		ID:          g.ID,
		Code:        g.Code,
		Description: g.Description,
		IsActive:    g.IsActive,
		CreatedAt:   g.CreatedAt,
		CreatedBy:   g.CreatedBy,
		UFs:         g.UFs,
	}
}

func toStateGroupResponses(list []*entity.StateGroup) []*response.StateGroupResponse {
	out := make([]*response.StateGroupResponse, 0, len(list))
	for _, g := range list {
		out = append(out, toStateGroupResponse(g))
	}
	return out
}

func toEntryOperationTypeResponse(o *entity.EntryOperationType) *response.EntryOperationTypeResponse {
	if o == nil {
		return nil
	}
	return &response.EntryOperationTypeResponse{
		ID:                 o.ID,
		Code:               o.Code,
		Description:        o.Description,
		InvoiceTypeCode:    o.InvoiceTypeCode,
		NatureOperation:    o.NatureOperation,
		ClassificationType: o.ClassificationType,
		ClassificationCode: o.ClassificationCode,
		StateGroupCode:     o.StateGroupCode,
		SupplierTypeCode:   o.SupplierTypeCode,
		IsActive:           o.IsActive,
		CreatedAt:          o.CreatedAt,
		CreatedBy:          o.CreatedBy,
	}
}

func toEntryOperationTypeResponses(list []*entity.EntryOperationType) []*response.EntryOperationTypeResponse {
	out := make([]*response.EntryOperationTypeResponse, 0, len(list))
	for _, o := range list {
		out = append(out, toEntryOperationTypeResponse(o))
	}
	return out
}
