//go:build integration

package structure_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// conversorDeTubo: 1 MM = 0,001 M. Não há conversão MM → KG.
type conversorDeTubo struct{}

func (conversorDeTubo) Factor(_ context.Context, _ int64, de, para string) (decimal.Decimal, bool, error) {
	if de == "MM" && para == "M" {
		return decimal.RequireFromString("0.001"), true, nil
	}
	return decimal.Zero, false, nil
}
func (c conversorDeTubo) ConvertQuantity(ctx context.Context, item int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	f, ok, err := c.Factor(ctx, item, de, para)
	return q.Mul(f), ok, err
}
func (c conversorDeTubo) ConvertUnitPrice(ctx context.Context, item int64, p decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	f, ok, err := c.Factor(ctx, item, de, para)
	if !ok || err != nil {
		return decimal.Zero, ok, err
	}
	return p.Div(f), true, nil
}

// TestFatorDaEstruturaAcompanhaAUnidadeDeEstoque reproduz a Tecnofer: o tubo
// passou a ser estocado em metro e as linhas guardaram o número velho — "104
// MM, fator 1" (o MRP comprava 4.160 m em vez de 4,16 m) e "104 MM, fator de
// barra 1/6000" (6 vezes menos). O recálculo regrava as duas, aponta a linha
// sem conversão e não toca a estrutura de outra empresa.
func TestFatorDaEstruturaAcompanhaAUnidadeDeEstoque(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	pai := int64(1_770_000_000 + testutil.UniqueCode()%10_000_000)
	tubo1, tubo2, chapa, paiB, tuboB := pai+1, pai+2, pai+3, pai+4, pai+5
	codigos := []int64{pai, tubo1, tubo2, chapa, paiB, tuboB}
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM item_structure_history WHERE parent_code = ANY($1)`, codigos)
		_, _ = pool.Exec(base, `DELETE FROM item_structures WHERE parent_code = ANY($1)`, codigos)
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code = ANY($1)`, codigos)
	})
	actor := testutil.Actor(t, pool)
	for _, c := range []int64{pai, tubo1, tubo2, chapa} {
		testutil.SeedItem(t, pool, a.Ctx, c, actor)
	}
	testutil.SeedItem(t, pool, b.Ctx, paiB, actor)
	testutil.SeedItem(t, pool, b.Ctx, tuboB, actor)
	if _, err := pool.Exec(base, `UPDATE items SET warehouse_unit_of_measurement = 'M' WHERE code = ANY($1)`, []int64{tubo1, tubo2, tuboB}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(base, `UPDATE items SET warehouse_unit_of_measurement = 'KG' WHERE code = $1`, chapa); err != nil {
		t.Fatal(err)
	}

	repo := structure.NewItemStructureRepository(sqlc.New(pool)).WithHistory(pool)
	linha := func(ctx context.Context, p, filho int64, seq int, qtdGravada, fator string) {
		t.Helper()
		if _, err := repo.Create(ctx, &entity.ItemStructure{ParentCode: p, ChildCode: filho,
			Quantity: decimal.NewFromInt(104), UnitOfMeasurement: "MM", Health: "ATIVO", Sequence: seq, CreatedBy: actor}); err != nil {
			t.Fatalf("linha %d→%d: %v", p, filho, err)
		}
		// O que ficou gravado antes de a unidade de estoque mudar.
		if _, err := pool.Exec(base, `UPDATE item_structures SET quantity_stock_uom = $3, conversion_factor = $4
			WHERE parent_code = $1 AND child_code = $2 AND is_active`, p, filho, qtdGravada, fator); err != nil {
			t.Fatal(err)
		}
	}
	linha(a.Ctx, pai, tubo1, 10, "104", "1")
	linha(a.Ctx, pai, tubo2, 20, "0.0173333333", "0.0001666667")
	linha(a.Ctx, pai, chapa, 30, "104", "1")
	// A empresa B tem o mesmo problema na estrutura dela.
	linha(b.Ctx, paiB, tuboB, 10, "104", "1")

	gravado := func(p, filho int64) (decimal.Decimal, decimal.Decimal) {
		t.Helper()
		var q, f decimal.Decimal
		if err := pool.QueryRow(base, `SELECT quantity_stock_uom, conversion_factor FROM item_structures
			WHERE parent_code = $1 AND child_code = $2 AND is_active`, p, filho).Scan(&q, &f); err != nil {
			t.Fatal(err)
		}
		return q, f
	}

	uc := &structure_uc.RecalcularFatores{Repo: structure.NewFatoresDaEstrutura(pool), Conv: conversorDeTubo{}}

	// Só conferir não grava nada.
	res, defasadas, err := uc.Executar(a.Ctx, entity.FiltroDeFatores{Pai: &pai}, false, "conferência")
	if err != nil {
		t.Fatal(err)
	}
	if len(defasadas) != 2 || res.Atualizadas != 0 || len(res.SemConversao) != 1 {
		t.Fatalf("conferência: defasadas=%d atualizadas=%d sem conversão=%d", len(defasadas), res.Atualizadas, len(res.SemConversao))
	}
	if q, _ := gravado(pai, tubo1); !q.Equal(decimal.NewFromInt(104)) {
		t.Fatalf("a conferência gravou: %s", q)
	}

	// Empresa B recalcula as dela: a estrutura da A fica como está.
	if _, err := uc.DaEmpresa(b.Ctx, "teste"); err != nil {
		t.Fatal(err)
	}
	if q, _ := gravado(pai, tubo1); !q.Equal(decimal.NewFromInt(104)) {
		t.Fatalf("o recálculo da empresa B mexeu na estrutura da A: %s", q)
	}
	if q, _ := gravado(paiB, tuboB); !q.Equal(decimal.RequireFromString("0.104")) {
		t.Fatalf("a linha da própria empresa B não foi recalculada: %s", q)
	}
	// Nem gravando direto pelo id da linha da A.
	linhasA, err := structure.NewFatoresDaEstrutura(pool).LinhasParaConferir(a.Ctx, entity.FiltroDeFatores{Pai: &pai})
	if err != nil || len(linhasA) != 3 {
		t.Fatalf("linhas da A: %d, %v", len(linhasA), err)
	}
	if gravou, err := structure.NewFatoresDaEstrutura(pool).GravarFator(b.Ctx, linhasA[0], decimal.NewFromInt(999), decimal.NewFromInt(9), "invasão"); err != nil || gravou {
		t.Fatalf("gravou=%v err=%v", gravou, err)
	}
	if q, _ := gravado(pai, tubo1); !q.Equal(decimal.NewFromInt(104)) {
		t.Fatalf("empresa B regravou a linha da A pelo id: %s", q)
	}
	if outras, _ := structure.NewFatoresDaEstrutura(pool).LinhasParaConferir(b.Ctx, entity.FiltroDeFatores{Pai: &pai}); len(outras) != 0 {
		t.Fatalf("empresa B leu %d linha(s) da estrutura da A", len(outras))
	}

	// O tubo mudou de unidade: recalcula as linhas em que é componente.
	n, avisos, err := uc.RecalcularDoComponente(a.Ctx, tubo1, "unidade de estoque alterada")
	if err != nil || n != 1 || len(avisos) != 0 {
		t.Fatalf("componente: n=%d avisos=%v err=%v", n, avisos, err)
	}
	q, f := gravado(pai, tubo1)
	if !q.Equal(decimal.RequireFromString("0.104")) || !f.Equal(decimal.RequireFromString("0.001")) {
		t.Fatalf("tubo 1: %s / fator %s, esperado 0,104 M / 0,001", q, f)
	}

	// A empresa toda (o que o MRP faz antes de calcular).
	n, avisos, err = uc.RecalcularDaEmpresa(a.Ctx, "MRP")
	if err != nil || n != 1 {
		t.Fatalf("empresa: n=%d err=%v", n, err)
	}
	if q, _ := gravado(pai, tubo2); !q.Equal(decimal.RequireFromString("0.104")) {
		t.Fatalf("tubo 2 (fator de barra velho): %s, esperado 0,104 M", q)
	}
	if len(avisos) != 1 {
		t.Fatalf("a linha sem conversão MM→KG tinha de virar aviso: %v", avisos)
	}
	// Rodar de novo não muda nada (nem grava histórico à toa).
	if n, _, _ := uc.RecalcularDaEmpresa(a.Ctx, "MRP"); n != 0 {
		t.Fatalf("segundo recálculo regravou %d linha(s)", n)
	}
	var hist int
	if err := pool.QueryRow(base, `SELECT count(*) FROM item_structure_history WHERE parent_code = $1
		AND after_state ? 'motivo'`, pai).Scan(&hist); err != nil {
		t.Fatal(err)
	}
	if hist != 2 {
		t.Fatalf("histórico do recálculo: %d linha(s), esperado 2", hist)
	}
}
