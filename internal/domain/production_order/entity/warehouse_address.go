package entity

// WarehouseAddress é um endereço de almoxarifado da manufatura. A tabela tem
// chave composta (empresa, almoxarifado, endereço) e não possui coluna id.
type WarehouseAddress struct {
	WarehouseID int64  `json:"warehouse_id"`
	Address     string `json:"address"`
	IsActive    bool   `json:"is_active"`
	// Atributos da migração 346. Sem devolvê-los, a tela lista endereços sem
	// saber a zona nem a ordem da rota — e o modal de seleção fica sem a única
	// informação que ajuda o conferente a se localizar no galpão.
	Zone         string   `json:"zone"`
	Capacity     *float64 `json:"capacity"`
	IsBlocked    bool     `json:"is_blocked"`
	BlockReason  *string  `json:"block_reason"`
	PickSequence int      `json:"pick_sequence"`
}
