package response

import (
	"time"

	"github.com/google/uuid"
)

// RestrictionResponse is the API representation of a restriction.
type RestrictionResponse struct {
	ID                   int64                          `json:"id"`
	Code                 int64                          `json:"code"`
	Situation            string                         `json:"situation"`
	CustomerCode         *int64                         `json:"customer_code,omitempty"`
	ItemCode             *int64                         `json:"item_code,omitempty"`
	ReasonCode           *int64                         `json:"reason_code,omitempty"`
	ClassificationType   *string                        `json:"classification_type,omitempty"`
	ClassificationOrigin *string                        `json:"classification_origin,omitempty"`
	DivisionID           *int64                         `json:"division_id,omitempty"`
	Weight               int                            `json:"weight"`
	Dominants            []RestrictionDominantResponse  `json:"dominants,omitempty"`
	Determinants         []RestrictionDeterminantResponse `json:"determinants,omitempty"`
	CreatedAt            time.Time                      `json:"created_at"`
	UpdatedAt            time.Time                      `json:"updated_at"`
	CreatedBy            uuid.UUID                      `json:"created_by"`
}

// RestrictionDominantResponse is the API representation of a restriction dominant.
type RestrictionDominantResponse struct {
	ID            int64  `json:"id"`
	RestrictionID int64  `json:"restriction_id"`
	QuestionID    int64  `json:"question_id"`
	Operator      string `json:"operator"`
	ConditionType string `json:"condition_type"`
	AnswerValue   string `json:"answer_value"`
	Sequence      int    `json:"sequence"`
}

// RestrictionDeterminantResponse is the API representation of a restriction determinant.
type RestrictionDeterminantResponse struct {
	ID            int64   `json:"id"`
	RestrictionID int64   `json:"restriction_id"`
	QuestionID    int64   `json:"question_id"`
	Operator      string  `json:"operator"`
	AnswerValue   *string `json:"answer_value,omitempty"`
}

// RestrictionReasonResponse is the API representation of a restriction reason.
type RestrictionReasonResponse struct {
	ID          int64     `json:"id"`
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	Situation   string    `json:"situation"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
