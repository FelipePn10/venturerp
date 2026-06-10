package response

import "github.com/google/uuid"

// ModifierResponse is the API representation of an item modifier.
type ModifierResponse struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
}
