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
		return nil, errors.New("o código da ferramenta deve ser maior que zero")
	}
	if name == "" {
		return nil, errors.New("informe o nome da ferramenta")
	}
	if lifeType == "" {
		lifeType = LifePieces
	}
	if !validLifeType(lifeType) {
		return nil, errors.New("o tipo de vida útil deve ser golpes, horas ou peças")
	}
	if lifeLimit < 0 {
		return nil, errors.New("o limite de vida útil não pode ser negativo")
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

// Tool-serial status codes. A serial is one physical copy of a tool master.
const (
	SerialActive      = "ATIVA"
	SerialMaintenance = "MANUTENCAO"
	SerialInactive    = "INATIVA"
	SerialRetired     = "BAIXADA" // permanently written off
)

// ToolSerial is a physical instance (serial number) of a tool master. A tool
// may have several serials, each worn independently; the tool production sheet
// binds one serial per production-order operation.
type ToolSerial struct {
	ID           int64
	ToolID       int64
	SerialNumber string
	Status       string // ATIVA | MANUTENCAO | INATIVA | BAIXADA
	LifeUsed     float64
	Location     string
	Notes        string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID

	// denormalized for reads
	ToolCode int64
	ToolName string
}

func validSerialStatus(s string) bool {
	switch s {
	case SerialActive, SerialMaintenance, SerialInactive, SerialRetired:
		return true
	default:
		return false
	}
}

// NewToolSerial builds a validated tool serial for a given tool master.
func NewToolSerial(toolID int64, serialNumber, status, location, notes string, createdBy uuid.UUID) (*ToolSerial, error) {
	if toolID <= 0 {
		return nil, errors.New("a ferramenta deve ser um código maior que zero")
	}
	if serialNumber == "" {
		return nil, errors.New("informe o número de série")
	}
	if status == "" {
		status = SerialActive
	}
	if !validSerialStatus(status) {
		return nil, errors.New("a situação deve ser ativa, em manutenção, inativa ou baixada")
	}
	return &ToolSerial{
		ToolID:       toolID,
		SerialNumber: serialNumber,
		Status:       status,
		Location:     location,
		Notes:        notes,
		IsActive:     true,
		CreatedBy:    createdBy,
	}, nil
}

// Available reports whether the serial can be assigned to run an operation.
func (s *ToolSerial) Available() bool {
	return s.IsActive && s.Status == SerialActive
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
