package modifier_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
)

func toModifierResponse(m *entity.Modifier) *response.ModifierResponse {
	if m == nil {
		return nil
	}
	return &response.ModifierResponse{
		ID:          m.ID,
		Description: m.Description,
		CreatedBy:   m.CreatedBy,
	}
}
