package structure_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// UpdateStructureComponentUseCase atualiza quantidade, unidade de medida,
// percentual de perda, posição e notas de um componente BOM existente.
// Nota: a máscara (parent_mask) e os IDs pai/filho NÃO são editáveis.
// Para mudar esses campos, remova e recrie o componente.
type UpdateStructureComponentUseCase struct {
	Repo  repository.ItemStructureRepository
	Auth  ports.AuthService
	Items any
}

func NewUpdateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
	items ...any,
) *UpdateStructureComponentUseCase {
	uc := &UpdateStructureComponentUseCase{
		Repo: repo,
		Auth: auth,
	}
	if len(items) > 0 {
		uc.Items = items[0]
	}
	return uc
}

func (uc *UpdateStructureComponentUseCase) Execute(
	ctx context.Context,
	dto request.UpdateStructureComponentDTO,
) (*entity.ItemStructure, error) {

	if !uc.Auth.UpdateStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Position < 1 {
		return nil, errorsuc.NewValidationError("a posição do componente é obrigatória e deve ser positiva")
	}

	parentCode, err := resolveItemCode(ctx, uc.Items, dto.ParentCode)
	if err != nil {
		return nil, err
	}
	childCode, err := resolveItemCode(ctx, uc.Items, dto.ChildCode)
	if err != nil {
		return nil, err
	}
	structure := &entity.ItemStructure{
		ParentCode: parentCode,
		ChildCode:  childCode,
		ParentMask: dto.ParentMask,
	}

	if err := structure.Update(
		dto.Quantity,
		dto.UnitOfMeasurement,
		dto.Health,
		dto.LossPercentage,
		dto.Position,
		dto.Notes,
		dto.StartDate,
		dto.EndDate,
		dto.LossFormula,
		dto.QuantityFormula,
		dto.QuantityRounding,
		dto.QuantityScale,
	); err != nil {
		return nil, err
	}
	structure.IsCoproduct = dto.IsCoproduct
	structure.IsFixedQty = dto.IsFixedQty
	structure.SetSubstitute(dto.SubstituteGroup, dto.SubstitutePriority)
	structure.WarehouseCode = dto.WarehouseCode
	structure.LineWarehouseCode = dto.LineWarehouseCode
	structure.SetupLoss = dto.SetupLoss
	structure.CostLossType = entity.NormalizeCostLossType(dto.CostLossType)
	structure.CostLoss = dto.CostLoss
	structure.CostCenterCode = dto.CostCenterCode
	structure.IsCriticalMPS = dto.IsCriticalMPS
	structure.GeneratesInspection = dto.GeneratesInspection

	// Executa update direto via business key
	updated, err := uc.Repo.Update(ctx, structure)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
