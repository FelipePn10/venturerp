package service

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// StockQtyForLength converts a cut/consumed piece of length `lengthMM` into the
// quantity to post against the material's stock balance, in its stock unit of
// measure. A bar is a 1-D object, so every stock UoM is reachable from its length:
//
//   - Piece/unit (UN, empty): one stock unit per physical piece (length-agnostic).
//   - Length (M, CM, MM, IN, MICROMETRO): derived geometrically; no factor needed.
//   - Mass / area / volume (KG, TONELADA, M2, M3): qty = metres × factorPerMeter,
//     where factorPerMeter is the stock-UoM quantity contained in ONE linear metre
//     of the bar — linear density (kg/m), strip width (m²/m) or cross-section
//     (m³/m). These require a positive factor because length alone can't determine
//     mass/area/volume without the bar's section.
//
// Keeping this pure makes the UoM math unit-testable and the single source of
// truth for how a cut translates into a stock baixa.
func StockQtyForLength(uom types.TypeUnitOfMeasurementItem, lengthMM, factorPerMeter float64) (float64, error) {
	if lengthMM < 0 {
		return 0, fmt.Errorf("length cannot be negative")
	}
	meters := lengthMM / 1000.0

	switch uom {
	case "", types.UN:
		return 1, nil
	case types.M:
		return meters, nil
	case types.CM:
		return lengthMM / 10.0, nil
	case types.MM:
		return lengthMM, nil
	case types.IN:
		return lengthMM / 25.4, nil
	case types.MICROMETRO:
		return lengthMM * 1000.0, nil
	case types.KG, types.TONELADA, types.M2, types.M3:
		if factorPerMeter <= 0 {
			return 0, fmt.Errorf("stock UoM %s needs a conversion factor (stock qty per linear metre, e.g. kg/m, m²/m, m³/m)", uom)
		}
		return meters * factorPerMeter, nil
	default:
		// Unknown unit: honour an explicit factor, else treat as one piece.
		if factorPerMeter > 0 {
			return meters * factorPerMeter, nil
		}
		return 1, nil
	}
}

// StockQtyForArea converts a consumed sheet of width×height (mm) into the stock
// quantity, for 2D materials:
//
//   - Piece/unit (UN, empty): one stock unit per sheet.
//   - Area (M2): the sheet area in m².
//   - Volume / mass (M3, KG, TONELADA): area(m²) × factor, where factor is the
//     stock-UoM quantity per square metre (thickness m³/m², weight kg/m²).
//   - Other units fall back to a piece unless an explicit factor is given.
func StockQtyForArea(uom types.TypeUnitOfMeasurementItem, widthMM, heightMM, factorPerM2 float64) (float64, error) {
	if widthMM < 0 || heightMM < 0 {
		return 0, fmt.Errorf("dimensions cannot be negative")
	}
	areaM2 := (widthMM / 1000.0) * (heightMM / 1000.0)

	switch uom {
	case "", types.UN:
		return 1, nil
	case types.M2:
		return areaM2, nil
	case types.M3, types.KG, types.TONELADA:
		if factorPerM2 <= 0 {
			return 0, fmt.Errorf("stock UoM %s needs a conversion factor (stock qty per m², e.g. thickness m³/m², weight kg/m²)", uom)
		}
		return areaM2 * factorPerM2, nil
	default:
		if factorPerM2 > 0 {
			return areaM2 * factorPerM2, nil
		}
		return 1, nil
	}
}
