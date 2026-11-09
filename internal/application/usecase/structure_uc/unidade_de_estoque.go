package structure_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/shopspring/decimal"
)

// resolverUnidadeDeEstoque converte a quantidade escrita pela engenharia para a
// unidade em que o item é ESTOCADO, e devolve também o fator usado.
//
// O problema que isto resolve: a linha da estrutura aceitava qualquer unidade
// sem nenhuma relação com a do item. Dava para cadastrar a chapa em KG no item
// e escrever "2 M2" na estrutura — e daí em diante o MRP reservava 2 KG, a
// ordem consumia 2 KG e o custo rateava sobre 2 KG. O erro não aparecia em
// lugar nenhum; aparecia no estoque, meses depois, como diferença sem
// explicação.
//
// Quando as unidades são iguais não há o que converter e o fator é 1 — que é o
// caso de praticamente toda estrutura já cadastrada. Quando são diferentes, a
// conversão PRECISA estar cadastrada: sem ela o sistema não tem como saber
// quantos quilos são dois metros quadrados daquela chapa, e adivinhar seria
// pior que recusar.
func resolverUnidadeDeEstoque(
	ctx context.Context,
	conv ports.UOMConverter,
	itemCode int64,
	itemLabel string,
	quantidade decimal.Decimal,
	unidadeDaEstrutura string,
	unidadeDeEstoque string,
) (qtdeEmEstoque decimal.Decimal, fator decimal.Decimal, err error) {
	daEstrutura := strings.ToUpper(strings.TrimSpace(unidadeDaEstrutura))
	doEstoque := strings.ToUpper(strings.TrimSpace(unidadeDeEstoque))

	// Sem unidade declarada, ou unidades iguais: a quantidade já está na
	// unidade de estoque.
	if daEstrutura == "" || doEstoque == "" || daEstrutura == doEstoque {
		return quantidade, decimal.NewFromInt(1), nil
	}

	if conv == nil {
		// Defensivo: sem conversor injetado não dá para garantir o número, e
		// gravar uma quantidade cuja unidade ninguém sabe converter é o bug
		// que esta função existe para impedir.
		return decimal.Zero, decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf(
			"o componente %s é estocado em %s e a estrutura está em %s; "+
				"o cadastro de conversões não está disponível nesta operação",
			itemLabel, doEstoque, daEstrutura))
	}

	f, achou, errConv := conv.Factor(ctx, itemCode, daEstrutura, doEstoque)
	if errConv != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("consultando a conversão de unidade do item %d: %w", itemCode, errConv)
	}
	if !achou || !f.IsPositive() {
		// A conversão pode ser cadastrada em qualquer direção — o sistema usa a
		// inversa quando só uma existe. Dizer as duas importa: "quantas UN tem
		// 1 MM" dá 0,000167, um número que ninguém tira de um desenho; "1 UN
		// tem 6000 MM" é o que está escrito na nota do fornecedor.
		return decimal.Zero, decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf(
			"o componente %s é estocado em %s e a estrutura está em %s, mas não existe conversão entre as duas. "+
				"Cadastre a conversão do item no sentido que você souber — quantos %s tem 1 %s, ou quantos %s tem 1 %s "+
				"(o sistema usa a inversa automaticamente) — ou escreva a quantidade em %s. "+
				"Sem a conversão o planejamento leria %.4g %s como se fossem %s.",
			itemLabel, doEstoque, daEstrutura,
			daEstrutura, doEstoque, doEstoque, daEstrutura,
			doEstoque, quantidade.InexactFloat64(), daEstrutura, doEstoque))
	}
	// A quantidade sai da fração exata (1/85 × 42,5 = 0,5), não do fator já
	// arredondado em 10 casas (0,0117647059 × 42,5 = 0,5000000008), que a OP
	// multiplicava pelo lote e levava para o estoque.
	if exato, ok := conv.(conversorExato); ok {
		q, achou, err := exato.ConvertQuantityExact(ctx, itemCode, quantidade, daEstrutura, doEstoque)
		if err != nil {
			return decimal.Zero, decimal.Zero, fmt.Errorf("convertendo a quantidade do item %d: %w", itemCode, err)
		}
		if achou {
			return q, f, nil
		}
	}
	return quantidade.Mul(f), f, nil
}

// conversorExato é opcional no conversor (o de produção o tem).
type conversorExato interface {
	ConvertQuantityExact(ctx context.Context, itemCode int64, qty decimal.Decimal, fromUOM, toUOM string) (decimal.Decimal, bool, error)
}
