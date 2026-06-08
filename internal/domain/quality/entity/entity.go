package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
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
	SampleSize       float64
	AcceptanceLevel  float64
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
	Nominal        *float64
	ToleranceUpper *float64
	ToleranceLower *float64
	Unit           *string
	IsCritical     bool
}

type QualityRecord struct {
	ID                int64
	PlanID            int64
	ProductionOrderID *int64
	Lot               *string
	ItemCode          int64
	InspectedQty      float64
	ApprovedQty       float64
	RejectedQty       float64
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
	MeasuredValue    float64
	IsConformant     bool
}

type NonConformance struct {
	ID                int64
	Code              int64
	QualityRecordID   *int64
	ProductionOrderID *int64
	ItemCode          int64
	Lot               *string
	NonConformQty     float64
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
	sampleSize, acceptanceLevel float64, createdBy uuid.UUID) (*InspectionPlan, error) {
	if itemCode <= 0 {
		return nil, errors.New("item_code must be positive")
	}
	if description == "" {
		return nil, errors.New("description is required")
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

func NewNonConformance(code, itemCode int64, description string, qty float64,
	severity NCSeverity, createdBy uuid.UUID) (*NonConformance, error) {
	if code <= 0 {
		return nil, errors.New("code must be positive")
	}
	if description == "" {
		return nil, errors.New("description is required")
	}
	if qty <= 0 {
		return nil, errors.New("nonconform_qty must be positive")
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
