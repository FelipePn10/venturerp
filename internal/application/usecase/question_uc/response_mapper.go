package question_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	assocentity "github.com/FelipePn10/panossoerp/internal/domain/associate_questions/entity"
	questionsentity "github.com/FelipePn10/panossoerp/internal/domain/questions/entity"
)

func toQuestionResponse(q *questionsentity.Question) *response.QuestionResponse {
	if q == nil {
		return nil
	}
	return &response.QuestionResponse{
		Name:      q.Name,
		CreatedBy: q.CreatedBy,
	}
}

func toAssociateQuestionDetailResponses(list []assocentity.AssociateQuestionDetail) []response.AssociateQuestionDetailResponse {
	out := make([]response.AssociateQuestionDetailResponse, 0, len(list))
	for _, d := range list {
		out = append(out, response.AssociateQuestionDetailResponse{
			ItemCode:     d.ItemCode,
			QuestionID:   d.QuestionID,
			QuestionName: d.QuestionName,
			Position:     d.Position,
			CreatedAt:    d.CreatedAt,
		})
	}
	return out
}

func toItemQuestionRowResponses(list []assocentity.ItemQuestionRow) []response.ItemQuestionRowResponse {
	out := make([]response.ItemQuestionRowResponse, 0, len(list))
	for _, r := range list {
		out = append(out, response.ItemQuestionRowResponse{
			ItemCode:         r.ItemCode,
			ItemBusinessCode: r.ItemBusinessCode,
			QuestionID:       r.QuestionID,
			QuestionName:     r.QuestionName,
			Position:         r.Position,
			CreatedAt:        r.CreatedAt,
		})
	}
	return out
}
