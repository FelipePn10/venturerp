package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// CreateStructureComponentUseCase adiciona um componente (filho) à estrutura
type CreateStructureComponentUseCase struct {
	Repo repository.ItemStructureRepository
	Auth ports.AuthService
}

func NewCreateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *CreateStructureComponentUseCase {
	return &CreateStructureComponentUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateStructureComponentUseCase) Execute(
	ctx context.Context,
	dto request.CreateStructureComponentDTO,
) (*entity.ItemStructure, error) {
	if !uc.Auth.CanCreateStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	parentExists, err := uc.Repo.ItemExists(ctx, dto.ParentCode)
	if err != nil {
		return nil, fmt.Errorf("checking parent item: %w", err)
	}
	if !parentExists {
		return nil, fmt.Errorf("parent item %d not found", dto.ParentCode)
	}

	childExists, err := uc.Repo.ItemExists(ctx, dto.ChildCode)
	if err != nil {
		return nil, fmt.Errorf("checking child item: %w", err)
	}
	if !childExists {
		return nil, fmt.Errorf("child item %d not found", dto.ChildCode)
	}

	// só bloqueia se o filho já é ancestral do pai (A→B→C→A)
	hasCycle, err := uc.Repo.HasCyclicReference(ctx, dto.ParentCode, dto.ChildCode)
	if err != nil {
		return nil, err
	}
	if hasCycle {
		return nil, fmt.Errorf("adding item %d as child of %d would create a cycle", dto.ChildCode, dto.ParentCode)
	}

	exists, err := uc.Repo.SequenceExists(ctx, dto.ParentCode, dto.Sequence)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("sequence %d already used in structure of item %d", dto.Sequence, dto.ParentCode)
	}

	structure, err := entity.NewItemStructure(
		dto.ParentCode,
		dto.ChildCode,
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
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	structure.IsCoproduct = dto.IsCoproduct
	structure.IsFixedQty = dto.IsFixedQty
	structure.SetSubstitute(dto.SubstituteGroup, dto.SubstitutePriority)

	return uc.Repo.Create(ctx, structure)
}
