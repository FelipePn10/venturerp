package enterprise_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity"
)

func toEnterpriseResponse(e *entity.Enterprise) *response.EnterpriseResponse {
	if e == nil {
		return nil
	}
	return &response.EnterpriseResponse{
		ID:        e.ID,
		Code:      e.Code,
		Name:      e.Name,
		CreatedBy: e.CreatedBy,
	}
}
