//go:build integration

package fiscal_test

import (
	"context"

	"github.com/shopspring/decimal"

	"testing"

	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O fator da nota de entrada vem do cadastro do item (direta ou inversa) e,
// sem ele, da conversão global da empresa — em NUMERIC, sem arredondar a inversa.
func TestFatorDeConversaoDoCadastro(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	testutil.Exec(t, pool, `INSERT INTO item_unit_conversions(item_code,from_uom,to_uom,factor,created_by) VALUES($1,'KG','M',0.1553096500,$2)`, item, ator)
	testutil.Exec(t, pool, `INSERT INTO global_unit_conversions(enterprise_id,from_uom,to_uom,factor,created_by) VALUES($1,'KG','G',1000,$2)`, emp, ator)
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM item_unit_conversions WHERE item_code=$1`, item)
		testutil.Exec(t, pool, `DELETE FROM global_unit_conversions WHERE enterprise_id=$1`, emp)
	})
	r := fiscalpg.NewFiscalRepositoryPG(pool).(interface {
		FatorDeConversao(context.Context, int64, string, string) (*decimal.Decimal, error)
	})

	casos := []struct{ de, para, quer string }{
		{"kg", "m", "0.15530965"},   // direta (caixa e espaços não importam)
		{"M", "KG", "6.4387499425"}, // inversa: 1/0,15530965 com 10 casas
		{"G", "KG", "0.001"},        // global inversa
		{"KG", "G", "1000"},         // global direta
	}
	for _, c := range casos {
		f, err := r.FatorDeConversao(ctx, item, c.de, c.para)
		if err != nil || f == nil {
			t.Fatalf("%s→%s: %v %v", c.de, c.para, f, err)
		}
		if f.String() != c.quer {
			t.Fatalf("%s→%s = %s, quer %s", c.de, c.para, f.String(), c.quer)
		}
	}
	if f, err := r.FatorDeConversao(ctx, item, "KG", "UN"); err != nil || f != nil {
		t.Fatalf("sem conversão cadastrada deveria ser nil: %v %v", f, err)
	}
}
