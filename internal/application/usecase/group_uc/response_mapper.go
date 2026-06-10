package group_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/group/entity"
)

func toGroupResponse(g *entity.Group) *response.GroupResponse {
	if g == nil {
		return nil
	}
	return &response.GroupResponse{
		ID:           g.ID,
		Code:         g.Code,
		Description:  g.Description,
		EnterpriseID: g.EnterpriseID,
		CreatedBy:    g.CreatedBy,
	}
}
