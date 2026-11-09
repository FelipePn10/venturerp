//go:build integration

package item_conversion_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	itemconv "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/itemgovernanca"
	uomrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/unit_of_measure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A segunda unidade do cadastro do item (VENT0200) é uma conversão: compra,
// nota e saldo a enxergam pelo mesmo grafo. A conversão explícita da VSUP0110
// do mesmo par ganha dela. O saldo em segunda unidade usa o peso que o
// movimento gravou (o gatilho congela o nominal) e, para movimento de antes de
// o item ter segunda unidade, o fator do grafo.
func TestSegundaUnidadeDoCadastroEntraNoGrafoENoSaldo(t *testing.T) {
	q, pool := testutil.Queries(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, _ testutil.Empresa) {
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_movements WHERE enterprise_id=$1`, a.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_balances WHERE enterprise_id=$1`, a.ID)
	})
	ator := testutil.Actor(t, pool)
	chapa := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, chapa)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM item_unit_conversion_events WHERE item_code=$1`, chapa)
		_, _ = pool.Exec(context.Background(), `DELETE FROM item_unit_conversions WHERE item_code=$1`, chapa)
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	mov := func(qtd string, peso any) {
		testutil.Exec(t, pool, `INSERT INTO stock_movements(item_code,warehouse_id,movement_type,quantity,created_by,enterprise_id,secondary_quantity) VALUES($1,$2,'IN',$3::numeric,$4,$5,$6::numeric)`,
			chapa, almox, qtd, ator, a.ID, peso)
	}
	// Entrada de antes de o item ter segunda unidade: o gatilho não tem fator e grava nulo.
	mov("4", nil)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='UN', warehouse_secondary_uom='KG', warehouse_secondary_factor=25 WHERE code=$1`, chapa)

	// 1) Só com o cadastro do item: KG → UN pela segunda unidade.
	cam, err := uomgraph.Resolver(a.Ctx, pool, a.ID, chapa, "", "", time.Time{}, "KG", "UN")
	if err != nil || cam == nil {
		t.Fatalf("KG→UN deveria sair da segunda unidade do cadastro: %v", err)
	}
	if cam.OrigemPrincipal() != service.OrigemSegundaUnidade {
		t.Fatalf("origem esperada SEGUNDA_UNIDADE, veio %s", cam.OrigemPrincipal())
	}
	if got := cam.Converter(decimal.RequireFromString("250")).StringFixed(4); got != "10.0000" {
		t.Fatalf("250 KG = 10 UN pela segunda unidade (25), deu %s", got)
	}
	// A outra empresa não enxerga o cadastro deste item.
	if cam, err := uomgraph.Resolver(b.Ctx, pool, b.ID, chapa, "", "", time.Time{}, "KG", "UN"); err != nil || cam != nil {
		t.Fatalf("empresa B não pode resolver pela segunda unidade do item da A: %v %+v", err, cam)
	}

	// 2) Saldo: 4 UN antigas pelo fator (100) + 2 UN pesadas (49) + 1 UN
	// nominal congelada pelo gatilho (25) = 174 KG.
	mov("2", "49")
	mov("1", nil)
	gov := itemgovernanca.New(pool)
	saldo, err := gov.SaldoDuasUnidades(a.Ctx, chapa)
	if err != nil {
		t.Fatal(err)
	}
	if !decimal.RequireFromString(saldo.SegundoSaldo).Equal(decimal.NewFromInt(174)) {
		t.Fatalf("segundo saldo = 4×25 + 49 + 25 = 174 KG, deu %s", saldo.SegundoSaldo)
	}
	if len(saldo.Detalhes) != 1 || !decimal.RequireFromString(saldo.Detalhes[0].SegundoSaldo).Equal(decimal.NewFromInt(174)) {
		t.Fatalf("o detalhe por almoxarifado também soma 174 KG: %+v", saldo.Detalhes)
	}

	// 3) A VSUP0110 cadastra 1 UN = 24 KG: ela ganha da segunda unidade.
	cat := uomrepo.NewCatalogo(pool)
	if f, err := cat.FatorDaConversaoDoItem(a.Ctx, chapa, "UN", "KG"); err != nil || f != nil {
		t.Fatalf("sem conversão na VSUP0110 o fator explícito é nil (a segunda unidade não conta): %v %v", err, f)
	}
	repo := itemconv.New(q, pool)
	c, err := entity.NewItemUnitConversionFraction(chapa, "", "UN", "KG", decimal.NewFromInt(1), decimal.NewFromInt(24), decimal.Zero, decimal.Zero, "VALUE", ator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(a.Ctx, c); err != nil {
		t.Fatal(err)
	}
	cam, err = uomgraph.Resolver(a.Ctx, pool, a.ID, chapa, "", "", time.Time{}, "UN", "KG")
	if err != nil || cam == nil || cam.OrigemPrincipal() != service.OrigemItem || !cam.Fator().Equal(decimal.NewFromInt(24)) {
		t.Fatalf("a conversão explícita (24) ganha da segunda unidade (25): %v %+v", err, cam)
	}
	if f, err := cat.FatorDaConversaoDoItem(a.Ctx, chapa, "KG", "UN"); err != nil || f == nil || f.Round(6).String() != "0.041667" {
		t.Fatalf("KG→UN pela VSUP0110 = 1/24: %v %v", err, f)
	}
	if f, err := cat.FatorDaConversaoDoItem(b.Ctx, chapa, "UN", "KG"); err != nil || f != nil {
		t.Fatalf("empresa B não lê a conversão do item da A: %v %v", err, f)
	}
	saldo, err = gov.SaldoDuasUnidades(a.Ctx, chapa)
	if err != nil {
		t.Fatal(err)
	}
	// O movimento antigo (sem peso) passa a usar 24; os congelados ficam.
	if !decimal.RequireFromString(saldo.SegundoSaldo).Equal(decimal.NewFromInt(170)) {
		t.Fatalf("com a VSUP0110 em 24: 4×24 + 49 + 25 = 170 KG, deu %s", saldo.SegundoSaldo)
	}

	// 4) Sincronização: o cadastro do item passa a 24 e o próximo movimento congela 24.
	sinc, ok := repo.(interface {
		SincronizarSegundaUnidade(ctx context.Context, itemCode int64, de, para string, fator decimal.Decimal) (*decimal.Decimal, bool, error)
	})
	if !ok {
		t.Fatal("o repositório de produção precisa sincronizar a segunda unidade (o caso de uso usa por asserção de tipo)")
	}
	antes, mudou, err := sinc.SincronizarSegundaUnidade(a.Ctx, chapa, "KG", "UN", decimal.NewFromInt(1).DivRound(decimal.NewFromInt(24), 10))
	if err != nil || !mudou || antes == nil || !antes.Equal(decimal.NewFromInt(25)) {
		t.Fatalf("a sincronização pelo sentido inverso grava 24 e devolve o 25 anterior: %v %v %v", err, mudou, antes)
	}
	if _, mudou, err := sinc.SincronizarSegundaUnidade(a.Ctx, chapa, "UN", "KG", decimal.NewFromInt(24)); err != nil || mudou {
		t.Fatalf("o mesmo fator de novo não muda nada: %v %v", err, mudou)
	}
	if _, mudou, err := sinc.SincronizarSegundaUnidade(a.Ctx, chapa, "UN", "M2", decimal.NewFromInt(3)); err != nil || mudou {
		t.Fatalf("par que não é estoque ↔ segunda unidade não mexe no cadastro: %v %v", err, mudou)
	}
	if _, mudou, err := sinc.SincronizarSegundaUnidade(b.Ctx, chapa, "UN", "KG", decimal.NewFromInt(30)); err != nil || mudou {
		t.Fatalf("empresa B não altera o item da A: %v %v", err, mudou)
	}
	mov("1", nil)
	if saldo, err = gov.SaldoDuasUnidades(a.Ctx, chapa); err != nil || !decimal.RequireFromString(saldo.SegundoSaldo).Equal(decimal.NewFromInt(194)) {
		t.Fatalf("170 + 1 UN congelada em 24 = 194 KG: %v %+v", err, saldo)
	}
	if _, err := gov.SaldoDuasUnidades(b.Ctx, chapa); err == nil {
		t.Fatal("empresa B não pode ler o saldo do item da A")
	}
}

