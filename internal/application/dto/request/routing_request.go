package request

// ─── operations ──────────────────────────────────────────────────────────────

type CreateOperationDTO struct {
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	Origin              string  `json:"origin"` // INTERNA | EXTERNA | TERCEIROS
	DefaultWorkCenterID *int64  `json:"default_work_center_id,omitempty"`
	StandardTime        float64 `json:"standard_time"` // legacy flat time (falls back to run_time)
	SetupTime           float64 `json:"setup_time"`    // setup per lot, in time_unit

	// Rich time model (defaults). All in TimeUnit (MIN|HORA|DIA).
	RunTime    float64 `json:"run_time"`     // machine time per run_base_qty
	LaborTime  float64 `json:"labor_time"`   // labor time per run_base_qty (0 ⇒ equals run)
	RunBaseQty float64 `json:"run_base_qty"` // pieces per run cycle (>=1)
	QueueTime  float64 `json:"queue_time"`   // fixed per lot
	WaitTime   float64 `json:"wait_time"`    // fixed per lot
	MoveTime   float64 `json:"move_time"`    // fixed per lot
	CrewSize   float64 `json:"crew_size"`    // simultaneous operators (>=1)
	TimeUnit   string  `json:"time_unit"`    // MIN | HORA | DIA (default HORA)

	// Subcontracting (EXTERNA / TERCEIROS).
	SupplierID           *int64    `json:"supplier_id,omitempty"`
	ServiceItemCode      *TextCode `json:"service_item_code,omitempty"`
	CostPerUnit          *float64  `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32    `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance string    `json:"third_party_remittance,omitempty"`

	// ScrapPct é o refugo padrão da operação, em %.
	ScrapPct float64 `json:"scrap_pct"`
}

type UpdateOperationDTO struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	Origin              string  `json:"origin"`
	Situation           string  `json:"situation"` // APROVADA | INATIVA
	DefaultWorkCenterID *int64  `json:"default_work_center_id,omitempty"`
	StandardTime        float64 `json:"standard_time"`
	SetupTime           float64 `json:"setup_time"`

	RunTime    float64 `json:"run_time"`
	LaborTime  float64 `json:"labor_time"`
	RunBaseQty float64 `json:"run_base_qty"`
	QueueTime  float64 `json:"queue_time"`
	WaitTime   float64 `json:"wait_time"`
	MoveTime   float64 `json:"move_time"`
	CrewSize   float64 `json:"crew_size"`
	TimeUnit   string  `json:"time_unit"`

	SupplierID           *int64    `json:"supplier_id,omitempty"`
	ServiceItemCode      *TextCode `json:"service_item_code,omitempty"`
	CostPerUnit          *float64  `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32    `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance string    `json:"third_party_remittance,omitempty"`

	// ScrapPct é o refugo padrão da operação, em %.
	ScrapPct float64 `json:"scrap_pct"`
}

// ─── routes ──────────────────────────────────────────────────────────────────

type CreateRouteDTO struct {
	ItemCode    TextCode      `json:"item_code"`
	Mask        *string       `json:"mask,omitempty"`
	Alternative int16         `json:"alternative"`
	Description *string       `json:"description,omitempty"`
	IsStandard  bool          `json:"is_standard"`
	ValidFrom   *DataFlexivel `json:"valid_from,omitempty"` // nil = vale desde sempre
	ValidTo     *DataFlexivel `json:"valid_to,omitempty"`   // nil = sem fim
}

type UpdateRouteDTO struct {
	ID          int64         `json:"id"`
	Description *string       `json:"description,omitempty"`
	Situation   string        `json:"situation"` // APROVADA | INATIVA
	IsStandard  bool          `json:"is_standard"`
	ValidFrom   *DataFlexivel `json:"valid_from,omitempty"`
	ValidTo     *DataFlexivel `json:"valid_to,omitempty"`
}

// ─── route operations ─────────────────────────────────────────────────────────

type AddRouteOperationDTO struct {
	RouteID      int64    `json:"route_id"`
	Sequence     int16    `json:"sequence"`
	OperationID  int64    `json:"operation_id"`
	WorkCenterID *int64   `json:"work_center_id,omitempty"`
	StandardTime *float64 `json:"standard_time,omitempty"`
	SetupTime    *float64 `json:"setup_time,omitempty"`
	// Rich time-model overrides (nil ⇒ inherit from the operation).
	RunTime    *float64 `json:"run_time,omitempty"`
	LaborTime  *float64 `json:"labor_time,omitempty"`
	RunBaseQty *float64 `json:"run_base_qty,omitempty"`
	QueueTime  *float64 `json:"queue_time,omitempty"`
	WaitTime   *float64 `json:"wait_time,omitempty"`
	MoveTime   *float64 `json:"move_time,omitempty"`
	CrewSize   *float64 `json:"crew_size,omitempty"`
	TimeUnit   *string  `json:"time_unit,omitempty"`
	// Subcontracting overrides (nil ⇒ inherit from the operation).
	SupplierID           *int64    `json:"supplier_id,omitempty"`
	ServiceItemCode      *TextCode `json:"service_item_code,omitempty"`
	CostPerUnit          *float64  `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32    `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance *string   `json:"third_party_remittance,omitempty"`
	// ScrapPct sobrepõe o refugo da operação (nil ⇒ herda).
	ScrapPct *float64 `json:"scrap_pct,omitempty"`
	// InspectionRequired marca a etapa como ponto de inspeção.
	InspectionRequired bool    `json:"inspection_required"`
	Situation          string  `json:"situation"` // APROVADA | INATIVA | FANTASMA
	Notes              *string `json:"notes,omitempty"`
}

