package request

type AssociateByQuestionItemRequestDTO struct {
	ItemCode   int64 `json:"item_code"`
	QuestionID int64 `json:"question_id"`
	Position   int   `json:"position"`
}
