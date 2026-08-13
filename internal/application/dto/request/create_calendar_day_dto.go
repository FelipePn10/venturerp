package request

type CreateCalendarDayDTO struct {
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Day         int     `json:"day"`
	IsWorkday   bool    `json:"is_workday"`
	Description *string `json:"description,omitempty"`
}

type GenerateIndustrialCalendarDTO struct {
	Year     int   `json:"year"`
	Month    *int  `json:"month,omitempty"`
	Weekdays []int `json:"weekdays,omitempty"`
}
