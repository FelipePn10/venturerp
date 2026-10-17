package warehouse_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository"
)

// UpdateWarehouseUseCase altera descrição, localização, tipo, disposição,
// reservas e a permissão de saldo negativo. Antes o almoxarifado só podia ser
// criado: um erro de digitação ficava para sempre.
type UpdateWarehouseUseCase struct {
	Repo repository.WarehouseRepository
	Auth ports.AuthService
}

func (uc *UpdateWarehouseUseCase) Execute(ctx context.Context, code string, dto request.UpdateWarehouseRequestDTO) (*response.WarehouseResponse, error) {
	if !uc.Auth.CanCreateWarehouse(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	atual, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("descrição do almoxarifado é obrigatória")
	}
	if !dto.Location.IsValid() {
		return nil, errorsuc.NewValidationError("localização deve ser INTERNO, EXTERNO, INSPECAO, REJEICAO, RESERVA, TRANSITO, ESPECIAL, EXPEDICAO ou ASSISTENCIA_TECNICA")
	}
	if !dto.Type.IsValid() {
		return nil, errorsuc.NewValidationError("tipo deve ser NORMAL ou LINHA DE PRODUÇÃO")
	}
	atual.Description = strings.TrimSpace(dto.Description)
	atual.Location, atual.Type = dto.Location, dto.Type
	atual.Disposition, atual.ReservationsAllowed = dto.Disposition, dto.ReservationsAllowed
	atual.AllowNegativeStock = dto.AllowNegativeStock
	salvo, err := uc.Repo.Update(ctx, atual)
	if err != nil {
		return nil, err
	}
	return toWarehouseResponse(salvo), nil
}
