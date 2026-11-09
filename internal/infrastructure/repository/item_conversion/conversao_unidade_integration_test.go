//go:build integration

package item_conversion_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	uomentity "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/entity"
	itemconv "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	uomrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/unit_of_measure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Catálogo, sinônimos, conversões do item e o grafo — cada empresa só enxerga
// o que é dela, e a conversão de um item vale só na empresa do item.
func TestConversaoDeUnidadeIsolamentoEGrafo(t *testing.T) {
	q, pool := testutil.Queries(t)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	ator := testutil.Actor(t, pool)
	chapa, outro := testutil.UniqueCode(), testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, chapa)
	testutil.NovosItens(t, pool, b.ID, ator, outro)

	// Empresa nova nasce com o catálogo semeado (gatilho da migração 474).
	for _, e := range []testutil.Empresa{a, b} {
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM units_of_measure WHERE enterprise_id=$1 AND code IN ('UN','KG','M2','G')`, e.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 4 {
			t.Fatalf("empresa %d deveria nascer com UN, KG, M2 e G no catálogo; tem %d", e.ID, n)
		}
	}

	repo := itemconv.New(q, pool)
	nova := func(de, para, qDe, qPara string) *entity.ItemUnitConversion {
		c, err := entity.NewItemUnitConversionFraction(chapa, "", de, para, decimal.RequireFromString(qDe), decimal.RequireFromString(qPara), decimal.Zero, decimal.Zero, "VALUE", ator)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	// 1 chapa = 3 M2; 1 M2 = 23,55 KG.
	unM2, err := repo.Create(a.Ctx, nova("UN", "M2", "1", "3"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(a.Ctx, nova("M2", "KG", "1", "23.55")); err != nil {
		t.Fatal(err)
	}

	// Encadeamento: KG → UN sem conversão direta.
	cam, err := uomgraph.Resolver(a.Ctx, pool, a.ID, chapa, "", "", time.Time{}, "KG", "UN")
	if err != nil || cam == nil {
		t.Fatalf("KG→UN deveria encadear KG→M2→UN: %v", err)
	}
	if got := cam.Converter(decimal.RequireFromString("706.5")).StringFixed(6); got != "10.000000" {
		t.Fatalf("706,5 KG = 10 chapas, deu %s", got)
	}
	// Padrão da empresa, mesma dimensão: G → KG.
	if cam, err := uomgraph.Resolver(a.Ctx, pool, a.ID, chapa, "", "", time.Time{}, "G", "KG"); err != nil || cam == nil || cam.OrigemPrincipal() != service.OrigemPadrao {
		t.Fatalf("G→KG deveria vir da conversão padrão: %v %+v", err, cam)
	}
	// A conversão inversa do mesmo fato, vigente junto, é recusada.
	if _, err := repo.Create(a.Ctx, nova("M2", "UN", "3", "1")); err == nil {
		t.Fatal("M2→UN contradiz/duplica UN→M2 vigente: deveria ser recusada")
	}

	// Isolamento: a empresa B não lê nem resolve a conversão do item da A.
	if cam, err := uomgraph.Resolver(b.Ctx, pool, b.ID, chapa, "", "", time.Time{}, "KG", "UN"); err != nil || cam != nil {
		t.Fatalf("empresa B não pode resolver conversão do item da A: %v %+v", err, cam)
	}
	if lista, err := repo.ListByItem(b.Ctx, chapa); err != nil || len(lista) != 0 {
		t.Fatalf("empresa B listou conversões do item da A: %v %d", err, len(lista))
	}
	if c, err := repo.GetByID(b.Ctx, unM2.ID); err == nil && c != nil {
		t.Fatal("empresa B leu a conversão da A pelo id")
	}
	if _, err := itemconv.New(q, pool).Create(b.Ctx, nova("UN", "MM", "1", "6000")); err == nil {
		t.Fatal("empresa B criou conversão para item da A")
	}

	// Sinônimo do catálogo: só na empresa que o cadastrou.
	unidades := uomrepo.New(pool)
	al, err := uomentity.NewAlias("PECA9", "UN", nil)
	if err != nil {
		t.Fatal(err)
	}
	al.CreatedBy = &ator
	if _, err := unidades.CreateAlias(a.Ctx, al); err != nil {
		t.Fatal(err)
	}
	ra, err := uomgraph.Normalizar(a.Ctx, pool, a.ID, nil, []string{"peça9", "Kg."})
	if err != nil {
		t.Fatal(err)
	}
	if ra["peça9"].Unidade != "UN" || ra["Kg."].Unidade != "KG" {
		t.Fatalf("A deveria traduzir peça9→UN e Kg.→KG: %+v", ra)
	}
	rb, err := uomgraph.Normalizar(b.Ctx, pool, b.ID, nil, []string{"peça9"})
	if err != nil {
		t.Fatal(err)
	}
	if rb["peça9"].Unidade != "" {
		t.Fatalf("o sinônimo da A vazou para a B: %+v", rb)
	}
}
