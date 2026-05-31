package entity

import "time"

type ItemClassificationMask struct {
	ID          int64
	Code        int64
	Mask        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

type ItemClassification struct {
	ID          int64
	Code        string
	MaskID      int64
	ParentID    *int64
	Level       int
	Description string
	IsActive    bool
	CreatedAt   time.Time
}
