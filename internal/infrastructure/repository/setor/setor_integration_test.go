//go:build integration

package setor_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	dom "github.com/FelipePn10/panossoerp/internal/domain/setor"
	setorrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/setor"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// O grupo do PDM aponta para o setor; o item herda; trocar a pessoa do setor
// muda o responsável sem tocar no grupo nem no item. Outra empresa não vê.
func TestResponsaveisPorGrupoESetor(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			for _, q := range []string{`DELETE FROM pdm_group_responsibles WHERE enterprise_id=$1`, `DELETE FROM enterprise_department_users WHERE enterprise_id=$1`,
				`DELETE FROM enterprise_departments WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM groups WHERE enterprise_id=$1`,
				`DELETE FROM user_enterprises WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`} {
				testutil.Exec(t, pool, q, e)
			}
		}
	})
	ctxA := context.WithValue(a.Ctx, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: a.ID})
	ctxB := context.WithValue(b.Ctx, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: b.ID})
	pessoa1, pessoa2 := uuid.New(), uuid.New()
	for i, u := range []uuid.UUID{pessoa1, pessoa2} {
		testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password,role,is_active,auth_version) VALUES($1,$2,$3,'x','USER',TRUE,1)`, u, fmt.Sprintf("Comprador %d", i+1), u.String()+"@t.local")
		testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER')`, u, a.ID)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1)`, []uuid.UUID{pessoa1, pessoa2})
	})
	testutil.Exec(t, pool, `INSERT INTO groups(code,description,enterprise_id,created_by,is_active) VALUES(10,'CHAPAS',$1,$2,TRUE)`, a.ID, ator)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	item := testutil.UniqueCode()
	codigo := fmt.Sprintf("CH-%d", item)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,pdm_group_code) VALUES($1,$2,$3,$4,$5,10)`, item, codigo, almox, ator, a.ID)

	repo := setorrepo.New(pool)
	for _, s := range []dom.Setor{{Codigo: "COMPRAS", Nome: "Compras de aço", Ativo: true}, {Codigo: "PCP", Nome: "Planejamento", Ativo: true}} {
		if err := repo.SalvarSetor(ctxA, s); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DefinirMembros(ctxA, "COMPRAS", []uuid.UUID{pessoa1}); err != nil {
		t.Fatal(err)
	}
	p, c := "PCP", "COMPRAS"
	if err := repo.SalvarResponsaveis(ctxA, dom.Responsaveis{Grupo: 10, Planejamento: &p, Compras: &c}, &ator); err != nil {
		t.Fatal(err)
	}
	r, err := repo.ResponsaveisDoItem(ctxA, codigo)
	if err != nil {
		t.Fatal(err)
	}
	if r.ComprasNome != "Compras de aço" || r.PlanejamentoNome != "Planejamento" || strings.Join(r.PessoasCompras, ",") != "Comprador 1" {
		t.Fatalf("o item herda do grupo: %+v", r)
	}
	// Comprador 1 saiu: troca no setor, e o item já mostra o novo.
	if err := repo.DefinirMembros(ctxA, "COMPRAS", []uuid.UUID{pessoa2}); err != nil {
		t.Fatal(err)
	}
	if r, _ = repo.ResponsaveisDoItem(ctxA, codigo); strings.Join(r.PessoasCompras, ",") != "Comprador 2" {
		t.Fatalf("troca da pessoa no setor: %+v", r)
	}
	// Setor desativado não pode ser escolhido; pessoa de outra empresa não entra.
	if err := repo.SalvarSetor(ctxA, dom.Setor{Codigo: "PCP", Nome: "Planejamento", Ativo: false}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SalvarResponsaveis(ctxA, dom.Responsaveis{Grupo: 10, Planejamento: &p}, &ator); err == nil || !strings.Contains(err.Error(), "desativado") {
		t.Fatalf("setor desativado: %v", err)
	}
	if err := repo.DefinirMembros(ctxB, "COMPRAS", []uuid.UUID{pessoa1}); err == nil {
		t.Fatal("outra empresa não tem o setor COMPRAS")
	}
	if s, err := repo.Setores(ctxB); err != nil || len(s) != 0 {
		t.Fatalf("outra empresa viu os setores: %v %v", s, err)
	}
}
