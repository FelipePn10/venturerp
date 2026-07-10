// Package entity holds the Drawing register (Cadastro de Desenhos) with revisions.
package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Drawing is a drawing header; its identity is code+digit+format, revised over
// time by DrawingRevision rows.
type Drawing struct {
	ID           int64
	Code         string
	Digit        string
	Format       string
	Model        string
	ItemCode     *int64
	Description  string
	UOM          string
	Weight       *float64
	MaterialSpec string // E.M.
	CreationDate *time.Time
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
	Revisions    []DrawingRevision
}

type DrawingRevision struct {
	ID            int64
	DrawingID     int64
	Revision      string
	StartDate     *time.Time
	EndDate       *time.Time
	MaterialSpec  string
	Reason        string
	ApprovedBy    string
	ApprovalDate  *time.Time
	IsCurrent     bool
	CreatedAt     time.Time
	Distributions []DrawingDistribution
}

type DrawingDistribution struct {
	ID            int64
	RevisionID    int64
	Recipient     string
	DistributedAt *time.Time
	Notes         string
}

// DrawingCharacteristic links a drawing to a configurator characteristic/variable.
type DrawingCharacteristic struct {
	ID               int64
	DrawingID        int64
	CharacteristicID int64
	Operator         string
	VariableID       *int64
}

func NewDrawing(code, digit, format string, createdBy uuid.UUID) (*Drawing, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("código do desenho é obrigatório")
	}
	return &Drawing{
		Code: code, Digit: digit, Format: format, IsActive: true, CreatedBy: createdBy,
	}, nil
}

// CompositeCode is the replication key: Desenho(first 20) + Dígito + Formato +
// Revisão. Used when replicating the change to the item's engineering tab.
func (d *Drawing) CompositeCode(revision string) string {
	code := d.Code
	if len(code) > 20 {
		code = code[:20]
	}
	return code + d.Digit + d.Format + revision
}

func (r *DrawingRevision) Validate() error {
	if strings.TrimSpace(r.Revision) == "" {
		return errors.New("código da revisão é obrigatório")
	}
	if r.StartDate != nil && r.EndDate != nil && r.EndDate.Before(*r.StartDate) {
		return errors.New("data fim não pode ser anterior à data início")
	}
	return nil
}
