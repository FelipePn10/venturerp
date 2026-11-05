//go:build integration

package stock_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func TestCadernos_DividemPorAlmoxarifadoELimite(t *testing.T) {
	e := novoAmbienteEstoque(t)
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A1", ""))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A2", ""))
	exigir(t, e.moverLote(t, e.item, e.almox, "IN", 10, 5, "A3", ""))
	inv := e.abrirInventario(t, nil)

	c, err := e.repo.GerarCadernos(e.ctx, inv.ID, 2)
	exigir(t, err)
	if len(c) != 2 || c[0].Linhas != 2 || c[1].Linhas != 1 {
		t.Fatalf("%+v", c)
	}
	exigir(t, e.repo.MarcarCadernoImpresso(e.ctx, inv.ID, 1, e.ator))
	exigir(t, e.repo.MarcarCadernoImpresso(e.ctx, inv.ID, 1, e.ator))
	c, err = e.repo.Cadernos(e.ctx, inv.ID)
	exigir(t, err)
	if c[0].Impressoes != 2 || c[0].Situacao() != "IMPRESSO" {
		t.Fatalf("a reimpressão conta: %+v", c[0])
	}
	// Refazer antes de contar é livre; depois da primeira contagem, não.
	c, err = e.repo.GerarCadernos(e.ctx, inv.ID, 1)
	exigir(t, err)
	if len(c) != 3 {
		t.Fatalf("refez em 3: %+v", c)
	}
	e.contar(t, inv.ID, "A1", "", 10)
	if _, err := e.repo.GerarCadernos(e.ctx, inv.ID, 2); !ehValidacao(err) {
		t.Fatalf("contagem iniciada: esperava recusa, veio %v", err)
	}
	c, err = e.repo.Cadernos(e.ctx, inv.ID)
	exigir(t, err)
	if c[0].Situacao() != "DIGITADO" {
		t.Fatalf("%+v", c[0])
	}
	itens, err := e.repo.ListInventoryItems(e.ctx, inv.ID)
	exigir(t, err)
	for _, it := range itens {
		if it.Notebook == nil {
			t.Fatalf("linha sem caderno: %+v", it)
		}
	}
	if err := e.repo.MarcarCadernoImpresso(e.ctx, inv.ID, 99, e.ator); err == nil {
		t.Fatal("caderno que não existe")
	}
}

