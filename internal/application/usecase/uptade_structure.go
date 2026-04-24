package usecase

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
	repo repository.ItemStructureRepository
	auth ports.AuthService
}

func (uc *UpdateStructureComponentUseCase) Execute(
	ctx context.Context,
	dto request.UpdateStructureComponentDTO,
) (*entity.ItemStructure, error) {

	if !uc.auth.UpdateStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	structure := &entity.ItemStructure{
		ParentCode: dto.ParentCode,
		ChildCode:  dto.ChildCode,
		ParentMask: dto.ParentMask,
	}

	if err := structure.Update(
		dto.Quantity,
		dto.UnitOfMeasurement,
		dto.Health,
		dto.LossPercentage,
		dto.Position,
		dto.Notes,
	); err != nil {
		return nil, err
	}

	// Executa update direto via business key
	updated, err := uc.repo.Update(ctx, structure)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
