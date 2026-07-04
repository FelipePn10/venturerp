package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Life-type codes for tool useful-life tracking.
const (
	LifeStrokes = "GOLPES"
	LifeHours   = "HORAS"
	LifePieces  = "PECAS"
)

// Tool status codes.
const (
	StatusActive      = "ATIVA"
	StatusMaintenance = "MANUTENCAO"
	StatusInactive    = "INATIVA"
)

// Tool is a die, jig, fixture or cutting tool with useful-life tracking.
type Tool struct {
	ID        int64
	Code      int64
	Name      string
	ToolType  string
	LifeType  string  // GOLPES | HORAS | PECAS
	LifeLimit float64 // 0 = no life tracking
	LifeUsed  float64
	Cost      float64
	Status    string // ATIVA | MANUTENCAO | INATIVA
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID
}

// RemainingLife returns the life left before replacement; -1 when untracked.
func (t *Tool) RemainingLife() float64 {
	if t.LifeLimit <= 0 {
		return -1
	}
	r := t.LifeLimit - t.LifeUsed
	if r < 0 {
		r = 0
	}
	return r
}

// NeedsReplacement reports whether the tool has reached its life limit.
func (t *Tool) NeedsReplacement() bool {
	return t.LifeLimit > 0 && t.LifeUsed >= t.LifeLimit
}

func validLifeType(lt string) bool {
	switch lt {
	case LifeStrokes, LifeHours, LifePieces:
		return true
	default:
		return false
	}
}

func NewTool(code int64, name, toolType, lifeType string, lifeLimit, cost float64, createdBy uuid.UUID) (*Tool, error) {
	if code <= 0 {
		return nil, errors.New("tool code must be positive")
	}
	if name == "" {
		return nil, errors.New("tool name is required")
	}
	if lifeType == "" {
		lifeType = LifePieces
	}
	if !validLifeType(lifeType) {
		return nil, errors.New("life_type must be GOLPES, HORAS or PECAS")
	}
	if lifeLimit < 0 {
		return nil, errors.New("life_limit cannot be negative")
	}
	if toolType == "" {
		toolType = "FERRAMENTA"
	}
	return &Tool{
		Code:      code,
		Name:      name,
		ToolType:  toolType,
		LifeType:  lifeType,
		LifeLimit: lifeLimit,
		Cost:      cost,
		Status:    StatusActive,
		IsActive:  true,
		CreatedBy: createdBy,
	}, nil
}

// RouteOpTool links a tool required to run a route operation.
type RouteOpTool struct {
	ID               int64
	RouteOperationID int64
	ToolID           int64
	QtyRequired      float64

	// denormalized for reads
	ToolCode  int64
	ToolName  string
	LifeType  string
	LifeLimit float64
	LifeUsed  float64
	Status    string
}
