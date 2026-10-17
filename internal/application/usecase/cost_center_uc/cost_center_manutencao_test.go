package cost_center_uc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

type fakeAuth struct {
	ports.AuthService
	admin bool
}

func (fakeAuth) CanEditCostCenter(context.Context) bool     { return true }
func (a fakeAuth) CanDeleteCostCenter(context.Context) bool { return a.admin }
func (fakeAuth) CanCreateCostCenter(context.Context) bool   { return true }

type fakeRepo struct {
	repository.CostCenterRepository
	centros map[int32]*entity.CostCenter
}

func (r *fakeRepo) GetByCode(_ context.Context, code int32) (*entity.CostCenter, error) {
	if c, ok := r.centros[code]; ok {
		v := *c
		return &v, nil
	}
	return nil, errorsuc.NewNotFoundError("centro de custo não encontrado")
}
func (r *fakeRepo) Create(_ context.Context, c *entity.CostCenter) (*entity.CostCenter, error) {
	return c, nil
}

type fakeManut struct {
	repository.CostCenterManutencaoRepository
	repo     *fakeRepo
	usos     []repository.UsoCentroCusto
	filhos   int64
	excluido bool
}

func (m *fakeManut) Usos(context.Context, *entity.CostCenter) ([]repository.UsoCentroCusto, error) {
	return m.usos, nil
}
func (m *fakeManut) FilhosAtivos(context.Context, int32) (int64, error) { return m.filhos, nil }
func (m *fakeManut) Alterar(_ context.Context, a repository.AlteracaoCentroCusto) error {
	for _, c := range m.repo.centros {
		if c.ID == a.ID {
			c.Description, c.ParentCode, c.Type, c.IsRatio = a.Description, a.ParentCode, types.TypeCC(a.Type), a.IsRatio
			c.StartDate, c.EndDate, c.IsActive = a.StartDate, a.EndDate, a.IsActive
		}
	}
	return nil
}
func (m *fakeManut) Excluir(context.Context, int64) error { m.excluido = true; return nil }

func pc(v int32) *int32 { return &v }
func pb(v bool) *bool   { return &v }

func cenario() (*fakeRepo, *fakeManut) {
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{centros: map[int32]*entity.CostCenter{
		10: {ID: 1, Code: 10, Description: "Fábrica", Type: types.CCProductive, StartDate: inicio, IsActive: true},
		11: {ID: 2, Code: 11, Description: "Usinagem", Type: types.CCProductive, ParentCode: pc(10), StartDate: inicio, IsActive: true},
		90: {ID: 3, Code: 90, Description: "Antigo", Type: types.CCAuxiliary, StartDate: inicio, IsActive: false},
	}}
	return repo, &fakeManut{repo: repo}
}

func conflito(err error) bool {
	var c *errorsuc.ConflictError
	return errors.As(err, &c)
}

func TestUpdateCostCenter_EmUso(t *testing.T) {
	ctx := context.Background()
	repo, manut := cenario()
	manut.usos = []repository.UsoCentroCusto{{Onde: "máquinas", Quantidade: 3}}
	uc := &UpdateCostCenterUseCase{Repo: repo, Manut: manut, Auth: fakeAuth{}}

	fim := "2026-12-31"
	out, err := uc.Execute(ctx, 11, request.UpdateCostCenterDTO{Description: "Usinagem CNC", ParentCode: pc(10), Type: "PRODUCTIVE", EndDate: &fim})
	if err != nil {
		t.Fatalf("descrição e data de fim mudam mesmo em uso: %v", err)
	}
	if out.Description != "Usinagem CNC" || out.EndDate == nil {
		t.Fatalf("não gravou: %+v", out)
	}
	for nome, dto := range map[string]request.UpdateCostCenterDTO{
		"tipo":   {Description: "x", ParentCode: pc(10), Type: "AUXILIARY"},
		"pai":    {Description: "x", Type: "PRODUCTIVE"},
		"rateio": {Description: "x", ParentCode: pc(10), Type: "PRODUCTIVE", IsRatio: pb(true)},
		"início": {Description: "x", ParentCode: pc(10), Type: "PRODUCTIVE", StartDate: "2026-02-01"},
	} {
		if _, err := uc.Execute(ctx, 11, dto); !conflito(err) || !strings.Contains(err.Error(), "máquinas (3)") {
			t.Fatalf("%s: esperava conflito listando o uso, veio %v", nome, err)
		}
	}
}

