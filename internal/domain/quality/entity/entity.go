package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InspectionPointType string
type InspectionResult string
type NCSeverity string
type NCDisposition string

const (
	PointReceiving InspectionPointType = "RECEBIMENTO"
	PointInProcess InspectionPointType = "PROCESSO"
	PointShipping  InspectionPointType = "EXPEDICAO"

	ResultApproved    InspectionResult = "APROVADO"
	ResultRejected    InspectionResult = "REJEITADO"
	ResultConditional InspectionResult = "CONDICIONAL"
	ResultPending     InspectionResult = "PENDENTE"

	SeverityCritical    NCSeverity = "CRITICA"
	SeverityMajor       NCSeverity = "MAIOR"
	SeverityMinor       NCSeverity = "MENOR"
	SeverityObservation NCSeverity = "OBSERVACAO"

	DispositionScrap       NCDisposition = "SUCATA"
	DispositionRework      NCDisposition = "RETRABALHO"
	DispositionConditional NCDisposition = "APROVADO_CONDICIONAL"
	DispositionReturn      NCDisposition = "DEVOLVIDO"
)

type InspectionPlan struct {
	ID               int64
	ItemCode         int64
	RouteOperationID *int64
	PointType        InspectionPointType
	Description      string
	SampleSize       decimal.Decimal
	AcceptanceLevel  decimal.Decimal
	Instructions     *string
	Characteristics  []InspectionCharacteristic
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CreatedBy        uuid.UUID
}

type InspectionCharacteristic struct {
	ID             int64
	PlanID         int64
	Name           string
	Nominal        *decimal.Decimal
	ToleranceUpper *decimal.Decimal
	ToleranceLower *decimal.Decimal
	Unit           *string
	IsCritical     bool
}

type QualityRecord struct {
	ID                int64
	PlanID            int64
	ProductionOrderID *int64
	Lot               *string
	ItemCode          int64
	InspectedQty      decimal.Decimal
	ApprovedQty       decimal.Decimal
	RejectedQty       decimal.Decimal
	Result            InspectionResult
	InspectorID       *int64
	InspectedAt       time.Time
	Notes             *string
	Measurements      []QualityMeasurement
	CreatedAt         time.Time
	CreatedBy         uuid.UUID
}

type QualityMeasurement struct {
	ID               int64
	RecordID         int64
	CharacteristicID int64
	MeasuredValue    decimal.Decimal
	IsConformant     bool
}

type NonConformance struct {
	ID              int64
	Code            int64
	QualityRecordID *int64
	// LotID: lote de inspeção de origem.
	LotID             *int64
	ProductionOrderID *int64
	ItemCode          int64
	Lot               *string
	NonConformQty     decimal.Decimal
	Description       string
	Severity          NCSeverity
	RootCause         *string
	CorrectiveAction  *string
	Disposition       *NCDisposition
	DisposedAt        *time.Time
	DisposedBy        *uuid.UUID
	IsOpen            bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
}

func NewInspectionPlan(itemCode int64, pointType InspectionPointType, description string,
	sampleSize, acceptanceLevel decimal.Decimal, createdBy uuid.UUID) (*InspectionPlan, error) {
	if itemCode <= 0 {
		return nil, domainerr.NewValidation("o item deve ser um código maior que zero")
	}
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição")
	}
	return &InspectionPlan{
		ItemCode:        itemCode,
		PointType:       pointType,
		Description:     description,
		SampleSize:      sampleSize,
		AcceptanceLevel: acceptanceLevel,
		IsActive:        true,
		CreatedBy:       createdBy,
	}, nil
}

func NewNonConformance(code, itemCode int64, description string, qty decimal.Decimal,
	severity NCSeverity, createdBy uuid.UUID) (*NonConformance, error) {
	if code <= 0 {
		return nil, domainerr.NewValidation("o código deve ser maior que zero")
	}
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição")
	}
	if !qty.IsPositive() {
		return nil, domainerr.NewValidation("a quantidade não conforme deve ser maior que zero")
	}
	return &NonConformance{
		Code:          code,
		ItemCode:      itemCode,
		NonConformQty: qty,
		Description:   description,
		Severity:      severity,
		IsOpen:        true,
		CreatedBy:     createdBy,
	}, nil
}
