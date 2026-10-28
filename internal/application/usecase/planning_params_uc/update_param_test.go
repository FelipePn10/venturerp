package planning_params_uc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
)

type fakeAuth struct {
	ports.AuthService
	actor uuid.UUID
}

func (a *fakeAuth) CanManagePlanningParams(context.Context) bool { return true }
func (a *fakeAuth) UserID(context.Context) (uuid.UUID, error)    { return a.actor, nil }

type fakeRepo struct{ gravadoPor uuid.UUID }

func (r *fakeRepo) GetByNumber(context.Context, int) (*entity.PlanningParam, error) { return nil, nil }
func (r *fakeRepo) GetByKey(context.Context, string) (*entity.PlanningParam, error) { return nil, nil }
func (r *fakeRepo) List(context.Context) ([]*entity.PlanningParam, error)           { return nil, nil }
func (r *fakeRepo) Update(_ context.Context, n int, v string, por uuid.UUID) (*entity.PlanningParam, error) {
	r.gravadoPor = por
	return &entity.PlanningParam{ParamNumber: n, Value: v, UpdatedBy: por}, nil
}

// O autor da alteração é o da sessão: um updated_by no corpo é ignorado.
func TestUpdate_AutorVemDaSessaoENaoDoCorpo(t *testing.T) {
	actor := uuid.New()
	repo := &fakeRepo{}
	uc := &UpdatePlanningParamUseCase{Repo: repo, Auth: &fakeAuth{actor: actor}}

	var dto request.UpdatePlanningParamDTO
	corpo := `{"param_number":1,"value":"true","updated_by":"` + uuid.New().String() + `"}`
	if err := json.Unmarshal([]byte(corpo), &dto); err != nil {
		t.Fatal(err)
	}
	if dto.UpdatedBy != uuid.Nil {
		t.Fatalf("updated_by do corpo não pode chegar ao caso de uso: %s", dto.UpdatedBy)
	}
	if _, err := uc.Execute(context.Background(), dto); err != nil {
		t.Fatal(err)
	}
	if repo.gravadoPor != actor {
		t.Fatalf("autor gravado %s, esperado o da sessão %s", repo.gravadoPor, actor)
	}
}
