package response

import (
	"time"

	"github.com/google/uuid"
)

// QuestionResponse is the API representation of a question.
type QuestionResponse struct {
	Name      string    `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`
}

// AssociateQuestionDetailResponse is the API representation of an item↔question association detail.
type AssociateQuestionDetailResponse struct {
	ItemCode     int64     `json:"item_code"`
	QuestionID   int64     `json:"question_id"`
	QuestionName string    `json:"question_name"`
	Position     int       `json:"position"`
	CreatedAt    time.Time `json:"created_at"`
}

// ItemQuestionRowResponse is the API representation of an item question listing row.
type ItemQuestionRowResponse struct {
	ItemCode         int64     `json:"item_code"`
	ItemBusinessCode int64     `json:"item_business_code"`
	QuestionID       int64     `json:"question_id"`
	QuestionName     string    `json:"question_name"`
	Position         int       `json:"position"`
	CreatedAt        time.Time `json:"created_at"`
}
