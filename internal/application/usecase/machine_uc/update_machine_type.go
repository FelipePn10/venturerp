package machine_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

type UpdateMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *UpdateMachineTypeUseCase) Execute(
	ctx context.Context,
	dto request.UpdateMachineTypeDTO,
) (*response.MachineTypeResponse, error) {
	if !uc.Auth.CanUpdateMachineType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !dto.Type.IsValid() {
		return nil, errorsuc.NewValidationError(
			fmt.Sprintf("classificação %q inválida para o tipo de máquina", string(dto.Type)))
	}

	// Sem o valor atual não há como distinguir "não mandou" de "desligou".
	atual, err := uc.Repo.GetTypeByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}

	mt := &entity.MachineType{
		Code:             dto.Code,
		Name:             dto.Name,
		Description:      dto.Description,
		Type:             dto.Type,
		RequiresOperator: ptrutil.BoolOr(dto.RequiresOperator, atual.RequiresOperator),
		IsActive:         ptrutil.BoolOr(dto.IsActive, atual.IsActive),
	}

	updated, err := uc.Repo.UpdateType(ctx, mt)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(updated), nil
}

// boolOuAtual existe para o teste exercitar a regra sem subir caso de uso
// inteiro; é a mesma de ptrutil.BoolOr.
func boolOuAtual(enviado *bool, atual bool) bool {
	if enviado == nil {
		return atual
	}
	return *enviado
}
