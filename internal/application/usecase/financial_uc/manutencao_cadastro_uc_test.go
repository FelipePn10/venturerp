package financial_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type manutAuth struct {
	ports.AuthService
	editar, excluir bool
}

func (a manutAuth) CanEditCadastroFinanceiro(context.Context) bool   { return a.editar }
func (a manutAuth) CanDeleteCadastroFinanceiro(context.Context) bool { return a.excluir }
func (a manutAuth) CanCreatePlanoContas(context.Context) bool        { return true }

type manutRepo struct {
	repository.FinancialRepository
	planos   map[string]*entity.PlanoContas // por código
	usos     []entity.UsoCadastro
	filhas   int64
	salvo    *entity.PlanoContas
	excluido bool
	centro   *entity.CentroCusto
	salvoCC  *entity.CentroCusto
}

func (r *manutRepo) GetPlanoContas(_ context.Context, id int64) (*entity.PlanoContas, error) {
	for _, p := range r.planos {
		if p.ID == id {
			c := *p
			return &c, nil
		}
	}
	return nil, errorsuc.NewNotFoundError("plano de contas não encontrado")
}
func (r *manutRepo) GetPlanoContasPorCodigo(_ context.Context, codigo string) (*entity.PlanoContas, error) {
	if p, ok := r.planos[codigo]; ok {
		c := *p
		return &c, nil
	}
	return nil, errorsuc.NewNotFoundError("plano de contas não encontrado")
}
func (r *manutRepo) UsosPlanoContas(context.Context, int64) ([]entity.UsoCadastro, error) {
	return r.usos, nil
}
func (r *manutRepo) FilhasAtivasPlanoContas(context.Context, string) (int64, error) {
	return r.filhas, nil
}
func (r *manutRepo) UpdatePlanoContas(_ context.Context, p *entity.PlanoContas) (*entity.PlanoContas, error) {
	r.salvo = p
	return p, nil
}
func (r *manutRepo) DeletePlanoContas(context.Context, int64) error { r.excluido = true; return nil }
func (r *manutRepo) CreatePlanoContas(_ context.Context, p *entity.PlanoContas) (*entity.PlanoContas, error) {
	r.salvo = p
	return p, nil
}
func (r *manutRepo) GetCentroCusto(context.Context, int64) (*entity.CentroCusto, error) {
	c := *r.centro
	return &c, nil
}
func (r *manutRepo) UsosCentroCusto(context.Context, int64) ([]entity.UsoCadastro, error) {
	return r.usos, nil
}
func (r *manutRepo) UpdateCentroCusto(_ context.Context, c *entity.CentroCusto) (*entity.CentroCusto, error) {
	r.salvoCC = c
	return c, nil
}
func (r *manutRepo) DeleteCentroCusto(context.Context, int64) error { r.excluido = true; return nil }

func pv[T any](v T) *T { return &v }

