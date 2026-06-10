package response

import (
	"time"

	"github.com/google/uuid"
)

// SalesForecastResponse is the API representation of a weekly sales forecast.
type SalesForecastResponse struct {
	ID        int64     `json:"id"`
	ItemCode  int64     `json:"item_code"`
	Mask      *string   `json:"mask,omitempty"`
	Week      int       `json:"week"`
	Year      int       `json:"year"`
	Quantity  float64   `json:"quantity"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SalesForecastBlockResponse is the API representation of a forecast block period.
type SalesForecastBlockResponse struct {
	ID        int64     `json:"id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}

// AppropriationTableResponse is the API representation of a daily appropriation table.
type AppropriationTableResponse struct {
	ID           int64     `json:"id"`
	Description  string    `json:"description"`
	MondayPct    float64   `json:"monday_pct"`
	TuesdayPct   float64   `json:"tuesday_pct"`
	WednesdayPct float64   `json:"wednesday_pct"`
	ThursdayPct  float64   `json:"thursday_pct"`
	FridayPct    float64   `json:"friday_pct"`
	SaturdayPct  float64   `json:"saturday_pct"`
	SundayPct    float64   `json:"sunday_pct"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    uuid.UUID `json:"created_by"`
}
