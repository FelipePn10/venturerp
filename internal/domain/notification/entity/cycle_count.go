package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CycleCountState string
type CycleCountOrigin string

const (
	CycleScheduled CycleCountState = "PROGRAMADA"
	CycleCounting  CycleCountState = "EM_CONTAGEM"
	CycleDivergent CycleCountState = "DIVERGENTE"
	CycleCompleted CycleCountState = "CONCLUIDA"
	CycleApproved  CycleCountState = "APROVADA"
	CycleCancelled CycleCountState = "CANCELADA"
)

const (
	CycleOriginManual     CycleCountOrigin = "MANUAL"
	CycleOriginItemPolicy CycleCountOrigin = "POLITICA_ITEM"
)

type CycleCount struct {
	ID                 uuid.UUID        `json:"id"`
	EnterpriseID       int64            `json:"enterprise_id"`
	WarehouseID        int64            `json:"warehouse_id"`
	WarehouseAddressID *int64           `json:"warehouse_address_id,omitempty"`
	ItemCode           string           `json:"item_code"`
	LegacyItemCode     int64            `json:"legacy_item_code,omitempty"`
	Mask               string           `json:"mask"`
	LotCode            string           `json:"lot_code"`
	ScheduledFor       time.Time        `json:"scheduled_for"`
	State              CycleCountState  `json:"state"`
	Origin             CycleCountOrigin `json:"origin"`
	PolicyDays         *int             `json:"policy_days,omitempty"`
	ExpectedQuantity   *decimal.Decimal `json:"expected_quantity,omitempty"`
	CountedQuantity    *decimal.Decimal `json:"counted_quantity,omitempty"`
	DivergenceQuantity *decimal.Decimal `json:"divergence_quantity,omitempty"`
	CountedBy          *uuid.UUID       `json:"counted_by,omitempty"`
	ApprovedBy         *uuid.UUID       `json:"approved_by,omitempty"`
	StartedAt          *time.Time       `json:"started_at,omitempty"`
	CompletedAt        *time.Time       `json:"completed_at,omitempty"`
	ApprovedAt         *time.Time       `json:"approved_at,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

func (c CycleCount) ValidateSchedule() error {
	if c.EnterpriseID <= 0 || c.WarehouseID <= 0 || strings.TrimSpace(c.ItemCode) == "" {
		return errors.New("empresa, almoxarifado e item são obrigatórios")
	}
	if c.ScheduledFor.IsZero() {
		return errors.New("data programada é obrigatória")
	}
	return nil
}
func CanTransition(from, to CycleCountState) bool {
	allowed := map[CycleCountState]map[CycleCountState]bool{CycleScheduled: {CycleCounting: true, CycleCancelled: true}, CycleCounting: {CycleDivergent: true, CycleCompleted: true, CycleCancelled: true}, CycleDivergent: {CycleCompleted: true, CycleCancelled: true}, CycleCompleted: {CycleApproved: true}, CycleApproved: {}, CycleCancelled: {}}
	return allowed[from][to]
}