func novoRepoPlanos() *manutRepo {
	return &manutRepo{planos: map[string]*entity.PlanoContas{
		"3":      {ID: 1, Codigo: "3", Descricao: "Despesas", Tipo: "DESPESA", Natureza: "DEBITO", Nivel: 1, IsActive: true},
		"3.1":    {ID: 2, Codigo: "3.1", Descricao: "Operacionais", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3"), Nivel: 2, IsActive: true},
		"3.1.01": {ID: 3, Codigo: "3.1.01", Descricao: "Energia", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1"), Nivel: 3, IsActive: true},
		"9":      {ID: 4, Codigo: "9", Descricao: "Inativa", Tipo: "DESPESA", Natureza: "DEBITO", Nivel: 1, IsActive: false},
	}}
}

func ehConflito(err error) bool {
	var c *errorsuc.ConflictError
	return errors.As(err, &c)
}

func ehValidacao(err error) bool {
	_, ok := errorsuc.AsValidation(err)
	return ok
}

func TestUpdatePlanoContas_EmUsoSoMudaDescricaoESituacao(t *testing.T) {
	ctx := context.Background()
	repo := novoRepoPlanos()
	repo.usos = []entity.UsoCadastro{{Onde: "títulos a pagar", Quantidade: 4}}
	uc := &UpdatePlanoContasUseCase{Repo: repo, Auth: manutAuth{editar: true}}

	// Renomear e desativar: permitido mesmo em uso.
	out, err := uc.Execute(ctx, 3, request.UpdatePlanoContasDTO{Codigo: "3.1.01", Descricao: "Energia elétrica", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1"), IsActive: pv(false)})
	if err != nil {
		t.Fatalf("renomear/desativar em uso deveria passar: %v", err)
	}
	if out.Descricao != "Energia elétrica" || out.IsActive {
		t.Fatalf("não gravou descrição/situação: %+v", out)
	}

	// Trocar tipo, código ou conta-pai: recusado com a lista de usos.
	casos := []request.UpdatePlanoContasDTO{
		{Codigo: "3.1.01", Descricao: "Energia", Tipo: "RECEITA", Natureza: "DEBITO", ParentCode: pv("3.1")},
		{Codigo: "3.1.02", Descricao: "Energia", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1")},
		{Codigo: "3.1.01", Descricao: "Energia", Tipo: "DESPESA", Natureza: "CREDITO", ParentCode: pv("3.1")},
		{Codigo: "3.1.01", Descricao: "Energia", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3")},
	}
	for i, dto := range casos {
		_, err := uc.Execute(ctx, 3, dto)
		if !ehConflito(err) || !strings.Contains(err.Error(), "títulos a pagar (4)") {
			t.Fatalf("caso %d: esperava conflito listando o uso, veio %v", i, err)
		}
	}
}

func TestUpdatePlanoContas_SemUsoMudaEstruturaComConferencia(t *testing.T) {
	ctx := context.Background()
	repo := novoRepoPlanos()
	uc := &UpdatePlanoContasUseCase{Repo: repo, Auth: manutAuth{editar: true}}

	out, err := uc.Execute(ctx, 3, request.UpdatePlanoContasDTO{Codigo: "3.2", Descricao: "Energia", Tipo: "despesa", Natureza: "debito", ParentCode: pv("3")})
	if err != nil {
		t.Fatalf("sem uso deveria permitir trocar a estrutura: %v", err)
	}
	if out.Codigo != "3.2" || out.Nivel != 2 || out.Tipo != "DESPESA" {
		t.Fatalf("nível/tipo não recalculados: %+v", out)
	}

	ruins := map[string]request.UpdatePlanoContasDTO{
		"pai inexistente": {Codigo: "3.1.01", Descricao: "x", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("7")},
		"pai inativo":     {Codigo: "3.1.01", Descricao: "x", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("9")},
		"pai = ela mesma": {Codigo: "3.1.01", Descricao: "x", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1.01")},
		"tipo inválido":   {Codigo: "3.1.01", Descricao: "x", Tipo: "CUSTO", Natureza: "DEBITO", ParentCode: pv("3.1")},
		"sem descrição":   {Codigo: "3.1.01", Descricao: " ", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1")},
	}
	for nome, dto := range ruins {
		if _, err := uc.Execute(ctx, 3, dto); !ehValidacao(err) {
			t.Fatalf("%s: esperava validação, veio %v", nome, err)
		}
	}

	// Ciclo: pôr "3" debaixo de "3.1" (que é filha de "3").
	if _, err := uc.Execute(ctx, 1, request.UpdatePlanoContasDTO{Codigo: "3", Descricao: "Despesas", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1")}); !ehValidacao(err) || !strings.Contains(err.Error(), "ciclo") {
		t.Fatalf("esperava recusa por ciclo, veio %v", err)
	}

	// Código já usado por outra conta.
	if _, err := uc.Execute(ctx, 3, request.UpdatePlanoContasDTO{Codigo: "3.1", Descricao: "x", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3")}); !ehConflito(err) {
		t.Fatalf("esperava conflito de código duplicado, veio %v", err)
	}
}

func TestUpdatePlanoContas_SituacaoRespeitaArvore(t *testing.T) {
	ctx := context.Background()
	repo := novoRepoPlanos()
	repo.filhas = 1
	uc := &UpdatePlanoContasUseCase{Repo: repo, Auth: manutAuth{editar: true}}
	if _, err := uc.Execute(ctx, 2, request.UpdatePlanoContasDTO{Codigo: "3.1", Descricao: "Operacionais", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3"), IsActive: pv(false)}); !ehConflito(err) {
		t.Fatalf("desativar com filha ativa deveria ser recusado, veio %v", err)
	}

	repo = novoRepoPlanos()
	repo.planos["9.1"] = &entity.PlanoContas{ID: 5, Codigo: "9.1", Descricao: "Filha", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("9"), Nivel: 2, IsActive: false}
	uc = &UpdatePlanoContasUseCase{Repo: repo, Auth: manutAuth{editar: true}}
	if _, err := uc.Execute(ctx, 5, request.UpdatePlanoContasDTO{Codigo: "9.1", Descricao: "Filha", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("9"), IsActive: pv(true)}); !ehConflito(err) {
		t.Fatalf("reativar com pai inativo deveria ser recusado, veio %v", err)
	}
}

func TestDeletePlanoContas(t *testing.T) {
	ctx := context.Background()
	repo := novoRepoPlanos()
	if err := (&DeletePlanoContasUseCase{Repo: repo, Auth: manutAuth{excluir: false}}).Execute(ctx, 3); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("sem permissão deveria recusar, veio %v", err)
	}
	repo.usos = []entity.UsoCadastro{{Onde: "contas filhas", Quantidade: 1}}
	if err := (&DeletePlanoContasUseCase{Repo: repo, Auth: manutAuth{excluir: true}}).Execute(ctx, 2); !ehConflito(err) || repo.excluido {
		t.Fatalf("em uso não pode excluir, veio %v (excluido=%v)", err, repo.excluido)
	}
	repo.usos = nil
	if err := (&DeletePlanoContasUseCase{Repo: repo, Auth: manutAuth{excluir: true}}).Execute(ctx, 3); err != nil || !repo.excluido {
		t.Fatalf("sem uso deveria excluir: %v", err)
	}
}

func TestCentroCusto_RegrasDeEdicaoEExclusao(t *testing.T) {
	ctx := context.Background()
	repo := &manutRepo{centro: &entity.CentroCusto{ID: 7, Codigo: "100", Descricao: "Usinagem", Tipo: "PRODUTIVO", IsActive: true},
		usos: []entity.UsoCadastro{{Onde: "títulos a pagar", Quantidade: 2}}}
	up := &UpdateCentroCustoUseCase{Repo: repo, Auth: manutAuth{editar: true}}
	if _, err := up.Execute(ctx, 7, request.UpdateCentroCustoDTO{Codigo: "100", Descricao: "Usinagem CNC", Tipo: "PRODUTIVO", IsActive: pv(false)}); err != nil {
		t.Fatalf("renomear/desativar em uso deveria passar: %v", err)
	}
	if _, err := up.Execute(ctx, 7, request.UpdateCentroCustoDTO{Codigo: "101", Descricao: "Usinagem", Tipo: "PRODUTIVO"}); !ehConflito(err) {
		t.Fatalf("trocar código em uso deveria ser recusado, veio %v", err)
	}
	if err := (&DeleteCentroCustoUseCase{Repo: repo, Auth: manutAuth{excluir: true}}).Execute(ctx, 7); !ehConflito(err) {
		t.Fatalf("excluir em uso deveria ser recusado, veio %v", err)
	}
	repo.usos = nil
	if _, err := up.Execute(ctx, 7, request.UpdateCentroCustoDTO{Codigo: "101", Descricao: "Usinagem", Tipo: "AUXILIAR"}); err != nil {
		t.Fatalf("sem uso deveria permitir trocar código/tipo: %v", err)
	}
	if err := (&DeleteCentroCustoUseCase{Repo: repo, Auth: manutAuth{excluir: true}}).Execute(ctx, 7); err != nil || !repo.excluido {
		t.Fatalf("sem uso deveria excluir: %v", err)
	}
}

func TestCreatePlanoContas_ConfereContaPai(t *testing.T) {
	ctx := context.Background()
	repo := novoRepoPlanos()
	uc := &CreatePlanoContasUseCase{Repo: repo, Auth: manutAuth{}}
	if _, err := uc.Execute(ctx, request.CreatePlanoContasDTO{Codigo: "5.1", Descricao: "x", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("5")}); !ehValidacao(err) {
		t.Fatalf("pai inexistente deveria ser recusado, veio %v", err)
	}
	if _, err := uc.Execute(ctx, request.CreatePlanoContasDTO{Codigo: "3.1.02", Descricao: "Água", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: pv("3.1")}); err != nil {
		t.Fatalf("com pai válido deveria criar: %v", err)
	}
	if repo.salvo.Nivel != 3 {
		t.Fatalf("nível deveria vir do código (3), veio %d", repo.salvo.Nivel)
	}
}
