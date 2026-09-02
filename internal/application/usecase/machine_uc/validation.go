package machine_uc

import (
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// As unidades e os períodos de capacidade são enums no banco: validamos aqui
// para devolver 422 em PT-BR em vez de deixar o Postgres recusar o valor.
var capacityUnits = []types.MachineCapacityUnit{
	types.Pieces, types.Units, types.Sheets, types.Kilogram, types.Ton,
	types.Meters, types.SquareMeters, types.CubicMeters, types.Liters,
}

var capacityPeriods = []types.CapacityPeriod{types.Minute, types.Hour, types.Day}

func joinUnits() string {
	parts := make([]string, 0, len(capacityUnits))
	for _, u := range capacityUnits {
		parts = append(parts, string(u))
	}
	return strings.Join(parts, ", ")
}

func joinPeriods() string {
	parts := make([]string, 0, len(capacityPeriods))
	for _, p := range capacityPeriods {
		parts = append(parts, string(p))
	}
	return strings.Join(parts, ", ")
}

// normalizeCapacityUnit aceita a unidade em qualquer caixa e recusa o que o
// enum do banco não conhece.
func normalizeCapacityUnit(raw types.MachineCapacityUnit) (types.MachineCapacityUnit, error) {
	value := types.MachineCapacityUnit(strings.ToUpper(strings.TrimSpace(string(raw))))
	if value == "" {
		return "", errorsuc.NewValidationError("informe a unidade de capacidade da máquina (" + joinUnits() + ")")
	}
	for _, u := range capacityUnits {
		if value == u {
			return u, nil
		}
	}
	return "", errorsuc.NewValidationError(
		fmt.Sprintf("unidade de capacidade %q inválida: use uma destas — %s", string(raw), joinUnits()))
}

// normalizeCapacityPeriod aceita o período em qualquer caixa.
func normalizeCapacityPeriod(raw types.CapacityPeriod) (types.CapacityPeriod, error) {
	value := types.CapacityPeriod(strings.ToUpper(strings.TrimSpace(string(raw))))
	if value == "" {
		return "", errorsuc.NewValidationError("informe o período de capacidade da máquina (" + joinPeriods() + ")")
	}
	for _, p := range capacityPeriods {
		if value == p {
			return p, nil
		}
	}
	return "", errorsuc.NewValidationError(
		fmt.Sprintf("período de capacidade %q inválido: use um destes — %s", string(raw), joinPeriods()))
}

// normalizeEfficiency aceita a eficiência como fração (0,9) ou como percentual
// (90) — a tela usa fração, mas o usuário costuma digitar o percentual.
func normalizeEfficiency(rate float64) (float64, error) {
	if rate <= 0 {
		return 0, errorsuc.NewValidationError("informe a eficiência da máquina (por exemplo 0,9 para 90%)")
	}
	if rate > 1 && rate <= 100 {
		return rate / 100, nil
	}
	if rate > 100 {
		return 0, errorsuc.NewValidationError("a eficiência da máquina não pode passar de 100%")
	}
	return rate, nil
}

// validateMachineFields aplica as validações comuns a criar e alterar máquina.
func validateMachineFields(code int64, name string, machineTypeCode int64, capacity float64) error {
	if code <= 0 {
		return errorsuc.NewValidationError("informe o código da máquina")
	}
	if strings.TrimSpace(name) == "" {
		return errorsuc.NewValidationError("informe o nome da máquina")
	}
	if machineTypeCode <= 0 {
		return errorsuc.NewValidationError("informe o tipo da máquina")
	}
	if capacity <= 0 {
		return errorsuc.NewValidationError("a capacidade da máquina deve ser maior que zero")
	}
	return nil
}
