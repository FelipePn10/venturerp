package request

import "github.com/shopspring/decimal"

type CreateSalesForecastDTO struct {
	ItemCode int64           `json:"item_code"`
	Mask     *string         `json:"mask,omitempty"`
	Week     int             `json:"week"`
	Year     int             `json:"year"`
	Quantity decimal.Decimal `json:"quantity"`
}

type UpdateSalesForecastDTO struct {
	ID       int64           `json:"id"`
	Quantity decimal.Decimal `json:"quantity"`
}

type CreateMonthlySalesForecastDTO struct {
	ItemCode        int64           `json:"item_code"`
	Mask            *string         `json:"mask,omitempty"`
	Year            int             `json:"year"`
	Month           int             `json:"month"`
	Quantity        decimal.Decimal `json:"quantity"`
	AcceptsFraction bool            `json:"accepts_fraction"`
	UpdateExisting  bool            `json:"update_existing"`
}

type CreateForecastBlockDTO struct {
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Reason    *string `json:"reason,omitempty"`
}

type CreateAppropriationTableDTO struct {
	Description  string          `json:"description"`
	MondayPct    decimal.Decimal `json:"monday_pct"`
	TuesdayPct   decimal.Decimal `json:"tuesday_pct"`
	WednesdayPct decimal.Decimal `json:"wednesday_pct"`
	ThursdayPct  decimal.Decimal `json:"thursday_pct"`
	FridayPct    decimal.Decimal `json:"friday_pct"`
	SaturdayPct  decimal.Decimal `json:"saturday_pct"`
	SundayPct    decimal.Decimal `json:"sunday_pct"`
	IsDefault    bool            `json:"is_default"`
}

type UpdateAppropriationTableDTO struct {
	ID           int64           `json:"id"`
	Description  string          `json:"description"`
	MondayPct    decimal.Decimal `json:"monday_pct"`
	TuesdayPct   decimal.Decimal `json:"tuesday_pct"`
	WednesdayPct decimal.Decimal `json:"wednesday_pct"`
	ThursdayPct  decimal.Decimal `json:"thursday_pct"`
	FridayPct    decimal.Decimal `json:"friday_pct"`
	SaturdayPct  decimal.Decimal `json:"saturday_pct"`
	SundayPct    decimal.Decimal `json:"sunday_pct"`
}

type SetDefaultAppropriationDTO struct {
	ID int64 `json:"id"`
}

type GenerateSalesForecastDTO struct {
	ItemCode        int64       `json:"item_code"`
	Mask            *string     `json:"mask,omitempty"`
	StartWeek       int         `json:"start_week"`
	StartYear       int         `json:"start_year"`
	History         []DataPoint `json:"history"`
	Periods         int         `json:"periods"`
	Model           string      `json:"model"`
	MAWindow        int         `json:"ma_window"`
	Alpha           float64     `json:"alpha"`
	Beta            float64     `json:"beta"`
	Gamma           float64     `json:"gamma"`
	SeasonLen       int         `json:"season_len"`
	UpdateExisting  bool        `json:"update_existing"`
	HistorySource   string      `json:"history_source,omitempty"`
	HistoryFrom     string      `json:"history_from,omitempty"`
	HistoryTo       string      `json:"history_to,omitempty"`
	TargetEndWeek   int         `json:"target_end_week,omitempty"`
	TargetEndYear   int         `json:"target_end_year,omitempty"`
	ProjectionPct   float64     `json:"projection_pct,omitempty"`
	AcceptsFraction bool        `json:"accepts_fraction"`
	ItemCodes       []int64     `json:"item_codes,omitempty"`
}

type DataPoint struct {
	Period   any     `json:"period"`
	Quantity float64 `json:"quantity"`
}
