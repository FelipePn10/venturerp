package response

type LotMaskResponse struct {
	ID                 int64                 `json:"id"`
	Application        string                `json:"application"`
	CustomerCode       *int64                `json:"customer_code,omitempty"`
	ItemCode           *int64                `json:"item_code,omitempty"`
	ClassificationType string                `json:"classification_type,omitempty"`
	ClassificationCode *int64                `json:"classification_code,omitempty"`
	ZeroOnYearChange   bool                  `json:"zero_on_year_change"`
	IsActive           bool                  `json:"is_active"`
	Description        string                `json:"description,omitempty"`
	Parts              []LotMaskPartResponse `json:"parts,omitempty"`
}

type LotMaskPartResponse struct {
	ID               int64  `json:"id"`
	LotMaskID        int64  `json:"lot_mask_id"`
	Sequence         int    `json:"sequence"`
	PartType         string `json:"part_type"`
	Value            string `json:"value"`
	Size             int    `json:"size"`
	DateFormat       string `json:"date_format,omitempty"`
	ZeroOnYearChange bool   `json:"zero_on_year_change"`
	CurrentValue     string `json:"current_value,omitempty"`
	LastYear         *int   `json:"last_year,omitempty"`
}

type GeneratedLotResponse struct {
	LotMaskID int64  `json:"lot_mask_id"`
	Code      string `json:"code"`
}