type UpdateRouteOperationDTO struct {
	ID                   int64     `json:"id"`
	WorkCenterID         *int64    `json:"work_center_id,omitempty"`
	StandardTime         *float64  `json:"standard_time,omitempty"`
	SetupTime            *float64  `json:"setup_time,omitempty"`
	RunTime              *float64  `json:"run_time,omitempty"`
	LaborTime            *float64  `json:"labor_time,omitempty"`
	RunBaseQty           *float64  `json:"run_base_qty,omitempty"`
	QueueTime            *float64  `json:"queue_time,omitempty"`
	WaitTime             *float64  `json:"wait_time,omitempty"`
	MoveTime             *float64  `json:"move_time,omitempty"`
	CrewSize             *float64  `json:"crew_size,omitempty"`
	TimeUnit             *string   `json:"time_unit,omitempty"`
	SupplierID           *int64    `json:"supplier_id,omitempty"`
	ServiceItemCode      *TextCode `json:"service_item_code,omitempty"`
	CostPerUnit          *float64  `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32    `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance *string   `json:"third_party_remittance,omitempty"`
	// ScrapPct sobrepõe o refugo da operação (nil ⇒ herda).
	ScrapPct *float64 `json:"scrap_pct,omitempty"`
	// InspectionRequired marca a etapa como ponto de inspeção.
	InspectionRequired bool    `json:"inspection_required"`
	Situation          string  `json:"situation"`
	Notes              *string `json:"notes,omitempty"`
}

// ─── alternative resources ────────────────────────────────────────────────────

type AddRouteOpResourceDTO struct {
	RouteOperationID int64   `json:"route_operation_id"`
	WorkCenterID     int64   `json:"work_center_id"`
	Priority         int16   `json:"priority"`    // 1 = most preferred
	TimeFactor       float64 `json:"time_factor"` // scales op time (1.0 = base); default 1
	IsPrimary        bool    `json:"is_primary"`  // when true, becomes the CT used by cost/CRP/lead-time
}

type UpdateRouteOpResourceDTO struct {
	ID         int64   `json:"id"`
	Priority   int16   `json:"priority"`
	TimeFactor float64 `json:"time_factor"`
}

// ─── network ─────────────────────────────────────────────────────────────────

type SetNetworkEdgeDTO struct {
	RouteID       int64   `json:"-"`
	PredecessorID int64   `json:"predecessor_id"`
	SuccessorID   int64   `json:"successor_id"`
	OverlapPct    float64 `json:"overlap_pct"`
}

type DeleteNetworkEdgeDTO struct {
	PredecessorID int64 `json:"predecessor_id"`
	SuccessorID   int64 `json:"successor_id"`
}

// ─── documentos de operação ───────────────────────────────────────────────────

// CreateOperationDocumentDTO anexa desenho, instrução ou ficha de processo.
// Exatamente um dos dois vínculos é preenchido: OperationID quando o documento
// vale em todo roteiro que usa a operação, RouteOperationID quando é daquela
// etapa (o desenho do item).
type CreateOperationDocumentDTO struct {
	OperationID      *int64  `json:"operation_id,omitempty"`
	RouteOperationID *int64  `json:"route_operation_id,omitempty"`
	Kind             string  `json:"kind"` // DESENHO | INSTRUCAO | FICHA | NORMA | FOTO | OUTRO
	Title            string  `json:"title"`
	Reference        *string `json:"reference,omitempty"` // caminho, URL ou código no controle de documentos
	Revision         *string `json:"revision,omitempty"`
	Instructions     *string `json:"instructions,omitempty"`
}

type UpdateOperationDocumentDTO struct {
	ID           int64   `json:"id"`
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Reference    *string `json:"reference,omitempty"`
	Revision     *string `json:"revision,omitempty"`
	Instructions *string `json:"instructions,omitempty"`
}

// ─── ponto de inspeção do roteiro ─────────────────────────────────────────────

// CreateRouteInspectionDTO cria o plano de inspeção já amarrado a uma etapa do
// roteiro, para quem desenha o processo não precisar sair da tela.
type CreateRouteInspectionDTO struct {
	RouteOperationID int64   `json:"route_operation_id"`
	PointType        string  `json:"point_type"` // RECEBIMENTO | PROCESSO | EXPEDICAO
	Description      string  `json:"description"`
	SampleSize       float64 `json:"sample_size"`
	AcceptanceLevel  float64 `json:"acceptance_level"`
	Instructions     *string `json:"instructions,omitempty"`
}
