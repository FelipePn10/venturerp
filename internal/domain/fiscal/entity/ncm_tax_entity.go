package entity

import "time"

type NcmTaxTable struct {
	ID          int64
	Ncm         string
	AliqIPI     float64
	AliqPis     float64
	AliqCofins  float64
	CstPis      string
	CstCofins   string
	CstIPI      string
	Description *string
	IsActive    bool
	CreatedAt   time.Time
}
