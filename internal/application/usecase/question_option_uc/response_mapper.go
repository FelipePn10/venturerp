package question_option_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/questions_options/entity"
)

func toQuestionOptionResponse(o *entity.QuestionsOptions) *response.QuestionOptionResponse {
	if o == nil {
		return nil
	}
	return &response.QuestionOptionResponse{
		ID:         o.ID,
		QuestionID: o.QuestionId,
		Value:      o.Value,
		CreatedBy:  o.CreatedBy,
	}
}

func toQuestionOptionResponsesFromValues(list []entity.QuestionsOptions) []*response.QuestionOptionResponse {
	out := make([]*response.QuestionOptionResponse, 0, len(list))
	for i := range list {
		out = append(out, toQuestionOptionResponse(&list[i]))
	}
	return out
}
