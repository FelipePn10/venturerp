package entity

import "math"

// DefaultWorkingHoursPerDay mirrors machine/service.DefaultWorkingMinutesPerDay (480 min = 8 h)
// and is used to convert DIA-based times into hours.
const DefaultWorkingHoursPerDay = 8.0

// Time-unit codes stored in operations.time_unit / route_operations.time_unit.
const (
	// A ficha de fábrica vem em segundos; aceitar a unidade evita que o
	// usuário converta à mão e arredonde no caminho.
	TimeUnitSecond = "SEGUNDO"
	TimeUnitMinute = "MIN"
	TimeUnitHour   = "HORA"
	TimeUnitDay    = "DIA"
)

// hoursMul returns the multiplier that converts a value expressed in `unit` to hours.
func hoursMul(unit string) float64 {
	switch unit {
	case TimeUnitSecond:
		return 1.0 / 3600.0
	case TimeUnitMinute:
		return 1.0 / 60.0
	case TimeUnitDay:
		return DefaultWorkingHoursPerDay
	default: // HORA or empty
		return 1.0
	}
}

// TimeComponents holds an operation's default time model (in its own Unit).
type TimeComponents struct {
	Setup      float64
	Run        float64 // machine/processing time per RunBaseQty pieces
	Labor      float64 // direct-labor time per RunBaseQty pieces (0 ⇒ equals Run)
	RunBaseQty float64 // pieces covered by one Run cycle (>=1)
	Queue      float64 // fixed per lot
	Wait       float64 // fixed per lot
	Move       float64 // fixed per lot
	CrewSize   float64 // simultaneous operators (>=1)
	Unit       string  // MIN | HORA | DIA
}

// TempoDeMaquinaDaOperacao é o tempo de máquina padrão da operação de
// biblioteca: `run_time`, ou o tempo padrão LEGADO quando ele é zero — como a
// tela sempre prometeu ("usado só quando o tempo de máquina é zero"). Operações
// cadastradas antes do modelo de tempo têm só o legado; sem este fallback elas
// entravam com zero hora de máquina na OP, no custo e no prazo.
func TempoDeMaquinaDaOperacao(run, legado float64) float64 {
	if run <= 0 && legado > 0 {
		return legado
	}
	return run
}

// TimeOverrides holds a route-operation's per-component overrides.
// A nil pointer means "inherit the operation default".
type TimeOverrides struct {
	Setup      *float64
	Run        *float64
	Labor      *float64
	RunBaseQty *float64
	Queue      *float64
	Wait       *float64
	Move       *float64
	CrewSize   *float64
	Unit       *string
}

// OperationTime is the effective, resolved time model of a single operation,
// already merged from the route-operation overrides + operation defaults and
// normalised to HOURS. Setup/queue/wait/move are per lot; Run/Labor are per
// RunBaseQty pieces.
type OperationTime struct {
	Setup      float64 `json:"setup_hours"`
	Run        float64 `json:"run_hours"`   // machine time per RunBaseQty
	Labor      float64 `json:"labor_hours"` // labor time per RunBaseQty (0 ⇒ equals Run)
	RunBaseQty float64 `json:"run_base_qty"`
	Queue      float64 `json:"queue_hours"`
	Wait       float64 `json:"wait_hours"`
	Move       float64 `json:"move_hours"`
	CrewSize   float64 `json:"crew_size"`
	// UnitsPerPiece é quanto da BASE do tempo cada peça tem: a área em m² na
	// base M2, o comprimento em metros na METRO_LINEAR etc. (ver FatorDaBase).
	// O tempo de máquina vale por RunBaseQty unidades da base. Zero = 1 (por
	// peça), que é o caso de sempre.
	UnitsPerPiece float64 `json:"units_per_piece"`
	// SemMedida marca a etapa com base de tempo que a peça não informa (m² sem
	// comprimento e largura no item, atributo de PDM ausente). Ela fica sem
	// tempo de máquina — chutar 1 daria um tempo errado em silêncio — e a
	// prontidão do roteiro trava o planejamento até o cadastro ser completado.
	SemMedida bool `json:"missing_measure,omitempty"`
}

