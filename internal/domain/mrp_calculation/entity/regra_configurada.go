package entity

import (
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Campos de regra configurada que o MRP lê.
const (
	CampoRegraPrazo      = "lead_time"
	CampoRegraLoteMinimo = "lote_minimo"
)

// CasaComMascara diz se a regra vale para a máscara. Sem condição, vale para
// todas; com condição, o tipo da regra compara a máscara com ela (EQUAL,
// DIFFERENT, RANGE "a-b" numérico). Tipo desconhecido com condição: vale.
func (r *ConfiguredItemRule) CasaComMascara(mask string) bool {
	cond := strings.TrimSpace(r.CondicaoMascara)
	if cond == "" {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(r.RuleType)) {
	case "EQUAL", "EQUALS":
		return mask == cond
	case "DIFFERENT":
		return mask != cond
	case "RANGE":
		minimo, maximo, ok := faixa(cond)
		if !ok {
			return false
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(mask), 64)
		return err == nil && v >= minimo && v <= maximo
	}
	return true
}

func faixa(cond string) (float64, float64, bool) {
	partes := strings.SplitN(cond, "-", 2)
	if len(partes) != 2 {
		return 0, 0, false
	}
	minimo, e1 := strconv.ParseFloat(strings.TrimSpace(partes[0]), 64)
	maximo, e2 := strconv.ParseFloat(strings.TrimSpace(partes[1]), 64)
	return minimo, maximo, e1 == nil && e2 == nil && minimo <= maximo
}

// ValidarRegraConfigurada confere o valor do campo que o MRP lê e a condição
// de máscara. Campo que o MRP não lê aceita qualquer valor.
func ValidarRegraConfigurada(campo, tipo, valor, condicao string) string {
	valor = strings.TrimSpace(valor)
	switch strings.TrimSpace(campo) {
	case CampoRegraPrazo:
		if n, err := strconv.Atoi(valor); err != nil || n < 0 || n > 3650 {
			return "o prazo (lead_time) é um número inteiro de dias úteis, de 0 a 3650"
		}
	case CampoRegraLoteMinimo:
		if d, err := decimal.NewFromString(strings.ReplaceAll(valor, ",", ".")); err != nil || d.IsNegative() {
			return "o lote mínimo é um número maior ou igual a zero"
		}
	}
	condicao = strings.TrimSpace(condicao)
	if condicao != "" && strings.EqualFold(strings.TrimSpace(tipo), "RANGE") {
		if _, _, ok := faixa(condicao); !ok {
			return "a condição de faixa é \"mínimo-máximo\" (ex.: 100-200)"
		}
	}
	return ""
}
