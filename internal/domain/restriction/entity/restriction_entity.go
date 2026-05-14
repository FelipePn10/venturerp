package entity

import (
	"time"

	"github.com/google/uuid"
)

type RestrictionSituation string

const (
	RestrictionActive   RestrictionSituation = "ACTIVE"
	RestrictionInactive RestrictionSituation = "INACTIVE"
)

type RestrictionOperator string

const (
	OperatorEqual      RestrictionOperator = "EQUAL"
	OperatorDifferent  RestrictionOperator = "DIFFERENT"
	OperatorGreater    RestrictionOperator = "GREATER"
	OperatorLess       RestrictionOperator = "LESS"
	OperatorBelongs    RestrictionOperator = "BELONGS"
	OperatorNotBelongs RestrictionOperator = "NOT_BELONGS"
	OperatorInvalid    RestrictionOperator = "INVALID"
)

type RestrictionCondition string

const (
	ConditionAnd RestrictionCondition = "AND"
	ConditionOr  RestrictionCondition = "OR"
)

type Restriction struct {
	ID                   int64
	Code                 int64
	Situation            RestrictionSituation
	ItemCode             *int64
	ReasonCode           *int64
	ClassificationType   *string
	ClassificationOrigin *string
	DivisionID           *int64
	Weight               int
	Dominants            []*RestrictionDominant
	Determinants         []*RestrictionDeterminant
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CreatedBy            uuid.UUID
}

type RestrictionDominant struct {
	ID            int64
	RestrictionID int64
	QuestionID    int64
	Operator      RestrictionOperator
	ConditionType RestrictionCondition
	AnswerValue   string
	Sequence      int
}

type RestrictionDeterminant struct {
	ID            int64
	RestrictionID int64
	QuestionID    int64
	Operator      RestrictionOperator
	AnswerValue   *string
}
