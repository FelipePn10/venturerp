package group_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/group/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/group/repository"
	"github.com/google/uuid"
)

type groupAuthStub struct {
	ports.AuthService
	actor uuid.UUID
}

func (a groupAuthStub) CanCreateGroup(context.Context) bool       { return true }
func (a groupAuthStub) UserID(context.Context) (uuid.UUID, error) { return a.actor, nil }

type groupRepoStub struct {
	repository.GroupRepository
	created *entity.Group
}

func (r *groupRepoStub) Create(_ context.Context, group *entity.Group) (*entity.Group, error) {
	r.created = group
	return group, nil
}

func TestCreateGroupUsesAuthenticatedActor(t *testing.T) {
	actor := uuid.New()
	repo := &groupRepoStub{}
	uc := NewCreateGroupUseCase(repo, groupAuthStub{actor: actor})
	if _, err := uc.Execute(context.Background(), request.CreateGroupDTO{Code: 10, Description: "Conjuntos"}); err != nil {
		t.Fatal(err)
	}
	if repo.created == nil || repo.created.CreatedBy != actor {
		t.Fatalf("created_by=%v, esperado ator JWT %v", repo.created, actor)
	}
}

func (*groupRepoStub) List(context.Context) ([]*entity.Group, error) { return nil, nil }

func TestDescricaoRepetidaEhRecusada(t *testing.T) {
	repo := &gruposFixos{}
	if err := descricaoLivre(context.Background(), repo, " chapas ", 0); err == nil {
		t.Fatal("o mesmo nome com outra grafia não pode virar outro grupo")
	}
	if err := descricaoLivre(context.Background(), repo, "CHAPAS", 10); err != nil {
		t.Fatalf("o próprio grupo mantém o nome: %v", err)
	}
}

type gruposFixos struct{ groupRepoStub }

func (*gruposFixos) List(context.Context) ([]*entity.Group, error) {
	return []*entity.Group{{Code: 10, Description: "Chapas"}}, nil
}

type grupoDuplicado struct {
	gruposFixos
	salvo *entity.Group
}

func (*grupoDuplicado) GetByCode(context.Context, int) (*entity.Group, error) {
	return &entity.Group{Code: 20, Description: "CHAPAS ", IsActive: true}, nil
}
func (r *grupoDuplicado) Update(_ context.Context, g *entity.Group) (*entity.Group, error) {
	r.salvo = g
	return g, nil
}

func TestDuplicadoAntigoPodeSerInativado(t *testing.T) {
	repo := &grupoDuplicado{}
	uc := NewUpdateGroupUseCase(repo, groupAuthStub{})
	inativo := false
	if _, err := uc.Execute(context.Background(), 20, "CHAPAS", &inativo); err != nil {
		t.Fatalf("manter o nome e inativar não é cadastrar duplicado: %v", err)
	}
	if repo.salvo == nil || repo.salvo.IsActive {
		t.Fatalf("o grupo fica inativo: %+v", repo.salvo)
	}
	if _, err := uc.Execute(context.Background(), 20, "Chapas Novas", nil); err != nil {
		t.Fatal(err)
	}
}
