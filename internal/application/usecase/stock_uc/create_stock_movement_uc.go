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

type CreateStockMovementUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CreateStockMovementUseCase) Execute(ctx context.Context, dto request.CreateStockMovementDTO) (*response.StockMovementResponse, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if dto.MovementType == entity.MovementTypeAddressTransfer {
		return nil, errorsuc.NewValidationError(
			"transferência entre endereços tem rotina própria: use \"Transferir entre endereços\", que baixa a origem e credita o destino na mesma operação")
	}
	if !entity.TipoMovimentoValido(dto.MovementType) {
		return nil, errorsuc.NewValidationError("tipo de movimento inválido: o sistema não reconhece \"" + dto.MovementType + "\" e o saldo não seria atualizado")
	}
	if dto.Quantity <= 0 && dto.MovementType != entity.MovementTypeAdjustment {
		return nil, errorsuc.NewValidationError("a quantidade do movimento precisa ser maior que zero")
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	m := &entity.StockMovement{
		ItemCode:      dto.ItemCode,
		Mask:          dto.Mask,
		WarehouseID:   dto.WarehouseID,
		MovementType:  dto.MovementType,
		Quantity:      dto.Quantity,
		UnitPrice:     dto.UnitPrice,
		TotalPrice:    totalDoMovimento(dto.TotalPrice, dto.Quantity, dto.UnitPrice),
		ReferenceType: dto.ReferenceType,
		ReferenceCode: dto.ReferenceCode,
		Lot:           dto.Lot,
		SerialNumber:  dto.SerialNumber,
		Batch:         dto.Batch,
		Notes:         dto.Notes,
		Address:       dto.Address,
		AddressTo:     dto.AddressTo,
		CreatedBy:     userID,
	}

	if dto.ExpirationDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.ExpirationDate)
		m.ExpirationDate = &t
	}

	created, err := uc.Repo.CreateMovement(ctx, m)
	if err != nil {
		return nil, err
	}
	return toStockMovementResponse(created), nil
}

// totalDoMovimento devolve o valor total do movimento. A tela não envia o campo
// e o caso de uso apenas copiava o que viesse: todo movimento ficava com valor
// total zero, o que some nos relatórios de valorização de estoque.
func totalDoMovimento(total, quantidade, unitario float64) float64 {
	if total > 0 {
		return total
	}
	return quantidade * unitario
}
