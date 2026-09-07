package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

// ItemStructure representa um componente dentro de uma estrutura de produto (BOM).
//
// Regras de negócio:
//   - ParentMask == nil  → componente genérico: aplica-se a TODAS as configurações
//   - ParentMask != nil  → componente específico: aplica-se APENAS à máscara informada
//   - Um item não pode ser componente de si mesmo
//   - A adição de um componente não pode criar um ciclo na árvore
type ItemStructure struct {
	ID                 int64
	ParentCode         int64
	ChildCode          int64
	ChildDescription   string
	Inherit            bool
	ParentMask         *string // nil = genérico
	Quantity           float64
	LossPercentage     float64 // 0–100 (%)
	LossFormula        *string // expressão matemática com variáveis de perguntas; substitui LossPercentage quando avaliável
	QuantityFormula    *string // expressão que calcula a quantidade a partir das variáveis do configurador (ex.: 2*(COMPRIMENTO/1000))
	QuantityRounding   string  // NONE | UP | DOWN | NEAREST, aplicado ao resultado da fórmula
	QuantityScale      int16   // casas decimais do arredondamento (0–6)
	UnitOfMeasurement  types.TypeUnitOfMeasurementItem
	Health             types.Health
	Sequence           int
	Notes              *string
	StartDate          *time.Time // nil = sem restrição de início
	EndDate            *time.Time // nil = sem restrição de fim
	IsCoproduct        bool       // true = SAÍDA (co-produto/subproduto/sucata), não insumo
	IsFixedQty         bool       // true = quantidade por OF (lote), não por unidade do pai
	SubstituteGroup    int16      // >0 = grupo de substitutos (mesmo pai); 0 = standalone
	SubstitutePriority int16      // menor = preferido; o mínimo do grupo é o primário
	IsActive           bool
	// WarehouseCode define de onde o componente é baixado; sem ele vale o
	// almoxarifado do cadastro do item.
	WarehouseCode *int64
	// LineWarehouseCode é o almoxarifado de linha, junto ao operador.
	LineWarehouseCode *int64
	// SetupLoss é consumida uma vez por ordem (preparação da máquina), não por
	// unidade do pai.
	SetupLoss float64
	// A perda de custo é separada da de engenharia: a de engenharia dimensiona
	// a necessidade, a de custo entra no cálculo do custo.
	CostLossType   string // PERCENTUAL | QUANTIDADE
	CostLoss       float64
	CostCenterCode *int64
	// IsCriticalMPS coloca o componente na linha crítica do plano mestre.
	IsCriticalMPS       bool
	GeneratesInspection bool
	CreatedBy           uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ConsultRow estende ItemStructure com campos desnormalizados do item filho
// retornados pela query de consulta de estrutura.
type ConsultRow struct {
	*ItemStructure
	WarehouseCode int64
	TypeStruct    int16
}

// WhereUsedRow representa uma linha do resultado da implosão de estrutura.
type WhereUsedRow struct {
	Level             int
	ParentCode        int64
	ChildCode         int64
	ParentDescription string
	Quantity          float64
	LossPercentage    float64
	ParentMask        *string
	Sequence          int
}
