package item_classification_uc

import (
	"strings"

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

// classificationView carries the mask and the sibling set needed to resolve the
// parent code and the hierarchical description without a round trip per row.
type classificationView struct {
	mask    *entity.ItemClassificationMask
	byID    map[int64]*entity.ItemClassification
	byLevel map[string]string // code → description
}

func newClassificationView(mask *entity.ItemClassificationMask, all []*entity.ItemClassification) *classificationView {
	v := &classificationView{mask: mask, byID: make(map[int64]*entity.ItemClassification, len(all)), byLevel: make(map[string]string, len(all))}
	for _, c := range all {
		v.byID[c.ID] = c
		v.byLevel[c.Code] = c.Description
	}
	return v
}

// fullDescription walks the code prefixes ("01", "01.02", "01.02.03") and joins
// the descriptions already known for each level.
func (v *classificationView) fullDescription(code string) string {
	parts := strings.Split(code, ".")
	names := make([]string, 0, len(parts))
	for i := range parts {
		prefix := strings.Join(parts[:i+1], ".")
		if desc, ok := v.byLevel[prefix]; ok {
			names = append(names, desc)
		}
	}
	return strings.Join(names, " > ")
}

func (v *classificationView) toResponse(c *entity.ItemClassification) *response.ItemClassificationResponse {
	if c == nil {
		return nil
	}
	out := &response.ItemClassificationResponse{
		ID:              c.ID,
		Code:            c.Code,
		MaskID:          c.MaskID,
		ParentID:        c.ParentID,
		Level:           c.Level,
		Description:     c.Description,
		FullDescription: c.Description,
		IsActive:        c.IsActive,
		CreatedAt:       c.CreatedAt,
	}
	if v != nil {
		if v.mask != nil {
			out.MaskCode, out.Mask = v.mask.Code, v.mask.Mask
		}
		if c.ParentID != nil {
			if parent, ok := v.byID[*c.ParentID]; ok {
				out.ParentCode = parent.Code
			}
		}
		if full := v.fullDescription(c.Code); full != "" {
			out.FullDescription = full
		}
	}
	// Sem o pai carregado, o código do pai ainda é derivável do próprio código.
	if out.ParentCode == "" {
		if idx := strings.LastIndex(c.Code, "."); idx > 0 {
			out.ParentCode = c.Code[:idx]
		}
	}
	return out
}

func (v *classificationView) toResponses(list []*entity.ItemClassification) []*response.ItemClassificationResponse {
	out := make([]*response.ItemClassificationResponse, 0, len(list))
	for _, c := range list {
		out = append(out, v.toResponse(c))
	}
	return out
}
