package restriction_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

func toRestrictionResponse(r *entity.Restriction) *response.RestrictionResponse {
	if r == nil {
		return nil
	}
	return &response.RestrictionResponse{
		ID:                   r.ID,
		Code:                 r.Code,
		Situation:            string(r.Situation),
		CustomerCode:         r.CustomerCode,
		ItemCode:             r.ItemCode,
		ReasonCode:           r.ReasonCode,
		ClassificationType:   r.ClassificationType,
		ClassificationOrigin: r.ClassificationOrigin,
		DivisionID:           r.DivisionID,
		Weight:               r.Weight,
		Dominants:            toRestrictionDominantValues(r.Dominants),
		Determinants:         toRestrictionDeterminantValues(r.Determinants),
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
		CreatedBy:            r.CreatedBy,
	}
}

func toRestrictionResponses(list []*entity.Restriction) []*response.RestrictionResponse {
	out := make([]*response.RestrictionResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toRestrictionResponse(r))
	}
	return out
}

func toRestrictionDominantValues(list []*entity.RestrictionDominant) []response.RestrictionDominantResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.RestrictionDominantResponse, 0, len(list))
	for _, d := range list {
		out = append(out, response.RestrictionDominantResponse{
			ID:            d.ID,
			RestrictionID: d.RestrictionID,
			QuestionID:    d.QuestionID,
			Operator:      string(d.Operator),
			ConditionType: string(d.ConditionType),
			AnswerValue:   d.AnswerValue,
			Sequence:      d.Sequence,
		})
	}
	return out
}

func toRestrictionDeterminantValues(list []*entity.RestrictionDeterminant) []response.RestrictionDeterminantResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.RestrictionDeterminantResponse, 0, len(list))
	for _, d := range list {
		out = append(out, response.RestrictionDeterminantResponse{
			ID:            d.ID,
			RestrictionID: d.RestrictionID,
			QuestionID:    d.QuestionID,
			Operator:      string(d.Operator),
			AnswerValue:   d.AnswerValue,
		})
	}
	return out
}

func toRestrictionReasonResponse(r *entity.RestrictionReason) *response.RestrictionReasonResponse {
	if r == nil {
		return nil
	}
	return &response.RestrictionReasonResponse{
		ID:          r.ID,
		Code:        r.Code,
		Description: r.Description,
		Situation:   r.Situation,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toRestrictionReasonResponses(list []*entity.RestrictionReason) []*response.RestrictionReasonResponse {
	out := make([]*response.RestrictionReasonResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toRestrictionReasonResponse(r))
	}
	return out
}
