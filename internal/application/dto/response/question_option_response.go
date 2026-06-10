package response

import "github.com/google/uuid"

// QuestionOptionResponse is the API representation of a question option.
type QuestionOptionResponse struct {
	ID         int64     `json:"id"`
	QuestionID int64     `json:"question_id"`
	Value      string    `json:"value"`
	CreatedBy  uuid.UUID `json:"created_by"`
}
