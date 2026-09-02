package mrp_calculation_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type configuredRuleAuth struct{ ports.AuthService }

func (configuredRuleAuth) CanConfiguredRulesMRP(context.Context) bool { return true }

type missingConfiguredRuleItem struct{ itemrepo.ItemRepository }

func (missingConfiguredRuleItem) FindItemByCode(context.Context, valueobject.ItemCode) (*itementity.Item, error) {
	return nil, itemrepo.ErrNotFound
}

type configuredRuleRepo struct {
	repository.MRPCalculationRepository
	called bool
}

func TestConfiguredRuleValidatesItemBeforePersistence(t *testing.T) {
	repo := &configuredRuleRepo{}
	uc := &ManageConfiguredItemRulesUseCase{Repo: repo, Auth: configuredRuleAuth{}, Items: missingConfiguredRuleItem{}}
	_, err := uc.Create(context.Background(), request.CreateConfiguredItemRuleDTO{ItemCode: 999, TableType: "ITEM", FieldName: "lot", RuleType: "EQUALS", RuleValue: "A", Sequence: 1})
	if err == nil || !strings.Contains(err.Error(), "item de referência não encontrado") || repo.called {
		t.Fatalf("validação de referência incorreta: err=%v called=%v", err, repo.called)
	}
}
