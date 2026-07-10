package request

import "github.com/google/uuid"

type LotMaskDTO struct {
	ID                 int64     `json:"id"`
	Application        string    `json:"application"`
	CustomerCode       *int64    `json:"customer_code"`
	ItemCode           *int64    `json:"item_code"`
	ClassificationType string    `json:"classification_type"`
	ClassificationCode *int64    `json:"classification_code"`
	ZeroOnYearChange   bool      `json:"zero_on_year_change"`
	Description        string    `json:"description"`
	CreatedBy          uuid.UUID `json:"-"`
}

type LotMaskPartDTO struct {
	ID               int64  `json:"id"`
	Sequence         int    `json:"sequence"`
	PartType         string `json:"part_type"` // CARACTER | DATA | SEQ_NUMERICA | SEQ_CARACTER
	Value            string `json:"value"`
	Size             int    `json:"size"`
	DateFormat       string `json:"date_format"`
	ZeroOnYearChange bool   `json:"zero_on_year_change"`
}

// GenerateLotDTO resolves which mask to use (explicit LotMaskID or by context)
// and produces a lot code, advancing the sequence state.
type GenerateLotDTO struct {
	LotMaskID          *int64 `json:"lot_mask_id"`
	Application        string `json:"application"`
	CustomerCode       *int64 `json:"customer_code"`
	ItemCode           *int64 `json:"item_code"`
	ClassificationCode *int64 `json:"classification_code"`
}
