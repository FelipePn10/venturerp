package unit_of_measure_uc

import (
	"context"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// Catalogo é o que as telas de compra precisam do catálogo de unidades.
type Catalogo interface {
	UnidadesForaDoCatalogo(ctx context.Context, codigos ...string) ([]string, error)
}

// ConferirNoCatalogo recusa unidade fora do catálogo da empresa (ou inativa).
// `onde` nomeia o campo na mensagem ("unidade do preço"). Sem catálogo
// injetado, ou sem unidade informada, não confere.
//
// A tabela de preço e a solicitação de compra aceitavam texto livre ("KGS",
// "CXA"): a tela já oferecia só o catálogo, mas uma integração gravava e o
// erro só aparecia no pedido, quando a conversão não achava a unidade.
func ConferirNoCatalogo(ctx context.Context, cat Catalogo, onde string, codigos ...*string) error {
	if cat == nil {
		return nil
	}
	var pedir []string
	for _, c := range codigos {
		if c != nil && strings.TrimSpace(*c) != "" {
			pedir = append(pedir, strings.ToUpper(strings.TrimSpace(*c)))
		}
	}
	if len(pedir) == 0 {
		return nil
	}
	fora, err := cat.UnidadesForaDoCatalogo(ctx, pedir...)
	if err != nil {
		return fmt.Errorf("conferindo a unidade no catálogo: %w", err)
	}
	if len(fora) == 0 {
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf("%s fora do catálogo da empresa ou inativa: %s — cadastre-a em Unidades e Conversões (VSUP0110 › Unidades) ou escolha outra",
		onde, strings.Join(fora, ", ")))
}

// NormalizarUnidade devolve a unidade em maiúsculas e sem espaços (nil continua nil).
func NormalizarUnidade(u *string) *string {
	if u == nil {
		return nil
	}
	t := strings.ToUpper(strings.TrimSpace(*u))
	if t == "" {
		return nil
	}
	return &t
}
