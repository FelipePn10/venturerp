package machine_uc

import (
	"context"
	"math"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

// ConsumableUseCase cuida do que a máquina gasta enquanto produz: gás de corte,
// eletrodo, arame. Guarda a autonomia de uma carga e o tempo de troca; a taxa de
// consumo fica na produtividade do item, porque depende do que está sendo feito.
type ConsumableUseCase struct {
	Repo repository.MachineRepository
}

func (uc *ConsumableUseCase) Upsert(ctx context.Context, dto request.MachineConsumableDTO) (*response.MachineConsumableResponse, error) {
	code := strings.TrimSpace(dto.Code)
	descricao := strings.TrimSpace(dto.Description)
	unidade := strings.TrimSpace(dto.Unit)
	if dto.MachineCode <= 0 || code == "" || descricao == "" || unidade == "" {
		return nil, errorsuc.NewValidationError("informe a máquina, o código, a descrição e a unidade do consumível")
	}
	if !finitePositive(dto.CapacityPerRefill) {
		return nil, errorsuc.NewValidationError("informe quanto rende uma carga completa — sem isso não dá para saber quando a troca acontece")
	}
	if math.IsNaN(dto.ReplacementMinutes) || math.IsInf(dto.ReplacementMinutes, 0) || dto.ReplacementMinutes < 0 {
		return nil, errorsuc.NewValidationError("o tempo de troca não pode ser negativo")
	}
	// A máquina tem de existir nesta empresa; sem esta conferência o erro viria
	// da chave estrangeira, em texto de banco.
	if _, err := uc.Repo.GetByCode(ctx, dto.MachineCode); err != nil {
		return nil, err
	}
	saved, err := uc.Repo.UpsertConsumable(ctx, &entity.MachineConsumable{
		MachineCode:        dto.MachineCode,
		Code:               code,
		Description:        descricao,
		Unit:               unidade,
		CapacityPerRefill:  dto.CapacityPerRefill,
		ReplacementMinutes: dto.ReplacementMinutes,
	})
	if err != nil {
		return nil, err
	}
	return toMachineConsumableResponse(saved), nil
}

func (uc *ConsumableUseCase) List(ctx context.Context, machineCode int64) ([]*response.MachineConsumableResponse, error) {
	rows, err := uc.Repo.ListConsumables(ctx, machineCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.MachineConsumableResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, toMachineConsumableResponse(c))
	}
	return out, nil
}

func (uc *ConsumableUseCase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errorsuc.NewValidationError("informe o consumível")
	}
	return uc.Repo.DeleteConsumable(ctx, id)
}
