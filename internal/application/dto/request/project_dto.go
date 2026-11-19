package request

// ProjectDTO grava um projeto (criação e alteração). Datas em AAAA-MM-DD.
type ProjectDTO struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	CustomerCode *int64 `json:"customer_code,omitempty"`
	Status       string `json:"status,omitempty"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	Notes        string `json:"notes,omitempty"`
}
