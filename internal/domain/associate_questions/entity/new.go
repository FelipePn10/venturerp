package entity

import (
	"errors"
	"time"
)

var (
	ErrInvalidPosition = errors.New("position must be greater than zero")
)

func New(
	item_code int64,
	questionId int64,
	position int,
) (*AssociateQuestion, error) {
	if position <= 0 {
		return nil, ErrInvalidPosition
	}

	return &AssociateQuestion{
		ItemCode:   item_code,
		QuestionID: questionId,
		Position:   position,
		CreatedAt:  time.Now(),
	}, nil
}
