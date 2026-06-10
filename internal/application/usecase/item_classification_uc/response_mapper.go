package item_classification_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func toClassificationMaskResponse(m *entity.ItemClassificationMask) *response.ItemClassificationMaskResponse {
	if m == nil {
		return nil
	}
	return &response.ItemClassificationMaskResponse{
		ID:          m.ID,
		Code:        m.Code,
		Mask:        m.Mask,
		Description: m.Description,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
	}
}

func toClassificationMaskResponses(list []*entity.ItemClassificationMask) []*response.ItemClassificationMaskResponse {
	out := make([]*response.ItemClassificationMaskResponse, 0, len(list))
	for _, m := range list {
		out = append(out, toClassificationMaskResponse(m))
	}
	return out
}

func toItemClassificationResponse(c *entity.ItemClassification) *response.ItemClassificationResponse {
	if c == nil {
		return nil
	}
	return &response.ItemClassificationResponse{
		ID:          c.ID,
		Code:        c.Code,
		MaskID:      c.MaskID,
		ParentID:    c.ParentID,
		Level:       c.Level,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}

func toItemClassificationResponses(list []*entity.ItemClassification) []*response.ItemClassificationResponse {
	out := make([]*response.ItemClassificationResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toItemClassificationResponse(c))
	}
	return out
}
