//go:build integration

package acesso_test

import (
	"context"
	"testing"

	dom "github.com/FelipePn10/panossoerp/internal/domain/acesso"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/acesso"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Perfis são da empresa: B não vê nem usa o perfil de A, e o banco recusa ligar
// o usuário de uma empresa ao perfil da outra. Vale a união dos perfis ATIVOS.
func TestPerfisDeAcessoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM acesso_perfis WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	for _, e := range []testutil.Empresa{a, b} {
		if _, err := pool.Exec(base, `INSERT INTO user_enterprises(user_id, enterprise_id, role) VALUES ($1, $2, 'USER') ON CONFLICT DO NOTHING`, actor, e.ID); err != nil {
			t.Fatal(err)
		}
	}
	repo := acesso.New(pool)

	vendas, err := repo.CriarPerfil(a.Ctx, &dom.Perfil{Nome: "Vendas", Ativo: true,
		Permissoes: dom.Permissoes{dom.Comercial: {dom.Ler: true, dom.IncluirAlterar: true}}}, actor)
	if err != nil {
		t.Fatal(err)
	}
	fiscal, err := repo.CriarPerfil(a.Ctx, &dom.Perfil{Nome: "Fiscal", Ativo: true,
		Permissoes: dom.Permissoes{dom.Fiscal: {dom.Ler: true, dom.Autorizar: true}}}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CriarPerfil(a.Ctx, &dom.Perfil{Nome: "vendas", Ativo: true}, actor); err == nil {
		t.Fatal("nome repetido (sem diferenciar maiúscula) deveria ser recusado")
	}
	if lista, _ := repo.ListarPerfis(b.Ctx); len(lista) != 0 {
		t.Fatalf("B viu os perfis de A: %+v", lista)
	}
	if err := repo.DefinirPerfisDoUsuario(b.Ctx, actor, []int64{vendas.ID}, actor); err == nil {
		t.Fatal("B ligou o usuário ao perfil de A")
	}

	if err := repo.DefinirPerfisDoUsuario(a.Ctx, actor, []int64{vendas.ID, fiscal.ID}, actor); err != nil {
		t.Fatal(err)
	}
	ac, err := repo.AcessoDoUsuario(base, actor, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ac.Restrito || !ac.Pode(dom.Comercial, dom.IncluirAlterar) || !ac.Pode(dom.Fiscal, dom.Autorizar) || ac.Pode(dom.Financeiro, dom.Ler) {
		t.Fatalf("união dos perfis: %+v", ac)
	}
	if outra, _ := repo.AcessoDoUsuario(base, actor, b.ID); outra.Restrito {
		t.Fatal("na empresa B o usuário não tem perfil: segue só com o papel")
	}

	// Perfil inativo deixa de valer.
	fiscal.Ativo = false
	if err := repo.AtualizarPerfil(a.Ctx, fiscal, actor); err != nil {
		t.Fatal(err)
	}
	ac, _ = repo.AcessoDoUsuario(base, actor, a.ID)
	if ac.Pode(dom.Fiscal, dom.Autorizar) || !ac.Pode(dom.Comercial, dom.Ler) {
		t.Fatalf("perfil inativo não vale: %+v", ac)
	}
	if err := repo.ExcluirPerfil(a.Ctx, vendas.ID); err == nil {
		t.Fatal("perfil em uso não pode ser excluído")
	}
	usuarios, err := repo.ListarUsuarios(a.Ctx)
	if err != nil || len(usuarios) == 0 {
		t.Fatalf("usuários da empresa: %v %v", usuarios, err)
	}
	achou := false
	for _, u := range usuarios {
		if u.UserID == actor && len(u.Perfis) == 2 {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("o usuário deveria aparecer com 2 perfis: %+v", usuarios)
	}
	if err := repo.DefinirPerfisDoUsuario(a.Ctx, actor, nil, actor); err != nil {
		t.Fatal(err)
	}
	if err := repo.ExcluirPerfil(a.Ctx, vendas.ID); err != nil {
		t.Fatalf("sem usuários o perfil pode ser excluído: %v", err)
	}
}
