package stock_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type CreateInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CreateInventoryUseCase) Execute(ctx context.Context, dto request.CreateInventoryDTO) (*response.PhysicalInventoryResponse, error) {
	if !uc.Auth.CanCreateInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if dto.Code <= 0 || dto.WarehouseID <= 0 || dto.Description == "" {
		return nil, errorsuc.NewValidationError("código, descrição e almoxarifado são obrigatórios")
	}
	startDate, parseErr := time.Parse("2006-01-02", dto.StartDate)
	if parseErr != nil {
		return nil, errorsuc.NewValidationError("data inicial inválida; use AAAA-MM-DD")
	}

	inv := &entity.PhysicalInventory{
		Code:        dto.Code,
		Description: dto.Description,
		WarehouseID: dto.WarehouseID,
		StartDate:   startDate,
		Status:      "OPEN",
		Notes:       dto.Notes,
		CreatedBy:   userID,
	}

	if dto.EndDate != nil {
		t, parseErr := time.Parse("2006-01-02", *dto.EndDate)
		if parseErr != nil || t.Before(startDate) {
			return nil, errorsuc.NewValidationError("data final inválida ou anterior à data inicial")
		}
		inv.EndDate = &t
	}

	created, err := uc.Repo.CreateInventory(ctx, inv)
	if err != nil {
		return nil, err
	}
	return toPhysicalInventoryResponse(created), nil
}
