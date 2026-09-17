package service

import (
	"math"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

const DefaultWorkingMinutesPerDay = 480.0 // 8h × 60min

// ProductionTimeResult is the output of a production time calculation.
type ProductionTimeResult struct {
	// Total wall-clock production time (machining + setup).
	TotalMinutes float64 `json:"total_minutes"`
	TotalHours   float64 `json:"total_hours"`
	TotalDays    float64 `json:"total_days"`

	// BatchCount = ceil(demandQty / productionBaseQty).
	//
	// productionBaseQty represents how many items a single production cycle covers.
	// Example: a hydraulic press that stamps 10 sheets per stroke → productionBaseQty = 10.
	//   demandQty = 73 sheets → BatchCount = ceil(73/10) = 8 cycles.
	// Even the last (partial) cycle occupies the machine for the full cycle time.
	BatchCount float64 `json:"batch_count"`

	// SetupMinutes is the one-time setup cost taken directly from ItemMachineTime.SetupTime.
	// It already accounts for the mask variant (e.g. changing jigs/fixtures for a specific size).
	SetupMinutes float64 `json:"setup_minutes"`

	// MachiningMinutes is the pure cycle time (BatchCount × normalised production_time), without setup.
	MachiningMinutes float64 `json:"machining_minutes"`

	// ConversionFactor is the item→machine unit multiplier used (e.g. KG→T = 0.001).
	ConversionFactor float64 `json:"conversion_factor"`

	// MachineIsBottleneck is true when the machine's effective capacity per minute
	// is lower than the throughput required to serve this demand in the calculated time.
	MachineIsBottleneck bool `json:"machine_is_bottleneck"`

	// MachineCapacityPerMinute is the machine's effective output per minute
	// after applying efficiency_rate: capacity * efficiency_rate / periodInMinutes.
	MachineCapacityPerMinute float64 `json:"machine_capacity_per_minute"`

	// Consumível: quanto a ordem gasta e quantas trocas ela obriga. A troca para
	// a máquina, então entra no tempo total — é isso que impede a ordem de "caber"
	// no turno no papel e estourar no chão.
	ConsumableDescription string  `json:"consumable_description,omitempty"`
	ConsumableUnit        string  `json:"consumable_unit,omitempty"`
	ConsumableUsed        float64 `json:"consumable_used"`
	ConsumableRefills     float64 `json:"consumable_refills"`
	ConsumableMinutes     float64 `json:"consumable_minutes"`

	StandardCycleMinutes  float64  `json:"standard_cycle_minutes"`
	EffectiveCycleMinutes float64  `json:"effective_cycle_minutes"`
	ProductionBaseQty     int      `json:"production_base_qty"`
	MachineEfficiencyRate float64  `json:"machine_efficiency_rate"`
	TimeBasis             string   `json:"time_basis"`
	EfficiencySource      string   `json:"efficiency_source"`
	ResourceTimeFactor    float64  `json:"resource_time_factor"`
	RequiredCapacityRate  float64  `json:"required_capacity_per_minute"`
	WorkingMinutesPerDay  float64  `json:"working_minutes_per_day"`
	CalculationFactors    []string `json:"calculation_factors"`
}

// CalculateProductionTime computes how long it takes to produce demandQty items
// on the given machine using the item+mask-specific production time configuration.
//
// Parameters:
//   - imt              — ItemMachineTime row for this item+mask+machine combination.
//   - machine          — Machine entity (capacity, efficiency, period).
//   - demandQty        — Quantity to produce in item units.
//   - conversionFactor — Converts item units → machine capacity units.
//   - workingMinsPerDay — Productive minutes per working day (default: 480 = 8 h).
func CalculateProductionTime(
	imt *entity.ItemMachineTime,
	machine *entity.Machine,
	demandQty float64,
	conversionFactor float64,
	workingMinsPerDay float64,
) ProductionTimeResult {
	if workingMinsPerDay <= 0 {
		workingMinsPerDay = DefaultWorkingMinutesPerDay
	}

	// Normalise the item-specific production time to minutes.
	productionTimeMinutes := imt.ProductionTime * periodToMinutes(imt.ProductionTimeUnit, workingMinsPerDay)
	efficiency := machine.EfficiencyRate
	if imt.EfficiencyRate != nil {
		efficiency = *imt.EfficiencyRate
	}
	if efficiency <= 0 || efficiency > 1 {
		efficiency = 1
	}
	resourceTimeFactor := 1.0
	effectiveCycleMinutes := productionTimeMinutes * resourceTimeFactor / efficiency

	// How many full (or partial) production cycles are needed?
	// ceil ensures a partial last batch still reserves a full machine cycle.
	batchCount := demandQty / float64(imt.ProductionBaseQty)
	if imt.TimeBasis != "PROPORTIONAL" {
		batchCount = math.Ceil(batchCount)
	}

	// Setup time comes exclusively from ItemMachineTime — it already reflects the
	// specific mask variant (different fixtures, jigs, or program loads per size).
	setupMinutes := imt.SetupTime

	machiningMinutes := batchCount * effectiveCycleMinutes

	// Trocas de consumível. A primeira carga já está montada, então o número de
	// PARADAS é o de cargas menos uma: gastar 250 de um cilindro de 100 exige
	// três cargas e duas trocas no meio da ordem. Gasto exatamente igual à
	// capacidade não obriga troca nenhuma — a ordem termina com o cilindro
	// zerado, e quem troca é a ordem seguinte.
	var consumableUsed, refills, consumableMinutes float64
	var consumableUnit, consumableDesc string
	if c := imt.Consumable; c != nil && c.PerHour > 0 && c.CapacityPerRefill > 0 {
		consumableUnit, consumableDesc = c.Unit, c.Description
		consumableUsed = machiningMinutes / 60 * c.PerHour
		if cargas := math.Ceil(consumableUsed / c.CapacityPerRefill); cargas > 1 {
			refills = cargas - 1
			consumableMinutes = refills * c.ReplacementMinutes
		}
	}

	totalMinutes := machiningMinutes + setupMinutes + consumableMinutes

	// Machine effective capacity in machine-units per minute.
	//
	// A eficiência usada aqui é a MESMA que dimensionou o ciclo. Antes esta linha
	// usava machine.EfficiencyRate enquanto o ciclo já usava a do item: o
	// resultado reportava "eficiência 50% (item)" e uma capacidade calculada a
	// 100%, o dobro do real. O efeito visível era o indicador de gargalo ficar
	// otimista justamente nos itens que rendem menos naquela máquina.
	machinePeriodMinutes := periodToMinutes(machine.CapacityPeriod, workingMinsPerDay)
	machineCapacityPerMinute := (machine.Capacity * efficiency) / machinePeriodMinutes

	// Bottleneck check: compare required throughput vs machine capacity.
	// Required throughput = demand (in machine units) / total available minutes.
	demandInMachineUnits := demandQty * conversionFactor
	var isBottleneck bool
	var requiredRate float64
	if totalMinutes > 0 && machineCapacityPerMinute > 0 {
		requiredRate = demandInMachineUnits / totalMinutes
		isBottleneck = requiredRate > machineCapacityPerMinute
	}

	basis := imt.TimeBasis
	if basis == "" {
		basis = "CYCLE"
	}
	source := "MACHINE"
	if imt.EfficiencyRate != nil {
		source = "ITEM"
	}
	return ProductionTimeResult{
		TimeBasis: basis, EfficiencySource: source,
		TotalMinutes:             totalMinutes,
		TotalHours:               totalMinutes / 60.0,
		TotalDays:                totalMinutes / workingMinsPerDay,
		BatchCount:               batchCount,
		SetupMinutes:             setupMinutes,
		MachiningMinutes:         machiningMinutes,
		ConversionFactor:         conversionFactor,
		MachineIsBottleneck:      isBottleneck,
		MachineCapacityPerMinute: machineCapacityPerMinute,
		ConsumableDescription:    consumableDesc,
		ConsumableUnit:           consumableUnit,
		ConsumableUsed:           consumableUsed,
		ConsumableRefills:        refills,
		ConsumableMinutes:        consumableMinutes,
		StandardCycleMinutes:     productionTimeMinutes,
		EffectiveCycleMinutes:    effectiveCycleMinutes,
		ProductionBaseQty:        imt.ProductionBaseQty,
		MachineEfficiencyRate:    efficiency,
		ResourceTimeFactor:       resourceTimeFactor,
		RequiredCapacityRate:     requiredRate,
		WorkingMinutesPerDay:     workingMinsPerDay,
		CalculationFactors: []string{
			"produção proporcional usa quantidade / base; ciclos fechados arredondam para cima",
			"tempo efetivo por ciclo = tempo padrão × fator do recurso / eficiência",
			"tempo total = setup + ciclos × tempo efetivo por ciclo + trocas de consumível",
			"trocas = teto(consumo / capacidade da carga) − 1; a primeira carga já está na máquina",
			"gargalo = capacidade requerida por minuto maior que a capacidade efetiva da máquina",
		},
	}
}

// periodToMinutes converts a CapacityPeriod enum value to minutes.
func periodToMinutes(period types.CapacityPeriod, workingMinsPerDay float64) float64 {
	switch period {
	case types.Second:
		return 1.0 / 60.0
	case types.Minute:
		return 1.0
	case types.Hour:
		return 60.0
	case types.Day:
		return workingMinsPerDay
	default:
		return workingMinsPerDay
	}
}
