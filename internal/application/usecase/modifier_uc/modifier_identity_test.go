package modifier_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
	"github.com/google/uuid"
)

type identityAuth struct {
	ports.AuthService
	actor uuid.UUID
}

func (a identityAuth) CanCreateModifier(context.Context) bool    { return true }
func (a identityAuth) UserID(context.Context) (uuid.UUID, error) { return a.actor, nil }

type identityRepo struct {
	created *entity.Modifier
}

func (r *identityRepo) Create(_ context.Context, m *entity.Modifier) (*entity.Modifier, error) {
	r.created = m
	return m, nil
}
func (r *identityRepo) List(context.Context) ([]*entity.Modifier, error)       { return nil, nil }
func (r *identityRepo) GetByID(context.Context, int) (*entity.Modifier, error) { return nil, nil }
func (r *identityRepo) Update(context.Context, *entity.Modifier) (*entity.Modifier, error) {
	return nil, nil
}

// O autor do modificador tem de vir do usuário autenticado. Quando vinha do
// corpo da requisição, a tela que não enviava created_by gravava um UUID vazio
// e o banco recusava por chave estrangeira.
func TestCreateModifierTakesAuthorFromAuthenticatedUser(t *testing.T) {
	actor := uuid.New()
	repo := &identityRepo{}
	uc := &CreateModifierUseCase{Repo: repo, Auth: identityAuth{actor: actor}}

	if _, err := uc.Execute(context.Background(), &entity.Modifier{Description: "Teste"}); err != nil {
		t.Fatal(err)
	}
	if repo.created == nil {
		t.Fatal("modificador não foi criado")
	}
	if repo.created.CreatedBy != actor {
		t.Fatalf("autor = %v, esperado %v", repo.created.CreatedBy, actor)
	}
}
