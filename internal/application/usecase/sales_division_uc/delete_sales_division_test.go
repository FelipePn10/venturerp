package sales_division_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	sdrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type deleteDivisionAuth struct {
	ports.AuthService
	allowed bool
}

func (a deleteDivisionAuth) CanDeleteSalesDivision(context.Context) bool { return a.allowed }

type deleteDivisionRepo struct {
	sdrepo.SalesDivisionRepository
	linked  bool
	deleted bool
}

func (r *deleteDivisionRepo) HasReferences(context.Context, int64) (bool, error) {
	return r.linked, nil
}
func (r *deleteDivisionRepo) Delete(context.Context, int64) error {
	r.deleted = true
	return nil
}

func TestDeleteSalesDivisionFree(t *testing.T) {
	repo := &deleteDivisionRepo{}
	err := (&DeleteSalesDivisionUseCase{Repo: repo, Auth: deleteDivisionAuth{allowed: true}}).Execute(context.Background(), 10)
	if err != nil || !repo.deleted {
		t.Fatalf("exclusão livre falhou: err=%v deleted=%v", err, repo.deleted)
	}
}

func TestDeleteSalesDivisionLinkedReturnsConflict(t *testing.T) {
	repo := &deleteDivisionRepo{linked: true}
	err := (&DeleteSalesDivisionUseCase{Repo: repo, Auth: deleteDivisionAuth{allowed: true}}).Execute(context.Background(), 10)
	if _, ok := errorsuc.AsConflict(err); !ok || repo.deleted {
		t.Fatalf("esperava conflito sem excluir: err=%v deleted=%v", err, repo.deleted)
	}
}

func TestDeleteSalesDivisionWithoutPermissionReturnsForbiddenError(t *testing.T) {
	err := (&DeleteSalesDivisionUseCase{Repo: &deleteDivisionRepo{}, Auth: deleteDivisionAuth{}}).Execute(context.Background(), 10)
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("esperava erro de permissão: %v", err)
	}
}
