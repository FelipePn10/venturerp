package machine_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	"github.com/google/uuid"
)

type CreateMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *CreateMachineTypeUseCase) Execute(ctx context.Context, dto request.CreateMachineTypeDTO, userID string) (*response.MachineTypeResponse, error) {
	if !uc.Auth.CanCreateType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("informe o nome do tipo de máquina")
	}
	if dto.Code <= 0 {
		return nil, errorsuc.NewValidationError("informe o código do tipo de máquina")
	}
	if !dto.Type.IsValid() {
		return nil, errorsuc.NewValidationError(
			fmt.Sprintf("classificação %q inválida para o tipo de máquina", string(dto.Type)))
	}
	authenticatedUserID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if authenticatedUserID == uuid.Nil {
		return nil, errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}
	if existing, getErr := uc.Repo.GetTypeByCode(ctx, dto.Code); getErr == nil && existing != nil {
		return nil, errorsuc.NewConflictError(
			fmt.Sprintf("já existe um tipo de máquina com o código %d", dto.Code))
	}
	mt := &entity.MachineType{
		Code:             dto.Code,
		Name:             strings.TrimSpace(dto.Name),
		Description:      dto.Description,
		Type:             dto.Type,
		RequiresOperator: dto.RequiresOperator,
		IsActive:         dto.AtivoOuPadrao(),
		CreatedBy:        authenticatedUserID,
	}
	created, err := uc.Repo.CreateType(ctx, mt)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(created), nil
}
