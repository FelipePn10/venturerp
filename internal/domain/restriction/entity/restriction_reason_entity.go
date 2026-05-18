package entity

import "time"

type RestrictionReason struct {
	ID          int64
	Code        int64
	Description string
	Situation   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
