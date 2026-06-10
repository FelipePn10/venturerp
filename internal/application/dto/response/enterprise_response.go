package response

import "github.com/google/uuid"

// EnterpriseResponse is the API representation of an enterprise.
type EnterpriseResponse struct {
	ID        int       `json:"id"`
	Code      int       `json:"code"`
	Name      string    `json:"name"`
	CreatedBy uuid.UUID `json:"created_by"`
}
