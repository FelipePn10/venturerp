package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
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
) (*entity.ConfiguredItemRule, error) {
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
	return uc.Repo.CreateConfiguredItemRule(ctx, rule)
}

func (uc *ManageConfiguredItemRulesUseCase) ListByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.ConfiguredItemRule, error) {
	return uc.Repo.GetConfiguredItemRules(ctx, itemCode)
}
