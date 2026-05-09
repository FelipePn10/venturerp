package service

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

type UnitCompatibility struct {
	Factor float64
}

// CheckUnitCompatibility valida se a unidade de medida de um item é compatível
// com a unidade de capacidade da máquina e retorna o fator de conversão.
// Retorna um erro quando as unidades são fisicamente incompatíveis (por exemplo, massa versus comprimento).
func CheckUnitCompatibility(
	itemUnit types.TypeUnitOfMeasurementItem,
	machineUnit types.MachineCapacityUnit,
) (UnitCompatibility, error) {
	factor, ok := conversionTable[itemUnit][machineUnit]
	if !ok {
		return UnitCompatibility{}, fmt.Errorf(
			"The item unit '%s' is incompatible with the machine unit '%s': "+
				"It is not possible to convert between different physical quantities.",
			itemUnit, machineUnit,
		)
	}
	return UnitCompatibility{Factor: factor}, nil
}

// conversionTable[itemUnit][machineUnit] = factor
// where: machineQty = itemQty * factor
var conversionTable = map[types.TypeUnitOfMeasurementItem]map[types.MachineCapacityUnit]float64{
	// --- Massa ---
	types.KG: {
		types.Kilogram: 1.0,
		types.Ton:      0.001,
	},
	types.TONELADA: {
		types.Ton:      1.0,
		types.Kilogram: 1000.0,
	},

	// --- Comprimento ---
	types.M: {
		types.Meters: 1.0,
	},
	types.MM: {
		types.Meters: 0.001,
	},
	types.CM: {
		types.Meters: 0.01,
	},
	types.IN: {
		types.Meters: 0.0254,
	},
	types.MICROMETRO: {
		types.Meters: 0.000001,
	},

	// --- Área ---
	types.M2: {
		types.SquareMeters: 1.0,
	},

	// --- Volume ---
	types.M3: {
		types.CubicMeters: 1.0,
		types.Liters:      1000.0,
	},

	// --- Unidade / Contagem ---
	types.UN: {
		types.Units:  1.0,
		types.Pieces: 1.0,
		types.Sheets: 1.0,
	},
}
