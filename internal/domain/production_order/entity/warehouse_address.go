package entity

type WarehouseAddress struct {
	ID          int64  `json:"id"`
	WarehouseID int64  `json:"warehouse_id"`
	Address     string `json:"address"`
	IsActive    bool   `json:"is_active"`
}
