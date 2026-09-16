package machine_uc

import (
	"context"
	"math"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type CreateItemMachineTimeUseCase struct {
	Repo     repository.MachineRepository
	ItemRepo itemrepo.ItemRepository
	Auth     ports.AuthService
}

func (uc *CreateItemMachineTimeUseCase) Execute(
	ctx context.Context, dto request.CreateItemMachineTimeDTO,
) (*response.ItemMachineTimeResponse, error) {
	if !uc.Auth.CanCreateItemTimeMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	unit, err := normalizeCapacityPeriod(dto.ProductionTimeUnit)
	if err != nil {
		return nil, err
	}
	if !finitePositive(dto.ProductionTime) || dto.ProductionBaseQty <= 0 || dto.ProductionBaseQty > math.MaxInt32 || math.IsNaN(dto.SetupTime) || math.IsInf(dto.SetupTime, 0) || dto.SetupTime < 0 || dto.Priority < 0 {
		return nil, errorsuc.NewValidationError("informe tempo e quantidade-base positivos, preparação e prioridade não negativas")
	}
	basis := strings.ToUpper(strings.TrimSpace(dto.TimeBasis))
	if basis == "" {
		basis = "CYCLE"
	}
	if basis != "CYCLE" && basis != "PROPORTIONAL" {
		return nil, errorsuc.NewValidationError("tipo de produção deve ser CYCLE ou PROPORTIONAL")
	}
	if dto.EfficiencyRate != nil {
		if !finitePositive(*dto.EfficiencyRate) {
			return nil, errorsuc.NewValidationError("eficiência do item deve ser maior que zero")
		}
		rate, err := normalizeEfficiency(*dto.EfficiencyRate)
		if err != nil {
			return nil, err
		}
		dto.EfficiencyRate = &rate
	}
	item, err := itemresolution.Resolve(ctx, uc.ItemRepo, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(item.Code)

	if _, err := uc.Repo.GetByCode(ctx, dto.MachineCode); err != nil {
		return nil, err
	}

	// Consumível e taxa andam juntos. O banco também recusa o par incompleto,
	// mas ali a mensagem seria o texto cru do CHECK.
	if (dto.ConsumableID == nil) != (dto.ConsumptionPerHour == nil) {
		return nil, errorsuc.NewValidationError("informe o consumível e o consumo por hora juntos, ou deixe os dois em branco")
	}
	if dto.ConsumptionPerHour != nil && !finitePositive(*dto.ConsumptionPerHour) {
		return nil, errorsuc.NewValidationError("o consumo por hora deve ser maior que zero")
	}

	imt := &entity.ItemMachineTime{
		EfficiencyRate: dto.EfficiencyRate, TimeBasis: basis,
		ConsumableID:       dto.ConsumableID,
		ConsumptionPerHour: dto.ConsumptionPerHour,
		ItemCode:           itemCode,
		Mask:               dto.Mask,
		MachineCode:        dto.MachineCode,
		ProductionTime:     dto.ProductionTime,
		ProductionTimeUnit: unit,
		ProductionBaseQty:  dto.ProductionBaseQty,
		SetupTime:          dto.SetupTime,
		Priority:           dto.Priority,
	}
	created, err := uc.Repo.CreateItemMachineTime(ctx, imt)
	if err != nil {
		return nil, err
	}
	return toItemMachineTimeResponse(created), nil
}

func (uc *CreateItemMachineTimeUseCase) GetByCodeTime(
	ctx context.Context,
	code int64,
) (*response.MachineResponse, error) {
	m, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineResponse(m), nil
}

func finitePositive(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }
