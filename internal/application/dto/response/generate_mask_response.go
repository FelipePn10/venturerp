package response

import (
	"time"

	"github.com/google/uuid"
)

// ItemMaskResponse is the API representation of a generated item mask.
type ItemMaskResponse struct {
	ID        int64                `json:"id"`
	ItemCode  int64                `json:"item_code"`
	Mask      string               `json:"mask"`
	MaskHash  string               `json:"mask_hash"`
	CreatedBy uuid.UUID            `json:"created_by"`
	CreatedAt time.Time            `json:"created_at"`
	Answers   []MaskAnswerResponse `json:"answers,omitempty"`
}

// MaskAnswerResponse is the API representation of a single mask answer.
type MaskAnswerResponse struct {
	QuestionID  int64  `json:"question_id"`
	OptionID    int64  `json:"option_id"`
	OptionValue string `json:"option_value"`
	Position    int    `json:"position"`
}
