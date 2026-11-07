package entity

import "github.com/google/uuid"

type Group struct {
	ID           int32
	Code         int
	Description  string
	EnterpriseID int
	CreatedBy    uuid.UUID
	IsActive     bool
}
