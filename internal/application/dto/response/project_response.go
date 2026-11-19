package response

// ProjectResponse: projeto com o cliente e as ordens de fabricação ligadas.
type ProjectResponse struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	CustomerCode *int64 `json:"customer_code,omitempty"`
	CustomerName string `json:"customer_name,omitempty"`
	Status       string `json:"status"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	Notes        string `json:"notes,omitempty"`
	OpenOrders   int    `json:"open_orders"`
	TotalOrders  int    `json:"total_orders"`
}
