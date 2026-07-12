package production_plan_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/google/uuid"
)

type planAuth struct {
	ports.AuthService
	userID uuid.UUID
}

func (a planAuth) CanCreateProductionPlan(context.Context) bool { return true }
func (a planAuth) CanUpdateProductionPlan(context.Context) bool { return true }
func (a planAuth) UserID(context.Context) (uuid.UUID, error)    { return a.userID, nil }

type planRepo struct{ stored *entity.ProductionPlan }

func (r *planRepo) Create(_ context.Context, p *entity.ProductionPlan) (*entity.ProductionPlan, error) {
	r.stored = p
	return p, nil
}
func (r *planRepo) Update(_ context.Context, p *entity.ProductionPlan) (*entity.ProductionPlan, error) {
	r.stored = p
	return p, nil
}
func (r *planRepo) GetByCode(context.Context, int64) (*entity.ProductionPlan, error) {
	return r.stored, nil
}
func (*planRepo) List(context.Context) ([]*entity.ProductionPlan, error) { return nil, nil }
func (*planRepo) Delete(context.Context, int64) error                    { return nil }
func (*planRepo) UpdateLastCalculated(context.Context, int64) error      { return nil }
func (r *planRepo) ReplaceInterFactories(_ context.Context, _ int64, entries []*entity.InterFactoryEnterprise) ([]*entity.InterFactoryEnterprise, error) {
	return entries, nil
}
func (*planRepo) ListInterFactories(context.Context, int64) ([]*entity.InterFactoryEnterprise, error) {
	return nil, nil
}

func TestInterFactoriesRejectDuplicatedEnterprise(t *testing.T) {
	existing, err := entity.NewProductionPlan(1, "Plano", entity.IndependentDemandsAll, false, nil, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	uc := &ManageProductionPlanInterFactoriesUseCase{Repo: &planRepo{stored: existing}, Auth: planAuth{userID: uuid.New()}}
	_, err = uc.Replace(context.Background(), 1, request.ReplaceProductionPlanInterFactoriesDTO{Enterprises: []request.ProductionPlanInterFactoryDTO{{EnterpriseCode: 2}, {EnterpriseCode: 2, AutoRelease: true}}})
	if err == nil {
		t.Fatal("expected duplicated enterprise validation error")
	}
}

func TestCreateProductionPlanUsesAuthenticatedUser(t *testing.T) {
	userID := uuid.New()
	repo := &planRepo{}
	uc := &CreateProductionPlanUseCase{Repo: repo, Auth: planAuth{userID: userID}}
	_, err := uc.Execute(context.Background(), request.CreateProductionPlanDTO{Code: 1, Name: "Plano", IndependentDemands: entity.IndependentDemandsAll})
	if err != nil {
		t.Fatal(err)
	}
	if repo.stored.CreatedBy != userID {
		t.Fatalf("expected authenticated creator %s, got %s", userID, repo.stored.CreatedBy)
	}
}

func TestUpdateProductionPlanPreservesCreator(t *testing.T) {
	creator := uuid.New()
	existing, err := entity.NewProductionPlan(1, "Antigo", entity.IndependentDemandsAll, false, nil, creator)
	if err != nil {
		t.Fatal(err)
	}
	repo := &planRepo{stored: existing}
	uc := &UpdateProductionPlanUseCase{Repo: repo, Auth: planAuth{userID: uuid.New()}}
	_, err = uc.Execute(context.Background(), request.UpdateProductionPlanDTO{Code: 1, Name: "Novo", IndependentDemands: entity.IndependentDemandsNo, PlanningTypes: []string{"MRP"}})
	if err != nil {
		t.Fatal(err)
	}
	if repo.stored.CreatedBy != creator {
		t.Fatalf("creator changed from %s to %s", creator, repo.stored.CreatedBy)
	}
}
