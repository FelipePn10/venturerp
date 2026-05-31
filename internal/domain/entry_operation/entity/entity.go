package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── State Group ───────────────────────────────────────────────────────────────

type StateGroup struct {
	ID          int64
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
	UFs         []string
}

func NewStateGroup(code int64, description string, createdBy uuid.UUID) (*StateGroup, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	return &StateGroup{
		Code:        code,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}, nil
}

// ─── Entry Operation Type ──────────────────────────────────────────────────────

type EntryOperationType struct {
	ID                 int64
	Code               int64
	Description        string
	InvoiceTypeCode    *int64
	NatureOperation    string
	ClassificationType *string
	ClassificationCode *string
	StateGroupCode     *int64
	SupplierTypeCode   *int64
	IsActive           bool
	CreatedAt          time.Time
	CreatedBy          uuid.UUID
}

func NewEntryOperationType(code int64, description, natureOperation string, createdBy uuid.UUID) (*EntryOperationType, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	natureOperation = strings.TrimSpace(natureOperation)
	if natureOperation == "" {
		return nil, fmt.Errorf("nature_operation is required")
	}
	switch natureOperation[0] {
	case '1', '2', '3':
	default:
		return nil, fmt.Errorf("nature_operation deve iniciar por 1 (dentro do estado), 2 (fora do estado) ou 3 (fora do país)")
	}
	return &EntryOperationType{
		Code:            code,
		Description:     description,
		NatureOperation: natureOperation,
		IsActive:        true,
		CreatedAt:       time.Now(),
		CreatedBy:       createdBy,
	}, nil
}

// ValidateUF applies the UF × Grupo de Estado rule based on the nature's first
// digit. ufInGroup tells whether the enterprise UF belongs to the operation's
// state group.
func (o *EntryOperationType) ValidateUF(enterpriseUF string, ufInGroup bool) error {
	if o.NatureOperation == "" {
		return nil
	}
	switch o.NatureOperation[0] {
	case '1': // dentro do estado → UF deve pertencer ao grupo
		if !ufInGroup {
			return fmt.Errorf("natureza 1 (dentro do estado): a UF %s da empresa deve pertencer ao grupo de estado da operação", enterpriseUF)
		}
	case '2': // fora do estado → UF NÃO deve pertencer ao grupo
		if ufInGroup {
			return fmt.Errorf("natureza 2 (fora do estado): a UF %s da empresa não deve pertencer ao grupo de estado da operação", enterpriseUF)
		}
	case '3': // fora do país → operação estrangeira (sem validação de grupo)
	}
	return nil
}
