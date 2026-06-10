package response

import "time"

// IndustrialCalendarResponse is the API representation of an industrial calendar day.
type IndustrialCalendarResponse struct {
	Year        int       `json:"year"`
	Month       int       `json:"month"`
	Day         int       `json:"day"`
	IsWorkday   bool      `json:"is_workday"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ItemCalendarPromiseResponse is the API representation of an item promise calendar day.
type ItemCalendarPromiseResponse struct {
	ID          int64     `json:"id"`
	ItemCode    int64     `json:"item_code"`
	Mask        string    `json:"mask"`
	Year        int       `json:"year"`
	Month       int       `json:"month"`
	Day         int       `json:"day"`
	IsWorkday   bool      `json:"is_workday"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
