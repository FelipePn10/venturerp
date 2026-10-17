package stock_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"strings"

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
	// Tipos lê o cadastro de tipos de movimento (opcional na montagem).
	Tipos TiposDeMovimento
}

// TiposDeMovimento lê o tipo de movimento da empresa pela sigla.
type TiposDeMovimento interface {
	TipoDeMovimento(ctx context.Context, sigla string) (*entity.TipoMovimentoConfig, error)
}

func (uc *CreateStockMovementUseCase) Execute(ctx context.Context, dto request.CreateStockMovementDTO) (*response.StockMovementResponse, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	valorizarPelaMedia := false
	var siglaTipo *string
	if dto.MovementTypeCode != nil && strings.TrimSpace(*dto.MovementTypeCode) != "" {
		if uc.Tipos == nil {
			return nil, errorsuc.NewValidationError("tipo de movimento configurado indisponível")
		}
		tipo, err := uc.Tipos.TipoDeMovimento(ctx, *dto.MovementTypeCode)
		if err != nil {
			return nil, err
		}
		if err := aplicarTipoDeMovimento(tipo, &dto); err != nil {
			return nil, err
		}
		valorizarPelaMedia = !tipo.AtualizaCusto
		sigla := tipo.Sigla
		siglaTipo = &sigla
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

		MovementTypeCode:   siglaTipo,
		CostCenterCode:     dto.CostCenterCode,
		ValorizarPelaMedia: valorizarPelaMedia,
	}

	if t, msg := datetime.Opcional("data de validade", dto.ExpirationDate); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	} else {
		m.ExpirationDate = t
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

// aplicarTipoDeMovimento faz o cadastro de tipos de movimento valer no
// lançamento manual. Antes o cadastro existia (VEST0200) mas o lançamento usava
// uma lista fixa: "consumo interno" e "devolução de produção" eram só IN/OUT,
// sem centro de custo e sem regra de custo.
func aplicarTipoDeMovimento(t *entity.TipoMovimentoConfig, dto *request.CreateStockMovementDTO) error {
	if !t.Ativo {
		return errorsuc.NewValidationError(fmt.Sprintf("o tipo de movimento %s está inativo", t.Sigla))
	}
	if !t.PermiteManual {
		return errorsuc.NewValidationError(fmt.Sprintf("o tipo de movimento %s (%s) não permite lançamento manual: ele é gerado pelo próprio processo", t.Sigla, t.Descricao))
	}
	switch t.Direcao {
	case "ENTRADA":
		dto.MovementType = entity.MovementTypeIn
	case "SAIDA":
		dto.MovementType = entity.MovementTypeOut
	case "TRANSFERENCIA":
		return errorsuc.NewValidationError(fmt.Sprintf("o tipo %s é de transferência: use a transferência entre almoxarifados", t.Sigla))
	default:
		if dto.MovementType != entity.MovementTypeIn && dto.MovementType != entity.MovementTypeOut && dto.MovementType != entity.MovementTypeAdjustment {
			return errorsuc.NewValidationError(fmt.Sprintf("o tipo %s serve para entrada e saída: diga qual (IN ou OUT)", t.Sigla))
		}
	}
	if t.ExigeCentroCusto && (dto.CostCenterCode == nil || *dto.CostCenterCode <= 0) {
		return errorsuc.NewValidationError(fmt.Sprintf("o tipo de movimento %s exige o centro de custo", t.Sigla))
	}
	return nil
}
