//go:build integration

package structure_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O recálculo lê, calcula e só depois grava. Se a linha ou a unidade de
// estoque do componente mudam no meio, gravar poria o número velho por cima
// do novo: a trava pula a linha. A linha antiga com as colunas nulas continua
// sendo corrigida, e a empresa B não grava na linha da A.
func TestTravaDoFatorDaEstrutura(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	pai := int64(1_790_000_000 + testutil.UniqueCode()%10_000_000)
	tubo, pai2, tubo2, pai3 := pai+1, pai+2, pai+3, pai+4
	codigos := []int64{pai, tubo, pai2, tubo2, pai3}
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM item_structure_history WHERE parent_code = ANY($1)`, codigos)
		_, _ = pool.Exec(base, `DELETE FROM item_structures WHERE parent_code = ANY($1)`, codigos)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code = ANY($1)`, codigos)
	})
	actor := testutil.Actor(t, pool)
	for _, c := range codigos {
		testutil.SeedItem(t, pool, a.Ctx, c, actor)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(base, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE items SET warehouse_unit_of_measurement = 'M' WHERE code = ANY($1)`, []int64{tubo, tubo2})
	nova := func(p, filho int64, qtdEstoque, fator any) {
		exec(`INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,quantity_stock_uom,conversion_factor,sequence,created_by)
			VALUES($1,$2,104,'MM',$3,$4,10,$5)`, p, filho, qtdEstoque, fator, actor)
	}
	nova(pai, tubo, 104, 1)
	nova(pai2, tubo2, 104, 1)
	nova(pai3, tubo, nil, nil) // linha antiga, de antes das colunas
	repo := structure.NewFatoresDaEstrutura(pool)
	ler := func(p int64) entity.LinhaDeFator {
		t.Helper()
		l, err := repo.LinhasParaConferir(a.Ctx, entity.FiltroDeFatores{Pai: &p})
		if err != nil || len(l) != 1 {
			t.Fatalf("linhas de %d: %d, %v", p, len(l), err)
		}
		return l[0]
	}
	gravado := func(p int64) string {
		t.Helper()
		var s string
		if err := pool.QueryRow(base, `SELECT quantity::numeric(20,4)||'|'||COALESCE(quantity_stock_uom::numeric(20,4)::text,'nulo')
			FROM item_structures WHERE parent_code=$1 AND is_active`, p).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	certo := decimal.RequireFromString("0.104")
	milesimo := decimal.RequireFromString("0.001")

	// 1. O usuário trocou 104 MM por 50 MM depois da leitura.
	lida := ler(pai)
	exec(`UPDATE item_structures SET quantity = 50, quantity_stock_uom = 0.05, conversion_factor = 0.001 WHERE parent_code = $1`, pai)
	if gravou, err := repo.GravarFator(a.Ctx, lida, certo, milesimo, "teste"); err != nil || gravou {
		t.Fatalf("gravou por cima da alteração do usuário: gravou=%v err=%v", gravou, err)
	}
	if g := gravado(pai); g != "50.0000|0.0500" {
		t.Fatalf("linha alterada pelo usuário ficou %s", g)
	}

	// 2. A unidade de estoque do componente mudou depois da leitura.
	lida = ler(pai2)
	exec(`UPDATE items SET warehouse_unit_of_measurement = 'CM' WHERE code = $1`, tubo2)
	if gravou, err := repo.GravarFator(a.Ctx, lida, certo, milesimo, "teste"); err != nil || gravou {
		t.Fatalf("gravou o fator da unidade antiga: gravou=%v err=%v", gravou, err)
	}

	// 3. Linha antiga com as colunas nulas: corrigida pelo recálculo.
	if gravou, err := repo.GravarFator(b.Ctx, ler(pai3), certo, milesimo, "invasão"); err != nil || gravou {
		t.Fatalf("empresa B gravou na linha da A: gravou=%v err=%v", gravou, err)
	}
	uc := &structure_uc.RecalcularFatores{Repo: repo, Conv: conversorDeTubo{}}
	res, _, err := uc.Executar(a.Ctx, entity.FiltroDeFatores{Pai: &pai3}, true, "teste")
	if err != nil || res.Atualizadas != 1 {
		t.Fatalf("linha nula: atualizadas=%d err=%v", res.Atualizadas, err)
	}
	if g := gravado(pai3); g != "104.0000|0.1040" {
		t.Fatalf("linha nula ficou %s", g)
	}
}
