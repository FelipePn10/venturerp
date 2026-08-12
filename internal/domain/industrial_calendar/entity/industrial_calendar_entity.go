package entity

import "time"

type IndustrialCalendar struct {
	EnterpriseID int64
	Year         int
	Month        int
	Day          int
	IsWorkday    bool
	Description  *string
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
