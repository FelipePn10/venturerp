package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CentroDeCustoResponse: o centro do cadastro único com a gestão (VCTB0102).
type CentroDeCustoResponse struct {
	ID                int64      `json:"id"`
	Code              int32      `json:"code"`
	Codigo            string     `json:"codigo"`
	Descricao         string     `json:"descricao"`
	ParentCode        *int32     `json:"parent_code,omitempty"`
	Tipo              string     `json:"tipo"`
	IsRatio           bool       `json:"is_ratio"`
	StartDate         string     `json:"start_date"`
	EndDate           *string    `json:"end_date,omitempty"`
	IsActive          bool       `json:"is_active"`
	ResponsavelID     *uuid.UUID `json:"responsavel_id,omitempty"`
	Responsavel       string     `json:"responsavel,omitempty"`
	AprovadorID       *uuid.UUID `json:"aprovador_id,omitempty"`
	Aprovador         string     `json:"aprovador,omitempty"`
	AprovaRequisicoes bool       `json:"aprova_requisicoes"`
	ContaPadraoID     *int64     `json:"conta_padrao_id,omitempty"`
}

// OrcamentoCentroResponse: os 12 meses de um centro, orçado e realizado.
type OrcamentoCentroResponse struct {
	CostCenterID int64             `json:"cost_center_id"`
	Code         int32             `json:"code"`
	Codigo       string            `json:"codigo"`
	Descricao    string            `json:"descricao"`
	Orcado       []decimal.Decimal `json:"orcado"`
	Realizado    []decimal.Decimal `json:"realizado"`
}

// NoCentroResponse: um nó da árvore consolidada.
type NoCentroResponse struct {
	ID                   int64              `json:"id"`
	Code                 int32              `json:"code"`
	Codigo               string             `json:"codigo"`
	Descricao            string             `json:"descricao"`
	Tipo                 string             `json:"tipo"`
	IsActive             bool               `json:"is_active"`
	Responsavel          string             `json:"responsavel,omitempty"`
	Orcado               decimal.Decimal    `json:"orcado"`
	Realizado            decimal.Decimal    `json:"realizado"`
	OrcadoConsolidado    decimal.Decimal    `json:"orcado_consolidado"`
	RealizadoConsolidado decimal.Decimal    `json:"realizado_consolidado"`
	Estourado            bool               `json:"estourado"`
	Filhos               []NoCentroResponse `json:"filhos"`
}

// AprovacaoRequisicaoResponse: a aprovação da requisição pelo centro.
type AprovacaoRequisicaoResponse struct {
	ID              int64      `json:"id"`
	RequisitionCode int64      `json:"requisition_code"`
	CostCenterID    int64      `json:"cost_center_id"`
	CentroCodigo    string     `json:"centro_codigo"`
	CentroDescricao string     `json:"centro_descricao"`
	Status          string     `json:"status"`
	AprovadorID     uuid.UUID  `json:"aprovador_id"`
	DecididoEm      *time.Time `json:"decidido_em,omitempty"`
	Justificativa   *string    `json:"justificativa,omitempty"`
	CriadoEm        time.Time  `json:"criado_em"`
	PodeDecidir     bool       `json:"pode_decidir"`
}
