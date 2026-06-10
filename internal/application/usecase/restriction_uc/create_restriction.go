package restriction_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

type CreateRestrictionUseCase struct {
	Repo repository.RestrictionRepository
	Auth ports.AuthService
}

func (uc *CreateRestrictionUseCase) Execute(
	ctx context.Context,
	dto request.CreateRestrictionDTO,
) (*response.RestrictionResponse, error) {
	if !uc.Auth.CanCreateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	sit := entity.RestrictionSituation(dto.Situation)
	if sit == "" {
		sit = entity.RestrictionActive
	}

	res, err := entity.NewRestriction(
		sit, dto.CustomerCode, dto.ItemCode, dto.ReasonCode,
		dto.ClassificationType, dto.ClassificationOrigin,
		dto.DivisionID, dto.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("building restriction: %w", err)
	}

	created, err := uc.Repo.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	for _, dom := range dto.Dominants {
		dominant := &entity.RestrictionDominant{
			RestrictionID: created.ID,
			QuestionID:    dom.QuestionID,
			Operator:      entity.RestrictionOperator(dom.Operator),
			ConditionType: entity.RestrictionCondition(dom.ConditionType),
			AnswerValue:   dom.AnswerValue,
			Sequence:      dom.Sequence,
		}
		d, err := uc.Repo.AddDominant(ctx, dominant)
		if err != nil {
			return nil, fmt.Errorf("adding dominant: %w", err)
		}
		created.Dominants = append(created.Dominants, d)
	}

	for _, det := range dto.Determinants {
		determinant := &entity.RestrictionDeterminant{
			RestrictionID: created.ID,
			QuestionID:    det.QuestionID,
			Operator:      entity.RestrictionOperator(det.Operator),
			AnswerValue:   det.AnswerValue,
		}
		d, err := uc.Repo.AddDeterminant(ctx, determinant)
		if err != nil {
			return nil, fmt.Errorf("adding determinant: %w", err)
		}
		created.Determinants = append(created.Determinants, d)
	}

	return toRestrictionResponse(created), nil
}
