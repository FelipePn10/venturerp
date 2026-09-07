package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// CreateStructureComponentUseCase adiciona um componente (filho) à estrutura
type CreateStructureComponentUseCase struct {
	Repo  repository.ItemStructureRepository
	Auth  ports.AuthService
	Items any
}

func NewCreateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
	items ...any,
) *CreateStructureComponentUseCase {
	uc := &CreateStructureComponentUseCase{
		Repo: repo,
		Auth: auth,
	}
	if len(items) > 0 {
		uc.Items = items[0]
	}
	return uc
}

func (uc *CreateStructureComponentUseCase) Execute(
	ctx context.Context,
	dto request.CreateStructureComponentDTO,
) (*entity.ItemStructure, error) {
	if !uc.Auth.CanCreateStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Sequence < 1 {
		return nil, errorsuc.NewValidationError("a posição do componente é obrigatória e deve ser positiva")
	}
	parentCode, err := resolveItemCode(ctx, uc.Items, dto.ParentCode)
	if err != nil {
		return nil, err
	}
	childItem, err := itemresolution.Resolve(ctx, uc.Items, dto.ChildCode)
	if err != nil {
		return nil, err
	}
	childCode := int64(childItem.Code)
	if dto.UnitOfMeasurement == "" {
		dto.UnitOfMeasurement = childItem.Warehouse.UnitOfMeasurement
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	parentExists, err := uc.Repo.ItemExists(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("checking parent item: %w", err)
	}
	if !parentExists {
		return nil, errorsuc.NewNotFoundError("item pai não encontrado")
	}

	childExists, err := uc.Repo.ItemExists(ctx, childCode)
	if err != nil {
		return nil, fmt.Errorf("checking child item: %w", err)
	}
	if !childExists {
		return nil, errorsuc.NewNotFoundError("item filho não encontrado")
	}

	// só bloqueia se o filho já é ancestral do pai (A→B→C→A)
	hasCycle, err := uc.Repo.HasCyclicReference(ctx, parentCode, childCode)
	if err != nil {
		return nil, err
	}
	if hasCycle {
		return nil, fmt.Errorf("o componente criaria um ciclo na estrutura")
	}

	exists, err := uc.Repo.SequenceExists(ctx, parentCode, dto.Sequence)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("a posição %d já está em uso na estrutura", dto.Sequence)
	}

	structure, err := entity.NewItemStructure(
		parentCode,
		childCode,
		dto.ParentMask,
		dto.Quantity,
		dto.UnitOfMeasurement,
		dto.Health,
		dto.LossPercentage,
		dto.Sequence,
		dto.Notes,
		dto.IsActive,
		dto.Inherit,
		dto.StartDate,
		dto.EndDate,
		dto.LossFormula,
		actor,
		dto.QuantityFormula,
		dto.QuantityRounding,
		dto.QuantityScale,
	)
	if err != nil {
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

	return uc.Repo.Create(ctx, structure)
}
