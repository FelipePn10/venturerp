package request

type CreateCalendarDayDTO struct {
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Day         int     `json:"day"`
	IsWorkday   bool    `json:"is_workday"`
	Description *string `json:"description,omitempty"`
}
