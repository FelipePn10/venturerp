package entity

// WarehouseAddress é um endereço de almoxarifado da manufatura. A tabela tem
// chave composta (empresa, almoxarifado, endereço) e não possui coluna id.
type WarehouseAddress struct {
	WarehouseID int64  `json:"warehouse_id"`
	Address     string `json:"address"`
	IsActive    bool   `json:"is_active"`
}
