package entity

import (
	"time"

	"github.com/google/uuid"
)

type PasswordChangeRequest struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	EnterpriseID    int64      `json:"enterprise_id"`
	UserName        string     `json:"user_name,omitempty"`
	UserEmail       string     `json:"user_email,omitempty"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