// ResolveOperationTime merges route-op overrides over operation defaults and
// converts every time component to hours. Overrides are interpreted in the
// override Unit (falling back to the operation Unit); defaults use the operation Unit.
func ResolveOperationTime(ov TimeOverrides, def TimeComponents) OperationTime {
	opMul := hoursMul(def.Unit)
	ovUnit := def.Unit
	if ov.Unit != nil && *ov.Unit != "" {
		ovUnit = *ov.Unit
	}
	ovMul := hoursMul(ovUnit)

	pick := func(o *float64, d float64) float64 {
		if o != nil {
			return *o * ovMul
		}
		return d * opMul
	}

	baseQty := def.RunBaseQty
	if ov.RunBaseQty != nil {
		baseQty = *ov.RunBaseQty
	}
	if baseQty <= 0 {
		baseQty = 1
	}

	crew := def.CrewSize
	if ov.CrewSize != nil {
		crew = *ov.CrewSize
	}
	if crew <= 0 {
		crew = 1
	}

	return OperationTime{
		Setup:      pick(ov.Setup, def.Setup),
		Run:        pick(ov.Run, def.Run),
		Labor:      pick(ov.Labor, def.Labor),
		RunBaseQty: baseQty,
		Queue:      pick(ov.Queue, def.Queue),
		Wait:       pick(ov.Wait, def.Wait),
		Move:       pick(ov.Move, def.Move),
		CrewSize:   crew,
	}
}

// Batches returns the number of run cycles for qty pieces. A partial last cycle
// still occupies a full cycle (ceil), mirroring the machine production-time logic.
func (t OperationTime) Batches(qty float64) float64 {
	base := t.RunBaseQty
	if base <= 0 {
		base = 1
	}
	if qty <= 0 {
		return 0
	}
	// Precisão de 1e-9 antes do teto: 0,36 m² × 100 peças dá 36,000000000000004
	// em float e viraria 37 ciclos.
	return math.Ceil(math.Round(qty*t.unidadesPorPeca()/base*1e9) / 1e9)
}

func (t OperationTime) unidadesPorPeca() float64 {
	if t.SemMedida {
		return 0
	}
	if t.UnitsPerPiece <= 0 {
		return 1
	}
	return t.UnitsPerPiece
}

// SetupMachineHours é a parcela de PREPARAÇÃO dentro da ocupação de máquina.
//
// Existe para o custo separar preparação de produção: as duas se atacam de formas
// opostas (setup alto pede lote maior; hora-máquina alta pede outro recurso) e no
// mesmo número nenhuma das duas aparece. Setup não escala com a quantidade — é uma
// vez por ordem —, e é justamente por isso que diluí-lo pelo lote muda o custo
// unitário.
func (t OperationTime) SetupMachineHours() float64 { return t.Setup }

// SetupLaborHours é a preparação em horas-homem: a equipe inteira participa do
// setup, então a parcela escala com o tamanho da equipe — igual a LaborHours.
func (t OperationTime) SetupLaborHours() float64 {
	crew := t.CrewSize
	if crew <= 0 {
		crew = 1
	}
	return t.Setup * crew
}

// MachineHours is the work-center occupancy (setup once + run per batch) in hours.
func (t OperationTime) MachineHours(qty float64) float64 {
	return t.Setup + t.Run*t.Batches(qty)
}

// LaborHours is the direct-labor effort in hours, scaled by crew size.
// When Labor is 0 the operator attends the machine for the full run time.
func (t OperationTime) LaborHours(qty float64) float64 {
	run := t.Labor
	if run <= 0 {
		run = t.Run
	}
	crew := t.CrewSize
	if crew <= 0 {
		crew = 1
	}
	return (t.Setup + run*t.Batches(qty)) * crew
}

// LeadTimeHours is the wall-clock duration of the operation for CPM scheduling:
// setup + machining + queue + wait + move, all in hours.
func (t OperationTime) LeadTimeHours(qty float64) float64 {
	return t.Setup + t.Queue + t.Wait + t.Move + t.Run*t.Batches(qty)
}
