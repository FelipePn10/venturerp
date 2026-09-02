package mrp_calculation_uc

import (
	"context"
	"errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type ManageConfiguredItemRulesUseCase struct {
	Repo  repository.MRPCalculationRepository
	Auth  ports.AuthService
	Items itemrepo.ItemRepository
}

func (uc *ManageConfiguredItemRulesUseCase) Create(
	ctx context.Context,
	dto request.CreateConfiguredItemRuleDTO,
) (*response.ConfiguredItemRuleResponse, error) {
	if !uc.Auth.CanConfiguredRulesMRP(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode <= 0 || strings.TrimSpace(dto.TableType) == "" || strings.TrimSpace(dto.FieldName) == "" || strings.TrimSpace(dto.RuleType) == "" || strings.TrimSpace(dto.RuleValue) == "" || dto.Sequence <= 0 {
		return nil, errorsuc.NewValidationError("item, tabela, campo, tipo, valor da regra e sequência positiva são obrigatórios")
	}
	if uc.Items == nil {
		return nil, errors.New("repositório de itens não configurado")
	}
	{
		code, err := valueobject.NewItemCode(dto.ItemCode)
		if err != nil {
			return nil, errorsuc.NewValidationError("código do item inválido")
		}
		if _, err = uc.Items.FindItemByCode(ctx, code); err != nil {
			if errors.Is(err, itemrepo.ErrNotFound) {
				return nil, errorsuc.NewValidationError("item de referência não encontrado — selecione um cadastro existente")
			}
			return nil, err
		}
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	rule := &entity.ConfiguredItemRule{
		ItemCode:  dto.ItemCode,
		TableType: dto.TableType,
		FieldName: dto.FieldName,
		RuleType:  dto.RuleType,
		RuleValue: dto.RuleValue,
		Sequence:  dto.Sequence,
		CreatedBy: actor,
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
