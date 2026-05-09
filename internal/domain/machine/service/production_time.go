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

	// How many full (or partial) production cycles are needed?
	// ceil ensures a partial last batch still reserves a full machine cycle.
	batchCount := math.Ceil(demandQty / float64(imt.ProductionBaseQty))

	// Setup time comes exclusively from ItemMachineTime — it already reflects the
	// specific mask variant (different fixtures, jigs, or program loads per size).
	setupMinutes := imt.SetupTime

	machiningMinutes := batchCount * productionTimeMinutes
	totalMinutes := machiningMinutes + setupMinutes

	// Machine effective capacity in machine-units per minute.
	machinePeriodMinutes := periodToMinutes(machine.CapacityPeriod, workingMinsPerDay)
	machineCapacityPerMinute := (machine.Capacity * machine.EfficiencyRate) / machinePeriodMinutes

	// Bottleneck check: compare required throughput vs machine capacity.
	// Required throughput = demand (in machine units) / total available minutes.
	demandInMachineUnits := demandQty * conversionFactor
	var isBottleneck bool
	if totalMinutes > 0 && machineCapacityPerMinute > 0 {
		requiredRate := demandInMachineUnits / totalMinutes
		isBottleneck = requiredRate > machineCapacityPerMinute
	}

	return ProductionTimeResult{
		TotalMinutes:             totalMinutes,
		TotalHours:               totalMinutes / 60.0,
		TotalDays:                totalMinutes / workingMinsPerDay,
		BatchCount:               batchCount,
		SetupMinutes:             setupMinutes,
		MachiningMinutes:         machiningMinutes,
		ConversionFactor:         conversionFactor,
		MachineIsBottleneck:      isBottleneck,
		MachineCapacityPerMinute: machineCapacityPerMinute,
	}
}

// periodToMinutes converts a CapacityPeriod enum value to minutes.
func periodToMinutes(period types.CapacityPeriod, workingMinsPerDay float64) float64 {
	switch period {
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
