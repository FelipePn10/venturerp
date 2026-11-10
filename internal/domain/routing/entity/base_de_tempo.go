package entity

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Base do tempo de máquina: em que medida da peça o tempo é contado.
//
// Na moveleira e na metalúrgica o tempo raramente é "por peça": a fita de borda
// corre pelo perímetro, a pintura pela área, o corte de tubo pelo comprimento,
// a furação pelo número de furos. Com a base certa, um único cadastro de
// operação vale para a porta de 40 cm e para a de 2 m — o tempo sai da medida
// do item, sem roteiro por tamanho.
const (
	BasePeca        = "PECA"
	BaseM2          = "M2"
	BaseMetroLinear = "METRO_LINEAR"
	BasePerimetro   = "PERIMETRO"
	BaseAtributo    = "ATRIBUTO"
)

// BasesDeTempo é a lista aceita, na ordem em que a tela mostra.
var BasesDeTempo = []string{BasePeca, BaseM2, BaseMetroLinear, BasePerimetro, BaseAtributo}

// NormalizaBaseDeTempo valida a base e o atributo. Base vazia = por peça.
func NormalizaBaseDeTempo(base string, atributo *string) (string, *string, error) {
	b := strings.ToUpper(strings.TrimSpace(base))
	if b == "" {
		b = BasePeca
	}
	valida := false
	for _, v := range BasesDeTempo {
		if v == b {
			valida = true
		}
	}
	if !valida {
		return "", nil, domainerr.NewValidation(fmt.Sprintf(
			"base de tempo %q inválida: use PECA, M2, METRO_LINEAR, PERIMETRO ou ATRIBUTO", base))
	}
	if b != BaseAtributo {
		return b, nil, nil
	}
	if atributo == nil || strings.TrimSpace(*atributo) == "" {
		return "", nil, domainerr.NewValidation(
			"informe o nome do atributo do item (ex.: FUROS, METROS_FITA) para o tempo contado por atributo")
	}
	nome := strings.ToUpper(strings.TrimSpace(*atributo))
	return b, &nome, nil
}

// MedidasDaPeca é o que o cadastro de engenharia do item diz sobre ele.
// Comprimento e largura vêm em MILÍMETROS (items.engineering_dimensions).
type MedidasDaPeca struct {
	ComprimentoMM float64
	LarguraMM     float64
	Atributos     map[string]string // atributos do PDM, nome em maiúsculas
}

// FatorDaBase diz quanto da base cada peça tem. ok=false quando o item não
// informa a medida que a base pede — a etapa não tem como ser dimensionada.
func FatorDaBase(base string, atributo *string, m MedidasDaPeca) (fator float64, ok bool) {
	switch base {
	case "", BasePeca:
		return 1, true
	case BaseM2:
		if m.ComprimentoMM <= 0 || m.LarguraMM <= 0 {
			return 0, false
		}
		return m.ComprimentoMM * m.LarguraMM / 1e6, true
	case BaseMetroLinear:
		if m.ComprimentoMM <= 0 {
			return 0, false
		}
		return m.ComprimentoMM / 1000, true
	case BasePerimetro:
		if m.ComprimentoMM <= 0 || m.LarguraMM <= 0 {
			return 0, false
		}
		return 2 * (m.ComprimentoMM + m.LarguraMM) / 1000, true
	case BaseAtributo:
		if atributo == nil {
			return 0, false
		}
		bruto, existe := m.Atributos[strings.ToUpper(strings.TrimSpace(*atributo))]
		if !existe {
			return 0, false
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(bruto), ",", "."), 64)
		if err != nil || v < 0 {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

// ErroSemMedida explica, para quem planeja, por que a etapa não tem tempo.
// Zero horas em silêncio fariam a OP, a capacidade e o custo saírem baratos e
// rápidos demais; a recusa leva a pessoa direto ao cadastro que falta.
func ErroSemMedida(op *RouteOperation) error {
	return domainerr.NewValidation(fmt.Sprintf(
		"a etapa %d (%s) conta o tempo %s e o item não informa essa medida: complete as dimensões ou o atributo no cadastro do item",
		op.Sequence, op.OperationName, RotuloDaBase(op.EffectiveTimeBasis, op.EffectiveTimeBasisAttribute)))
}

// RotuloDaBase é o nome da base para mensagens ("por m²").
func RotuloDaBase(base string, atributo *string) string {
	switch base {
	case BaseM2:
		return "por m²"
	case BaseMetroLinear:
		return "por metro linear"
	case BasePerimetro:
		return "por metro de perímetro"
	case BaseAtributo:
		if atributo != nil {
			return "por " + strings.ToLower(*atributo)
		}
		return "por atributo"
	}
	return "por peça"
}
