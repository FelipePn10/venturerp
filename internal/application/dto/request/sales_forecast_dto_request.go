package request

type CreateSalesForecastDTO struct {
	ItemCode int64   `json:"item_code"`
	Mask     *string `json:"mask,omitempty"`
	Week     int     `json:"week"`
	Year     int     `json:"year"`
	Quantity float64 `json:"quantity"`
}

type UpdateSalesForecastDTO struct {
	ID       int64   `json:"id"`
	Quantity float64 `json:"quantity"`
}

type CreateForecastBlockDTO struct {
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Reason    *string `json:"reason,omitempty"`
}

type CreateAppropriationTableDTO struct {
	Description  string  `json:"description"`
	MondayPct    float64 `json:"monday_pct"`
	TuesdayPct   float64 `json:"tuesday_pct"`
	WednesdayPct float64 `json:"wednesday_pct"`
	ThursdayPct  float64 `json:"thursday_pct"`
	FridayPct    float64 `json:"friday_pct"`
	SaturdayPct  float64 `json:"saturday_pct"`
	SundayPct    float64 `json:"sunday_pct"`
	IsDefault    bool    `json:"is_default"`
}

type UpdateAppropriationTableDTO struct {
	ID           int64   `json:"id"`
	Description  string  `json:"description"`
	MondayPct    float64 `json:"monday_pct"`
	TuesdayPct   float64 `json:"tuesday_pct"`
	WednesdayPct float64 `json:"wednesday_pct"`
	ThursdayPct  float64 `json:"thursday_pct"`
	FridayPct    float64 `json:"friday_pct"`
	SaturdayPct  float64 `json:"saturday_pct"`
	SundayPct    float64 `json:"sunday_pct"`
}

type SetDefaultAppropriationDTO struct {
	ID int64 `json:"id"`
}