// A embalagem do cadastro do item (1 CX = 12 UN) converte sem cadastro na
// VSUP0110, aparece na cobertura e fica dentro da empresa do item.
func TestEmbalagemDoCadastroEntraNoGrafoENaCobertura(t *testing.T) {
	q, pool := testutil.Queries(t)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	ator := testutil.Actor(t, pool)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	gtin := fmt.Sprintf("789%010d", item%10_000_000_000)
	testutil.Exec(t, pool, `UPDATE items SET warehouse_unit_of_measurement='UN' WHERE code=$1`, item)
	testutil.Exec(t, pool, `INSERT INTO item_gtins(enterprise_id,item_code,gtin,uom,package_quantity) VALUES($1,$2,$3,'CX',12)`, a.ID, item, gtin)

	cam, err := uomgraph.Resolver(a.Ctx, pool, a.ID, item, "", "", time.Time{}, "CX", "UN")
	if err != nil || cam == nil || cam.OrigemPrincipal() != service.OrigemEmbalagem {
		t.Fatalf("CX→UN deveria sair da embalagem do cadastro: %v %+v", err, cam)
	}
	if got := cam.Converter(decimal.NewFromInt(3)); !got.Equal(decimal.NewFromInt(36)) {
		t.Fatalf("3 CX = 36 UN, deu %s", got)
	}
	if cam, err := uomgraph.Resolver(b.Ctx, pool, b.ID, item, "", "", time.Time{}, "CX", "UN"); err != nil || cam != nil {
		t.Fatalf("empresa B não converte pela embalagem do item da A: %v %+v", err, cam)
	}

	repo := itemconv.New(q, pool)
	u, err := repo.UnitsOfItem(a.Ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	achou := false
	for _, uso := range u.Usos {
		if uso.Onde == "EMBALAGEM" && uso.Unidade == "CX" {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("a cobertura deveria listar a embalagem CX: %+v", u.Usos)
	}
}
