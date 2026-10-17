package request

// TransferenciaAlmoxarifadoDTO move material de um almoxarifado para outro.
type TransferenciaAlmoxarifadoDTO struct {
	ItemCode        int64   `json:"item_code"`
	Mask            string  `json:"mask"`
	FromWarehouseID int64   `json:"from_warehouse_id"`
	ToWarehouseID   int64   `json:"to_warehouse_id"`
	Quantity        string  `json:"quantity"`
	Lot             *string `json:"lot,omitempty"`
	FromAddress     *string `json:"from_address,omitempty"`
	ToAddress       *string `json:"to_address,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

// EstornoMovimentoDTO: o motivo é obrigatório (fica no histórico).
type EstornoMovimentoDTO struct {
	Reason string `json:"reason"`
}

// RequisicaoMaterialDTO cria a requisição ao almoxarifado.
type RequisicaoMaterialDTO struct {
	Code                  int64                       `json:"code,omitempty"`
	WarehouseID           int64                       `json:"warehouse_id"`
	RequesterName         *string                     `json:"requester_name,omitempty"`
	RequesterEmployeeCode *int64                      `json:"requester_employee_code,omitempty"`
	CostCenterCode        *int32                      `json:"cost_center_code,omitempty"`
	NeededDate            *string                     `json:"needed_date,omitempty"`
	Notes                 *string                     `json:"notes,omitempty"`
	Items                 []RequisicaoMaterialItemDTO `json:"items"`
}

type RequisicaoMaterialItemDTO struct {
	ItemCode int64   `json:"item_code"`
	Mask     string  `json:"mask"`
	Quantity string  `json:"quantity"`
	Notes    *string `json:"notes,omitempty"`
}

// AtendimentoRequisicaoDTO: o que entregar de cada linha.
type AtendimentoRequisicaoDTO struct {
	Lines []AtendimentoLinhaDTO `json:"lines"`
}

type AtendimentoLinhaDTO struct {
	ItemID   int64   `json:"item_id"`
	Quantity string  `json:"quantity"`
	Lot      *string `json:"lot,omitempty"`
	Address  *string `json:"address,omitempty"`
}

// ReposicaoParaRequisicaoDTO: itens da reposição que viram requisição de compra.
type ReposicaoParaRequisicaoDTO struct {
	Notes *string                      `json:"notes,omitempty"`
	Items []ReposicaoItemRequisicaoDTO `json:"items"`
}

type ReposicaoItemRequisicaoDTO struct {
	ItemCode int64   `json:"item_code"`
	Quantity float64 `json:"quantity"`
}
