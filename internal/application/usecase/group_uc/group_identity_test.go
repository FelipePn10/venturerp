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
