package request

import "github.com/google/uuid"

type GenerateMaskItemRequestDTO struct {
	ItemCode     int64             `json:"item_code"`
	CustomerCode *int64            `json:"customer_code"`
	DivisionID   *int64            `json:"division_id"`
	Answers      []MaskAnswerInput `json:"answers"`
	CreatedBy    uuid.UUID         `json:"created_by"`
}

type MaskAnswerInput struct {
	QuestionID int64 `json:"question_id"`
	OptionID   int64 `json:"option_id"`
	Position   int   `json:"position"`
}
