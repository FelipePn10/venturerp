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
	// Items traduz o código de negócio do item para a chave interna. Opcional
	// para os testes que não tocam em itens.
	Items any
}

func (uc *CreateRestrictionUseCase) Execute(
	ctx context.Context,
	dto request.CreateRestrictionDTO,
) (*response.RestrictionResponse, error) {
	if !uc.Auth.CanCreateRestriction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// Conferir antes de gravar: uma cláusula inválida deixava a restrição
	// criada sem as condições, e o usuário recebia 500 em inglês.
	if err := validarClausulas(dto.Dominants, dto.Determinants); err != nil {
		return nil, err
	}

	itemCode, err := resolverCodigoDoItem(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}

	sit := entity.RestrictionSituation(dto.Situation)
	if sit == "" {
		sit = entity.RestrictionActive
	}

	res, errNova := entity.NewRestriction(
		sit, dto.CustomerCode, itemCode, dto.ReasonCode,
		dto.ClassificationType, dto.ClassificationOrigin,
		dto.DivisionID, dto.CreatedBy,
	)
	if errNova != nil {
		return nil, fmt.Errorf("building restriction: %w", errNova)
	}

	created, err := uc.Repo.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	for _, dom := range dto.Dominants {
		dominant := &entity.RestrictionDominant{
			RestrictionID: created.ID,
			QuestionID:    dom.QuestionID,
			Operator:      normalizarOperador(dom.Operator),
			ConditionType: normalizarCondicao(dom.ConditionType),
			AnswerValue:   dom.AnswerValue,
			Sequence:      dom.Sequence,
		}
		d, err := uc.Repo.AddDominant(ctx, dominant)
		if err != nil {
			// Sem as condições a restrição não significa nada e ainda bloquearia
			// combinações por engano: desfaz para não deixar meia regra no banco.
			uc.descartar(ctx, created.Code)
			return nil, fmt.Errorf("gravando a condição (SE) da restrição: %w", err)
		}
		created.Dominants = append(created.Dominants, d)
	}

	for _, det := range dto.Determinants {
		determinant := &entity.RestrictionDeterminant{
			RestrictionID: created.ID,
			QuestionID:    det.QuestionID,
			Operator:      normalizarOperador(det.Operator),
			AnswerValue:   det.AnswerValue,
		}
		d, err := uc.Repo.AddDeterminant(ctx, determinant)
		if err != nil {
			uc.descartar(ctx, created.Code)
			return nil, fmt.Errorf("gravando a consequência (ENTÃO) da restrição: %w", err)
		}
		created.Determinants = append(created.Determinants, d)
	}

	return toRestrictionResponse(created), nil
}

// descartar inativa a restrição recém-criada quando suas cláusulas não puderam
// ser gravadas. Uma restrição sem condições bloquearia combinações sem motivo;
// inativa, fica visível para conserto sem atrapalhar o configurador. Melhor
// esforço: se a inativação também falhar, o erro original é o que interessa.
func (uc *CreateRestrictionUseCase) descartar(ctx context.Context, code int64) {
	_ = uc.Repo.Deactivate(ctx, code)
}
