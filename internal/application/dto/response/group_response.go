package response

import "github.com/google/uuid"

// GroupResponse is the API representation of an item group.
type GroupResponse struct {
	ID           int32     `json:"id"`
	Code         int       `json:"code"`
	Description  string    `json:"description"`
	EnterpriseID int       `json:"enterprise_id"`
	CreatedBy    uuid.UUID `json:"created_by"`
}
