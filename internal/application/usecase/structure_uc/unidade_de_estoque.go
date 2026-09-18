package structure_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
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
	quantidade float64,
	unidadeDaEstrutura string,
	unidadeDeEstoque string,
) (qtdeEmEstoque float64, fator float64, err error) {
	daEstrutura := strings.ToUpper(strings.TrimSpace(unidadeDaEstrutura))
	doEstoque := strings.ToUpper(strings.TrimSpace(unidadeDeEstoque))

	// Sem unidade declarada, ou unidades iguais: a quantidade já está na
	// unidade de estoque.
	if daEstrutura == "" || doEstoque == "" || daEstrutura == doEstoque {
		return quantidade, 1, nil
	}

	if conv == nil {
		// Defensivo: sem conversor injetado não dá para garantir o número, e
		// gravar uma quantidade cuja unidade ninguém sabe converter é o bug
		// que esta função existe para impedir.
		return 0, 0, errorsuc.NewValidationError(fmt.Sprintf(
			"o componente %s é estocado em %s e a estrutura está em %s; "+
				"o cadastro de conversões não está disponível nesta operação",
			itemLabel, doEstoque, daEstrutura))
	}

	f, achou, errConv := conv.Factor(ctx, itemCode, daEstrutura, doEstoque)
	if errConv != nil {
		return 0, 0, fmt.Errorf("consultando a conversão de unidade do item %d: %w", itemCode, errConv)
	}
	if !achou || f <= 0 {
		return 0, 0, errorsuc.NewValidationError(fmt.Sprintf(
			"o componente %s é estocado em %s, mas a estrutura está em %s e não existe conversão cadastrada "+
				"entre as duas. Cadastre a conversão do item (quantos %s tem 1 %s) ou use %s na estrutura — "+
				"sem isso o sistema reservaria %s achando que são %s.",
			itemLabel, doEstoque, daEstrutura, doEstoque, daEstrutura, doEstoque, daEstrutura, doEstoque))
	}
	return quantidade * f, f, nil
}
