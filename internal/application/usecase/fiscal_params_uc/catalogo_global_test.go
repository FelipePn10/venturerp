package fiscal_params_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type papel bool

func (p papel) CanManageGlobalCatalogs(context.Context) bool { return bool(p) }

type catalogoRepo struct {
	repository.FiscalParamsRepository
	gravou int
}

func (r *catalogoRepo) CreateCFOP(_ context.Context, c *entity.CFOP) (*entity.CFOP, error) {
	r.gravou++
	return c, nil
}
func (r *catalogoRepo) UpdateCFOP(_ context.Context, c *entity.CFOP) (*entity.CFOP, error) {
	r.gravou++
	return c, nil
}
func (r *catalogoRepo) CreateLegalDevice(_ context.Context, d *entity.LegalDevice) (*entity.LegalDevice, error) {
	r.gravou++
	return d, nil
}
func (r *catalogoRepo) UpdateLegalDevice(_ context.Context, d *entity.LegalDevice) (*entity.LegalDevice, error) {
	r.gravou++
	return d, nil
}

// CFOP e dispositivo legal valem para todas as empresas do banco: só ADMIN
// inclui ou altera; USER (ou caso de uso sem autenticação) é recusado sem gravar.
func TestCatalogoGlobalSoAdmin(t *testing.T) {
	ctx := context.Background()
	for _, auth := range []catalogAuth{papel(false), nil} {
		repo := &catalogoRepo{}
		cfop := &CFOPUseCase{Repo: repo, Auth: auth}
		disp := &LegalDeviceUseCase{Repo: repo, Auth: auth}
		erros := []error{}
		_, err := cfop.Create(ctx, request.CreateCFOPDTO{Code: 5949, Description: "Outra"})
		erros = append(erros, err)
		_, err = cfop.Update(ctx, request.UpdateCFOPDTO{ID: 1, Code: 5949, Description: "Outra"})
		erros = append(erros, err)
		_, err = disp.Create(ctx, request.CreateLegalDeviceDTO{Type: "ICMS", Description: "Art. 1"})
		erros = append(erros, err)
		_, err = disp.Update(ctx, request.UpdateLegalDeviceDTO{ID: 1, Type: "ICMS", Description: "Art. 1"})
		erros = append(erros, err)
		for i, e := range erros {
			if !errors.Is(e, errorsuc.ErrUnauthorized) {
				t.Fatalf("operação %d sem ADMIN: erro = %v; esperado não autorizado", i, e)
			}
		}
		if repo.gravou != 0 {
			t.Fatalf("gravou %d vez(es) sem ADMIN", repo.gravou)
		}
	}
	repo := &catalogoRepo{}
	if _, err := (&CFOPUseCase{Repo: repo, Auth: papel(true)}).Create(ctx, request.CreateCFOPDTO{Code: 5949, Description: "Outra"}); err != nil {
		t.Fatal(err)
	}
	if _, err := (&LegalDeviceUseCase{Repo: repo, Auth: papel(true)}).Create(ctx, request.CreateLegalDeviceDTO{Type: "ICMS", Description: "Art. 1"}); err != nil {
		t.Fatal(err)
	}
	if repo.gravou != 2 {
		t.Fatalf("ADMIN gravou %d vez(es), esperado 2", repo.gravou)
	}
}