func TestTolerancia_ValeOMaiorLimiteDosPerfisDoUsuario(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	bg := context.Background()
	perfil := func(nome string) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO acesso_perfis(enterprise_id,nome,created_by,updated_by) VALUES($1,$2,$3,$3) RETURNING id`, e.empresa, nome, e.ator).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id,enterprise_id,role) VALUES($1,$2,'USER') ON CONFLICT DO NOTHING`, e.ator, e.empresa)
	p1, p2 := perfil("Conferente "+t.Name()), perfil("Supervisor "+t.Name())
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM acesso_perfis WHERE id = ANY($1)`, []int64{p1, p2})
		testutil.Exec(t, pool, `DELETE FROM user_enterprises WHERE user_id=$1 AND enterprise_id=$2`, e.ator, e.empresa)
	})
	limite, err := e.repo.LimiteDeAjusteDoUsuario(e.ctx, e.ator)
	exigir(t, err)
	if limite != nil {
		t.Fatalf("sem perfil, sem limite: %+v", limite)
	}
	v := func(s string) *decimal.Decimal { d := dec(s); return &d }
	exigir(t, e.repo.SalvarToleranciaDoPerfil(e.ctx, stockentity.ToleranciaDoPerfil{PerfilID: p1, ValorDocumento: v("100"), ValorLinha: v("10")}, e.ator))
	exigir(t, e.repo.SalvarToleranciaDoPerfil(e.ctx, stockentity.ToleranciaDoPerfil{PerfilID: p2, ValorDocumento: v("500")}, e.ator))
	for _, p := range []int64{p1, p2} {
		testutil.Exec(t, pool, `INSERT INTO acesso_usuario_perfis(enterprise_id,user_id,perfil_id,created_by) VALUES($1,$2,$3,$2)`, e.empresa, e.ator, p)
	}
	limite, err = e.repo.LimiteDeAjusteDoUsuario(e.ctx, e.ator)
	exigir(t, err)
	if limite == nil || !limite.ValorDocumento.Equal(dec("500")) || limite.ValorLinha != nil {
		t.Fatalf("maior documento e linha liberada pelo outro perfil: %+v", limite)
	}
	testutil.Exec(t, pool, `UPDATE acesso_perfis SET ativo=FALSE WHERE id=$1`, p2)
	limite, err = e.repo.LimiteDeAjusteDoUsuario(e.ctx, e.ator)
	exigir(t, err)
	if !limite.ValorDocumento.Equal(dec("100")) || !limite.ValorLinha.Equal(dec("10")) {
		t.Fatalf("perfil inativo não conta: %+v", limite)
	}
	lista, err := e.repo.ToleranciasPorPerfil(e.ctx)
	exigir(t, err)
	if len(lista) != 1 {
		t.Fatalf("lista só os ativos: %+v", lista)
	}
	exigir(t, e.repo.ExcluirToleranciaDoPerfil(e.ctx, p1))
	limite, err = e.repo.LimiteDeAjusteDoUsuario(e.ctx, e.ator)
	exigir(t, err)
	if limite != nil {
		t.Fatalf("%+v", limite)
	}
	if err := e.repo.SalvarToleranciaDoPerfil(e.ctx, stockentity.ToleranciaDoPerfil{PerfilID: 999999999, ValorLinha: v("1")}, e.ator); !ehValidacao(err) {
		t.Fatalf("perfil de fora: %v", err)
	}
}

// Máscara com política calcula só com o saldo dela; o resto segue no item.
func TestReposicao_PoliticaPorMascaraNaoContaDuasVezes(t *testing.T) {
	e := novoAmbienteEstoque(t)
	pool := testutil.Pool(t)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_minimum_stock=5, engineering_type=1 WHERE code=$1`, e.item)
	mover := func(mask string, qtd int64) {
		_, err := e.repo.CreateMovement(e.ctx, &stockentity.StockMovement{ItemCode: e.item, Mask: mask, WarehouseID: e.almox, MovementType: "IN",
			Quantity: decimal.NewFromInt(qtd), UnitPrice: decimal.NewFromInt(1), CreatedBy: e.ator})
		exigir(t, err)
	}
	mover("AZUL", 3)
	mover("VERDE", 50)
	minimo := dec("10")
	_, err := e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{ItemCode: e.item, Mask: "AZUL", WarehouseID: e.almox, Minimo: &minimo,
		Origem: stockentity.OrigemCompra, Ativa: true, CreatedBy: e.ator})
	exigir(t, err)

	sug, err := e.repo.SugestoesDeReposicao(e.ctx)
	exigir(t, err)
	if len(sug) != 1 || sug[0].Mask != "AZUL" || !sug[0].Disponivel.Equal(dec("3")) || !sug[0].Sugerido.Equal(dec("7")) {
		t.Fatalf("só a AZUL pede, com o saldo dela: %+v", sug)
	}
	// A outra máscara do mesmo almoxarifado ganha política própria sem trocar a primeira.
	_, err = e.repo.SalvarPoliticaDeReposicao(e.ctx, stockentity.PoliticaReposicao{ItemCode: e.item, Mask: "VERDE", WarehouseID: e.almox, Minimo: &minimo,
		Origem: stockentity.OrigemCompra, Ativa: true, CreatedBy: e.ator})
	exigir(t, err)
	lista, err := e.repo.ListarPoliticasDeReposicao(e.ctx)
	exigir(t, err)
	if len(lista) != 2 {
		t.Fatalf("%+v", lista)
	}
}
