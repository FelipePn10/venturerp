package generate_mask_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity"
)

func toItemMaskResponse(m *entity.ItemMask) *response.ItemMaskResponse {
	if m == nil {
		return nil
	}
	answers := make([]response.MaskAnswerResponse, 0, len(m.Answers))
	for _, a := range m.Answers {
		answers = append(answers, response.MaskAnswerResponse{
			QuestionID:  a.QuestionID(),
			OptionID:    a.OptionID(),
			OptionValue: a.OptionValue(),
			Position:    a.Position(),
		})
	}
	return &response.ItemMaskResponse{
		ID:        m.ID,
		ItemCode:  m.ItemCode,
		Mask:      m.Mask,
		MaskHash:  m.MaskHash,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
		Answers:   answers,
	}
}
