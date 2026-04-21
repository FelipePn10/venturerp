package request

type AssociateProductQuestionsRequest struct {
	ItemCode  int64 `json:"item_code"`
	Questions []struct {
		QuestionID int64 `json:"question_id"`
		Position   int   `json:"position"`
	} `json:"questions"`
}
