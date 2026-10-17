//go:build integration

package cost_center_test

import (
	"errors"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
	ccrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O uso do centro industrial é contado em todas as tabelas que o referenciam
// (pelo id e pelo código) contra o esquema real, e outra empresa não altera nem
// exclui o centro.
func TestIntegration_ManutencaoCentroIndustrial(t *testing.T) {
	pool := testutil.Pool(t)
	q, _ := testutil.Queries(t)
	repo := ccrepo.NewCostCenterRepositorySQLC(q)
	manut := ccrepo.NewManutencaoPG(pool)
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM cost_centers WHERE enterprise_id = ANY($1) AND parent_code IS NOT NULL`, []int64{a.ID, b.ID})
		testutil.Exec(t, pool, `DELETE FROM cost_centers WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	base := int32(testutil.UniqueCode()%1_000_000) + 1_000_000_000
	hoje := time.Now()
	testutil.Exec(t, pool, `INSERT INTO cost_centers(code,description,type,start_date,created_by,enterprise_id) VALUES($1,'Fábrica','PRODUCTIVE',$2::date,$3,$4)`, base, hoje, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO cost_centers(code,description,type,start_date,created_by,enterprise_id,parent_code) VALUES($1,'Usinagem','PRODUCTIVE',$2::date,$3,$4,$5)`, base+1, hoje, actor, a.ID, base)

	pai, err := repo.GetByCode(a.Ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	usos, err := manut.Usos(a.Ctx, pai)
	if err != nil {
		t.Fatalf("contando o uso: %v", err)
	}
	if len(usos) != 1 || usos[0].Onde != "centros de custo filhos" || usos[0].Quantidade != 1 {
		t.Fatalf("o pai tem 1 filho e nenhum outro uso: %+v", usos)
	}
	if n, err := manut.FilhosAtivos(a.Ctx, base); err != nil || n != 1 {
		t.Fatalf("filhos ativos: %d %v", n, err)
	}

	alt := repository.AlteracaoCentroCusto{ID: pai.ID, Description: "Fábrica 1", Type: "PRODUCTIVE", StartDate: pai.StartDate, IsActive: true}
	if err := manut.Alterar(b.Ctx, alt); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("B não pode alterar o centro de A: %v", err)
	}
	if err := manut.Excluir(b.Ctx, pai.ID); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("B não pode excluir o centro de A: %v", err)
	}
	if err := manut.Alterar(a.Ctx, alt); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetByCode(a.Ctx, base); got.Description != "Fábrica 1" {
		t.Fatalf("não alterou: %+v", got)
	}

	filho, _ := repo.GetByCode(a.Ctx, base+1)
	if err := manut.Alterar(a.Ctx, repository.AlteracaoCentroCusto{ID: filho.ID, Description: filho.Description, ParentCode: filho.ParentCode, Type: "PRODUCTIVE", StartDate: filho.StartDate, IsActive: false}); err != nil {
		t.Fatal(err)
	}
	todos, err := manut.ListarComInativos(a.Ctx)
	if err != nil || len(todos) != 2 {
		t.Fatalf("listagem do cadastro com inativos (2): %d %v", len(todos), err)
	}
	if err := manut.Excluir(a.Ctx, filho.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByCode(a.Ctx, base+1); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("centro excluído ainda aparece: %v", err)
	}
}