func TestUpdateCostCenter_SemUsoConfereArvore(t *testing.T) {
	ctx := context.Background()
	repo, manut := cenario()
	uc := &UpdateCostCenterUseCase{Repo: repo, Manut: manut, Auth: fakeAuth{}}
	if _, err := uc.Execute(ctx, 10, request.UpdateCostCenterDTO{Description: "Fábrica", Type: "PRODUCTIVE", ParentCode: pc(11)}); err == nil || !strings.Contains(err.Error(), "ciclo") {
		t.Fatalf("esperava recusa por ciclo, veio %v", err)
	}
	if _, err := uc.Execute(ctx, 11, request.UpdateCostCenterDTO{Description: "x", Type: "PRODUCTIVE", ParentCode: pc(90)}); err == nil || !strings.Contains(err.Error(), "inativo") {
		t.Fatalf("esperava recusa por pai inativo, veio %v", err)
	}
	if _, err := uc.Execute(ctx, 11, request.UpdateCostCenterDTO{Description: "x", Type: "OUTRO", ParentCode: pc(10)}); err == nil {
		t.Fatal("tipo inválido deveria ser recusado")
	}
	if _, err := uc.Execute(ctx, 11, request.UpdateCostCenterDTO{Description: "Usinagem", Type: "AUXILIARY"}); err != nil {
		t.Fatalf("sem uso, trocar tipo e tirar o pai deveria passar: %v", err)
	}

	manut.filhos = 1
	if _, err := uc.Execute(ctx, 10, request.UpdateCostCenterDTO{Description: "Fábrica", Type: "PRODUCTIVE", IsActive: pb(false)}); !conflito(err) {
		t.Fatalf("desativar com filho ativo deveria ser recusado, veio %v", err)
	}
}

func TestDeleteCostCenter(t *testing.T) {
	ctx := context.Background()
	repo, manut := cenario()
	if err := (&DeleteCostCenterUseCase{Repo: repo, Manut: manut, Auth: fakeAuth{}}).Execute(ctx, 90); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("só ADMIN exclui, veio %v", err)
	}
	manut.usos = []repository.UsoCentroCusto{{Onde: "ordens de produção", Quantidade: 1}}
	if err := (&DeleteCostCenterUseCase{Repo: repo, Manut: manut, Auth: fakeAuth{admin: true}}).Execute(ctx, 90); !conflito(err) || manut.excluido {
		t.Fatalf("em uso não exclui, veio %v", err)
	}
	manut.usos = nil
	if err := (&DeleteCostCenterUseCase{Repo: repo, Manut: manut, Auth: fakeAuth{admin: true}}).Execute(ctx, 90); err != nil || !manut.excluido {
		t.Fatalf("sem uso deveria excluir: %v", err)
	}
}

func TestCreateCostCenter_ConferePai(t *testing.T) {
	ctx := context.Background()
	repo, _ := cenario()
	uc := &CreateCostCenterUseCase{Repo: repo, Auth: fakeAuth{}}
	if _, err := uc.Execute(ctx, request.CreateCostCenterDTO{Code: 12, Description: "x", Type: types.CCProductive, StartDate: "2026-01-01", ParentCode: pc(55)}); err == nil {
		t.Fatal("pai inexistente deveria ser recusado")
	}
	if _, err := uc.Execute(ctx, request.CreateCostCenterDTO{Code: 12, Description: "x", Type: types.CCProductive, StartDate: "2026-01-01", ParentCode: pc(10)}); err != nil {
		t.Fatalf("pai válido deveria criar: %v", err)
	}
}
