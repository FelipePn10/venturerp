package user_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	userentity "github.com/FelipePn10/panossoerp/internal/domain/user/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/user/repository"
)

type registerRepo struct {
	repository.UserRepository
	createdEnterpriseID int64
}

func (r *registerRepo) Create(_ context.Context, _ *userentity.User, enterpriseID int64) error {
	r.createdEnterpriseID = enterpriseID
	return nil
}

type registerAuth struct {
	enterpriseID int64
	err          error
}

func (a registerAuth) EnterpriseID(context.Context) (int64, error) {
	return a.enterpriseID, a.err
}

func TestRegisterUsesAuthenticatedEnterprise(t *testing.T) {
	repo := &registerRepo{}
	uc := NewRegisterUserUseCase(repo, registerAuth{enterpriseID: 42})
	err := uc.Execute(context.Background(), request.RegisterUserDTO{
		Name: "New User", Email: "new@example.com", Password: "Password123!", EnterpriseCode: 999,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.createdEnterpriseID != 42 {
		t.Fatalf("created enterprise = %d, want authenticated enterprise 42", repo.createdEnterpriseID)
	}
}

func TestRegisterRejectsMissingAuthenticatedEnterprise(t *testing.T) {
	repo := &registerRepo{}
	uc := NewRegisterUserUseCase(repo, registerAuth{err: errors.New("missing identity")})
	err := uc.Execute(context.Background(), request.RegisterUserDTO{Name: "New User", Email: "new@example.com", Password: "Password123!"})
	if !errors.Is(err, ErrRegisterUserForbidden) {
		t.Fatalf("error = %v, want ErrRegisterUserForbidden", err)
	}
	if repo.createdEnterpriseID != 0 {
		t.Fatal("repository must not be called without an authenticated enterprise")
	}
}
