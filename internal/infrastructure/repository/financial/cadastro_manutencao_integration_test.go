//go:build integration

package financial_test

import (
	"context"
	"errors"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func usoEm(usos []entity.UsoCadastro, onde string) int64 {
	for _, u := range usos {
		if u.Onde == onde {
			return u.Quantidade
		}
	}
	return 0
}

// Manutenção do plano de contas e do centro de custo: o uso é contado em todas
// as tabelas que apontam para o cadastro (a consulta roda contra o esquema real),
// e nada atravessa de uma empresa para a outra.
func TestIntegration_ManutencaoPlanoECentro(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		testutil.Exec(t, pool, `DELETE FROM contas_pagar WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM plano_contas WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM centros_custo WHERE enterprise_id = ANY($1)`, ids)
	})

	pai, err := repo.CreatePlanoContas(a.Ctx, &entity.PlanoContas{Codigo: "3", Descricao: "Despesas", Tipo: "DESPESA", Natureza: "DEBITO", Nivel: 1, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	paiCod := "3"
	filha, err := repo.CreatePlanoContas(a.Ctx, &entity.PlanoContas{Codigo: "3.1", Descricao: "Energia", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: &paiCod, Nivel: 2, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	// Mesmo código na outra empresa: não pode contar como filha da conta de A.
	if _, err := repo.CreatePlanoContas(b.Ctx, &entity.PlanoContas{Codigo: "3.9", Descricao: "Outra", Tipo: "DESPESA", Natureza: "DEBITO", ParentCode: &paiCod, Nivel: 2, IsActive: true}); err != nil {
		t.Fatal(err)
	}
	centro, err := repo.CreateCentroCusto(a.Ctx, &entity.CentroCusto{Codigo: "100", Descricao: "Usinagem", Tipo: "PRODUTIVO", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}

	usos, err := repo.UsosPlanoContas(a.Ctx, pai.ID)
	if err != nil {
		t.Fatalf("contando o uso do plano: %v", err)
	}
	if n := usoEm(usos, "contas filhas"); n != 1 {
		t.Fatalf("a conta 3 de A tem 1 filha (a de B não conta), veio %d: %+v", n, usos)
	}
	if n, err := repo.FilhasAtivasPlanoContas(a.Ctx, "3"); err != nil || n != 1 {
		t.Fatalf("filhas ativas: %d %v", n, err)
	}

	hoje := time.Now()
	testutil.Exec(t, pool, `INSERT INTO contas_pagar(numero_documento,valor_bruto,data_emissao,data_vencimento,plano_contas_id,centro_custo_id,criado_por,enterprise_id)
		VALUES('CAD-1',10,$1::date,$1::date,$2,$3,$4,$5)`, hoje, filha.ID, centro.ID, actor, a.ID)
	usos, err = repo.UsosPlanoContas(a.Ctx, filha.ID)
	if err != nil || usoEm(usos, "títulos a pagar") != 1 {
		t.Fatalf("o título a pagar tem de aparecer no uso do plano: %+v %v", usos, err)
	}
	usosCC, err := repo.UsosCentroCusto(a.Ctx, centro.ID)
	if err != nil || usoEm(usosCC, "títulos a pagar") != 1 {
		t.Fatalf("o título a pagar tem de aparecer no uso do centro: %+v %v", usosCC, err)
	}

	// Outra empresa não enxerga, não altera e não exclui.
	if _, err := repo.GetPlanoContas(b.Ctx, filha.ID); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("B não pode ler o plano de A: %v", err)
	}
	if err := repo.DeletePlanoContas(b.Ctx, filha.ID); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("B não pode excluir o plano de A: %v", err)
	}
	if _, err := repo.UpdateCentroCusto(b.Ctx, &entity.CentroCusto{ID: centro.ID, Codigo: "x", Descricao: "x", Tipo: "x"}); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("B não pode alterar o centro de A: %v", err)
	}

	// Alterar e listar com inativos.
	filha.Descricao, filha.IsActive = "Energia elétrica", false
	if got, err := repo.UpdatePlanoContas(a.Ctx, filha); err != nil || got.Descricao != "Energia elétrica" || got.IsActive {
		t.Fatalf("alterando: %+v %v", got, err)
	}
	todos, err := repo.ListPlanoContasComInativos(a.Ctx)
	if err != nil || len(todos) != 2 {
		t.Fatalf("a listagem do cadastro traz ativos e inativos de A (2): %d %v", len(todos), err)
	}
	ativos, _ := repo.ListPlanoContas(a.Ctx)
	if len(ativos) != 1 {
		t.Fatalf("a listagem de escolha continua só com ativos (1), veio %d", len(ativos))
	}

	// Excluir sem uso.
	testutil.Exec(t, pool, `DELETE FROM contas_pagar WHERE enterprise_id = $1`, a.ID)
	if err := repo.DeletePlanoContas(a.Ctx, filha.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteCentroCusto(a.Ctx, centro.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetCentroCusto(a.Ctx, centro.ID); !errors.As(err, new(*errorsuc.NotFoundError)) {
		t.Fatalf("centro excluído ainda aparece: %v", err)
	}
	_ = bg
}
