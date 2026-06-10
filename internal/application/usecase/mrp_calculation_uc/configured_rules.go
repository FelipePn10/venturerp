package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type ManageConfiguredItemRulesUseCase struct {
	Repo repository.MRPCalculationRepository
	Auth ports.AuthService
}

func (uc *ManageConfiguredItemRulesUseCase) Create(
	ctx context.Context,
	dto request.CreateConfiguredItemRuleDTO,
) (*response.ConfiguredItemRuleResponse, error) {
	if !uc.Auth.CanConfiguredRulesMRP(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rule := &entity.ConfiguredItemRule{
		ItemCode:  dto.ItemCode,
		TableType: dto.TableType,
		FieldName: dto.FieldName,
		RuleType:  dto.RuleType,
		RuleValue: dto.RuleValue,
		Sequence:  dto.Sequence,
		CreatedBy: dto.CreatedBy,
	}
	created, err := uc.Repo.CreateConfiguredItemRule(ctx, rule)
	if err != nil {
		return nil, err
	}
	return toConfiguredItemRuleResponse(created), nil
}

func (uc *ManageConfiguredItemRulesUseCase) ListByItem(
	ctx context.Context,
	itemCode int64,
) ([]*response.ConfiguredItemRuleResponse, error) {
	list, err := uc.Repo.GetConfiguredItemRules(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toConfiguredItemRuleResponses(list), nil
}
