package request

import "github.com/google/uuid"

type DominantDTO struct {
	QuestionID    int64  `json:"question_id"`
	Operator      string `json:"operator"`
	ConditionType string `json:"condition_type"` // AND, OR
	AnswerValue   string `json:"answer_value"`
	Sequence      int    `json:"sequence"`
}

type DeterminantDTO struct {
	QuestionID  int64   `json:"question_id"`
	Operator    string  `json:"operator"`
	AnswerValue *string `json:"answer_value"`
}

type CreateRestrictionDTO struct {
	Situation            string           `json:"situation"` // ACTIVE, INACTIVE
	ItemCode             *int64           `json:"item_code"`
	ReasonCode           *int64           `json:"reason_code"`
	ClassificationType   *string          `json:"classification_type"`
	ClassificationOrigin *string          `json:"classification_origin"`
	DivisionID           *int64           `json:"division_id"`
	Dominants            []DominantDTO    `json:"dominants"`
	Determinants         []DeterminantDTO `json:"determinants"`
	CreatedBy            uuid.UUID        `json:"created_by"`
}

type UpdateRestrictionDTO struct {
	Code                 int64   `json:"code"`
	Situation            string  `json:"situation"`
	ItemCode             *int64  `json:"item_code"`
	ReasonCode           *int64  `json:"reason_code"`
	ClassificationType   *string `json:"classification_type"`
	ClassificationOrigin *string `json:"classification_origin"`
	DivisionID           *int64  `json:"division_id"`
}
