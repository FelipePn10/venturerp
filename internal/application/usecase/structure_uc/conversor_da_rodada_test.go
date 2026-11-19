package structure_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

// conversorContado conta as consultas ao cadastro: 1 MM = 0,001 M; de 1/3 em
// "TERCO" para a fração exata (Factor arredondado ≠ conversão exata).
type conversorContado struct {
	consultas int
	falhar    bool
}

func (c *conversorContado) CaminhoExato(_ context.Context, _ int64, de, para string) (decimal.Decimal, func(decimal.Decimal) decimal.Decimal, bool, error) {
	c.consultas++
	if c.falhar {
		return decimal.Zero, nil, false, errors.New("banco fora")
	}
	switch {
	case de == "MM" && para == "M":
		return decimal.RequireFromString("0.001"), func(q decimal.Decimal) decimal.Decimal { return q.Div(decimal.NewFromInt(1000)) }, true, nil
	case de == "TERCO" && para == "UN":
		return decimal.RequireFromString("0.3333333333"), func(q decimal.Decimal) decimal.Decimal { return q.Div(decimal.NewFromInt(3)) }, true, nil
	}
	return decimal.Zero, nil, false, nil
}
func (c *conversorContado) Factor(ctx context.Context, i int64, de, para string) (decimal.Decimal, bool, error) {
	f, _, ok, err := c.CaminhoExato(ctx, i, de, para)
	return f, ok, err
}
func (c *conversorContado) ConvertQuantity(ctx context.Context, i int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	_, conv, ok, err := c.CaminhoExato(ctx, i, de, para)
	if !ok || err != nil {
		return decimal.Zero, ok, err
	}
	return conv(q), true, nil
}
func (c *conversorContado) ConvertUnitPrice(context.Context, int64, decimal.Decimal, string, string) (decimal.Decimal, bool, error) {
	return decimal.Zero, false, nil
}
func (c *conversorContado) ConvertQuantityExact(ctx context.Context, i int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	return c.ConvertQuantity(ctx, i, q, de, para)
}

type linhasFixas struct {
	linhas  []entity.LinhaDeFator
	gravado map[int64]decimal.Decimal
}

func (r *linhasFixas) LinhasParaConferir(context.Context, entity.FiltroDeFatores) ([]entity.LinhaDeFator, error) {
	return r.linhas, nil
}
func (r *linhasFixas) GravarFator(_ context.Context, l entity.LinhaDeFator, qtd, _ decimal.Decimal, _ string) (bool, error) {
	r.gravado[l.ID] = qtd
	return true, nil
}

// Muitas linhas do mesmo componente: uma consulta por (item, unidade da linha,
// unidade de estoque) na rodada — antes eram duas por linha. A quantidade sai
// da fração exata, igual sem o cache.
func TestRecalculoConsultaCadaConversaoUmaVezPorRodada(t *testing.T) {
	var linhas []entity.LinhaDeFator
	for i := int64(1); i <= 50; i++ {
		linhas = append(linhas, entity.LinhaDeFator{ID: i, ParentCode: 100 + i, ChildCode: 7, Quantidade: decimal.NewFromInt(104),
			Unidade: "mm", UnidadeEstoque: "M", QtdGravada: decimal.NewFromInt(104), FatorGravado: decimal.NewFromInt(1)})
	}
	linhas = append(linhas, entity.LinhaDeFator{ID: 99, ParentCode: 999, ChildCode: 8, Quantidade: decimal.NewFromInt(1),
		Unidade: "TERCO", UnidadeEstoque: "UN", QtdGravada: decimal.NewFromInt(1), FatorGravado: decimal.NewFromInt(1)})
	conv := &conversorContado{}
	repo := &linhasFixas{linhas: linhas, gravado: map[int64]decimal.Decimal{}}
	res, _, err := (&RecalcularFatores{Repo: repo, Conv: conv}).Executar(context.Background(), entity.FiltroDeFatores{}, true, "teste")
	if err != nil || res.Atualizadas != 51 {
		t.Fatalf("err=%v atualizadas=%d", err, res.Atualizadas)
	}
	if conv.consultas != 2 {
		t.Fatalf("consultas ao cadastro: %d, esperado 2 (uma por item+unidades)", conv.consultas)
	}
	if !repo.gravado[1].Equal(decimal.RequireFromString("0.104")) {
		t.Fatalf("linha 1: %s", repo.gravado[1])
	}
	if repo.gravado[99].String() != decimal.NewFromInt(1).Div(decimal.NewFromInt(3)).String() {
		t.Fatalf("a quantidade saiu do fator arredondado, não da fração exata: %s", repo.gravado[99])
	}
}

// Falha de leitura não fica guardada: interrompe a rodada com o erro.
func TestRecalculoNaoGuardaErroDeLeitura(t *testing.T) {
	conv := &conversorContado{falhar: true}
	repo := &linhasFixas{linhas: []entity.LinhaDeFator{{ID: 1, ChildCode: 7, Quantidade: decimal.NewFromInt(1), Unidade: "MM", UnidadeEstoque: "M"}},
		gravado: map[int64]decimal.Decimal{}}
	if _, _, err := (&RecalcularFatores{Repo: repo, Conv: conv}).Executar(context.Background(), entity.FiltroDeFatores{}, true, "teste"); err == nil {
		t.Fatal("erro de leitura do cadastro virou 'sem conversão' ou foi engolido")
	}
	rodada := novoConversorDaRodada(conv).(*conversorDaRodada)
	_, _, _ = rodada.Factor(context.Background(), 7, "MM", "M")
	if len(rodada.guardado) != 0 {
		t.Fatal("guardou o erro de leitura")
	}
}
