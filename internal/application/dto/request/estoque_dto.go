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
	Code        int64 `json:"code,omitempty"`
	WarehouseID int64 `json:"warehouse_id"`
	// DestinationWarehouseID faz da requisição uma transferência para este almoxarifado.
	DestinationWarehouseID *int64                      `json:"destination_warehouse_id,omitempty"`
	RequesterName          *string                     `json:"requester_name,omitempty"`
	RequesterEmployeeCode  *int64                      `json:"requester_employee_code,omitempty"`
	CostCenterCode         *int32                      `json:"cost_center_code,omitempty"`
	NeededDate             *string                     `json:"needed_date,omitempty"`
	Notes                  *string                     `json:"notes,omitempty"`
	Items                  []RequisicaoMaterialItemDTO `json:"items"`
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
	// DestinationAddress: endereço de chegada, na requisição de transferência.
	DestinationAddress *string `json:"destination_address,omitempty"`
	// ViaTransit manda a entrega para o trânsito, em vez de transferir direto.
	ViaTransit bool `json:"via_transit,omitempty"`
	// TransitWarehouseID: qual trânsito usar, quando a empresa tem mais de um.
	TransitWarehouseID int64 `json:"transit_warehouse_id,omitempty"`
}

// EnvioTransitoDTO: transferência em duas etapas.
type EnvioTransitoDTO struct {
	ItemCode           int64   `json:"item_code"`
	Mask               string  `json:"mask"`
	FromWarehouseID    int64   `json:"from_warehouse_id"`
	ToWarehouseID      int64   `json:"to_warehouse_id"`
	TransitWarehouseID int64   `json:"transit_warehouse_id,omitempty"`
	Quantity           string  `json:"quantity"`
	Lot                *string `json:"lot,omitempty"`
	FromAddress        *string `json:"from_address,omitempty"`
	ExpectedDate       *string `json:"expected_date,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// BaixaTransitoDTO: action é RECEBER, DEVOLVER ou PERDA.
type BaixaTransitoDTO struct {
	Action   string  `json:"action"`
	Quantity string  `json:"quantity"`
	Address  *string `json:"address,omitempty"`
	Reason   string  `json:"reason,omitempty"`
}

// PoliticaReposicaoDTO: mínimo e máximo do item em um almoxarifado. Valor vazio
// herda do cadastro do item.
type PoliticaReposicaoDTO struct {
	ItemCode          int64   `json:"item_code"`
	WarehouseID       int64   `json:"warehouse_id"`
	MinimumQty        *string `json:"minimum_qty,omitempty"`
	MaximumQty        *string `json:"maximum_qty,omitempty"`
	SafetyQty         *string `json:"safety_qty,omitempty"`
	SourceType        string  `json:"source_type"`
	SourceWarehouseID *int64  `json:"source_warehouse_id,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

// GerarReposicaoDTO: sugestões aceitas, que viram requisição de compra, ordem de
// reposição ou ordem planejada conforme a origem de cada linha.
type GerarReposicaoDTO struct {
	Notes *string             `json:"notes,omitempty"`
	Lines []LinhaReposicaoDTO `json:"lines"`
}

type LinhaReposicaoDTO struct {
	ItemCode          int64  `json:"item_code"`
	WarehouseID       *int64 `json:"warehouse_id,omitempty"`
	SourceType        string `json:"source_type"`
	SourceWarehouseID *int64 `json:"source_warehouse_id,omitempty"`
	Quantity          string `json:"quantity"`
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
