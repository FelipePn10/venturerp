package entity

import "time"

type IndustrialCalendar struct {
	Year        int
	Month       int
	Day         int
	IsWorkday   bool
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
