package machine_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
)

// O caso de uso CalculateProductionTime calcula quanto tempo leva para produzir
//uma determinada quantidade demandada de uma variante de item + máscara em uma máquina
//específica.

// Normaliza todas as diferenças de período de tempo (MINUTO / HORA / DIA),
// aplica o fator de conversão de unidade item→máquina e seleciona a configuração
// correta de ItemMachineTime com base no código do item, máscara e código da máquina.
type CalculateProductionTimeUseCase struct {
	Repo     repository.MachineRepository
	ItemRepo itemrepo.ItemRepository
	Auth     ports.AuthService
}

// ProductionTimeInput is the request payload for the use case.
type ProductionTimeInput struct {
	ItemCode int64 `json:"item_code"`

	// Mask identifies the dimensional variant of the item (e.g. "130#240#234").
	// Pass nil or empty string to use the default (maskless) configuration.
	Mask *string `json:"mask,omitempty"`

	// MachineCode identifies the machine to be used.
	MachineCode int64 `json:"machine_code"`

	// DemandQty is the quantity to produce, expressed in the item's unit of measurement.
	DemandQty float64 `json:"demand_qty"`

	// WorkingMinutesPerDay overrides the default 480 min/day when positive.
	// Future improvement: pull this from the industrial calendar.
	WorkingMinutesPerDay float64 `json:"working_minutes_per_day,omitempty"`
}

// Execute calculates the production time and returns a detailed result.
func (uc *CalculateProductionTimeUseCase) Execute(
	ctx context.Context,
	input ProductionTimeInput,
) (*machinesvc.ProductionTimeResult, error) {
	if !uc.Auth.CanCreateItemTimeMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if input.DemandQty <= 0 {
		return nil, fmt.Errorf("demand_qty must be greater than zero")
	}

	// --- 1. Fetch item to obtain its unit of measurement ---
	itemCodeVO, err := valueobject.NewItemCode(input.ItemCode)
	if err != nil {
		return nil, fmt.Errorf("invalid item code: %w", err)
	}
	item, err := uc.ItemRepo.FindItemByCode(ctx, itemCodeVO)
	if err != nil {
		return nil, fmt.Errorf("item %d not found: %w", input.ItemCode, err)
	}

	// --- 2. Fetch machine ---
	machine, err := uc.Repo.GetByCode(ctx, input.MachineCode)
	if err != nil {
		return nil, fmt.Errorf("machine %d not found: %w", input.MachineCode, err)
	}

	// --- 3. Validate unit compatibility and obtain conversion factor ---
	compat, err := machinesvc.CheckUnitCompatibility(
		item.Warehouse.UnitOfMeasurement,
		machine.CapacityUnit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"item %d (unit: %s) is incompatible with machine %d (unit: %s): %w",
			input.ItemCode, item.Warehouse.UnitOfMeasurement,
			input.MachineCode, machine.CapacityUnit,
			err,
		)
	}

	// --- 4. Resolve the requested mask (nil/empty → default "") ---
	mask := ""
	if input.Mask != nil {
		mask = *input.Mask
	}

	// --- 5. Select the best ItemMachineTime for this item + mask + machine ---
	//
	// Selection rules (priority order):
	//   a) Exact mask match on this machine (lower Priority number wins).
	//   b) Default (empty mask) on this machine — fallback when no specific config exists.
	//
	// The SQL query already filters is_active = TRUE, so every returned record is active.
	imt, err := uc.selectBestIMT(ctx, input.ItemCode, mask, input.MachineCode)
	if err != nil {
		return nil, err
	}

	workingMins := input.WorkingMinutesPerDay
	if workingMins <= 0 {
		workingMins = machinesvc.DefaultWorkingMinutesPerDay
	}

	// --- 6. Calculate ---
	result := machinesvc.CalculateProductionTime(
		imt,
		machine,
		input.DemandQty,
		compat.Factor,
		workingMins,
	)

	return &result, nil
}

// selectBestIMT returns the best-matching ItemMachineTime for the given
// item + mask + machine combination.
//
// It first tries an exact mask match; if none exists it falls back to the
// default (empty-mask) configuration. Within each group the record with the
// lowest Priority value wins (0 = highest priority).
//
// Note: the SQL query behind ListItemMachineTimes already filters is_active = TRUE,
// so we do not need to check the IsActive field on the entity here.
func (uc *CalculateProductionTimeUseCase) selectBestIMT(
	ctx context.Context,
	itemCode int64,
	mask string,
	machineCode int64,
) (*entity.ItemMachineTime, error) {
	imts, err := uc.Repo.ListItemMachineTimes(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("error fetching production time config: %w", err)
	}

	var exactMatch *entity.ItemMachineTime   // exact mask match on this machine
	var defaultMatch *entity.ItemMachineTime // empty-mask fallback on this machine

	for _, imt := range imts {
		if imt.MachineCode != machineCode {
			continue
		}

		imtMask := ""
		if imt.Mask != nil {
			imtMask = *imt.Mask
		}

		switch imtMask {
		case mask:
			if exactMatch == nil || imt.Priority < exactMatch.Priority {
				exactMatch = imt
			}
		case "":
			// Only use as default when the requested mask is not empty
			// (if mask == "", the exact match above already covers the default case).
			if mask != "" {
				if defaultMatch == nil || imt.Priority < defaultMatch.Priority {
					defaultMatch = imt
				}
			}
		}
	}

	if exactMatch != nil {
		return exactMatch, nil
	}
	if defaultMatch != nil {
		return defaultMatch, nil
	}

	if mask != "" {
		return nil, fmt.Errorf(
			"no active production time config found for item %d, mask '%s', machine %d "+
				"(also checked default/empty-mask config)",
			itemCode, mask, machineCode,
		)
	}
	return nil, fmt.Errorf(
		"no active production time config found for item %d on machine %d",
		itemCode, machineCode,
	)
}
