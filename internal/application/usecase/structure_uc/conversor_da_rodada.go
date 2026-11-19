package structure_uc

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// caminhoExato é opcional no conversor (o de produção o tem): fator e
// conversão exata numa consulta só.
type caminhoExato interface {
	CaminhoExato(ctx context.Context, itemCode int64, fromUOM, toUOM string) (decimal.Decimal, func(decimal.Decimal) decimal.Decimal, bool, error)
}

type chaveDeConversao struct {
	item     int64
	de, para string
}

type conversaoGuardada struct {
	fator     decimal.Decimal
	converter func(decimal.Decimal) decimal.Decimal
	achou     bool
}

// conversorDaRodada guarda, durante UMA rodada de recálculo, a conversão de
// cada (item, unidade da linha, unidade de estoque). Sem ele, cada linha
// remontava o grafo de conversões duas vezes — 2.000 linhas de tubo levavam
// ~7 s antes de cada cálculo do MRP. Vale só para a rodada: a próxima lê o
// cadastro de novo, então não há conversão velha guardada. Erro de leitura
// não é guardado.
type conversorDaRodada struct {
	ports.UOMConverter
	guardado map[chaveDeConversao]conversaoGuardada
}

func novoConversorDaRodada(conv ports.UOMConverter) ports.UOMConverter {
	if conv == nil {
		return nil
	}
	if _, ok := conv.(caminhoExato); !ok {
		return conv
	}
	return &conversorDaRodada{UOMConverter: conv, guardado: map[chaveDeConversao]conversaoGuardada{}}
}

func (c *conversorDaRodada) caminho(ctx context.Context, item int64, de, para string) (conversaoGuardada, error) {
	k := chaveDeConversao{item, strings.ToUpper(strings.TrimSpace(de)), strings.ToUpper(strings.TrimSpace(para))}
	if g, ok := c.guardado[k]; ok {
		return g, nil
	}
	f, conv, achou, err := c.UOMConverter.(caminhoExato).CaminhoExato(ctx, item, de, para)
	if err != nil {
		return conversaoGuardada{}, err
	}
	g := conversaoGuardada{fator: f, converter: conv, achou: achou && conv != nil}
	c.guardado[k] = g
	return g, nil
}

func (c *conversorDaRodada) Factor(ctx context.Context, item int64, de, para string) (decimal.Decimal, bool, error) {
	g, err := c.caminho(ctx, item, de, para)
	if err != nil || !g.achou {
		return decimal.Zero, false, err
	}
	return g.fator, true, nil
}

func (c *conversorDaRodada) ConvertQuantityExact(ctx context.Context, item int64, q decimal.Decimal, de, para string) (decimal.Decimal, bool, error) {
	g, err := c.caminho(ctx, item, de, para)
	if err != nil || !g.achou {
		return decimal.Zero, false, err
	}
	return g.converter(q), true, nil
}
