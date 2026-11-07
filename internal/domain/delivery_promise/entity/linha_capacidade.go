package entity

import (
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// LinhaDeCapacidade é o tanque ou a linha em que o item ocupa capacidade na
// promessa de entrega (pintura, galvanização, montagem). Cada uma tem a sua
// capacidade por dia.
type LinhaDeCapacidade struct {
	Codigo           int64           `json:"code"`
	Descricao        string          `json:"description"`
	CapacidadeDiaria decimal.Decimal `json:"daily_capacity"`
	Unidade          string          `json:"uom"`
	Ativa            bool            `json:"is_active"`
}

// Conferir normaliza e valida antes de gravar.
func (l *LinhaDeCapacidade) Conferir() error {
	l.Descricao = strings.Join(strings.Fields(l.Descricao), " ")
	l.Unidade = strings.ToUpper(strings.TrimSpace(l.Unidade))
	if l.Unidade == "" {
		l.Unidade = "UN"
	}
	if l.Codigo <= 0 {
		return domainerr.NewValidation("informe o código da linha, um número maior que zero")
	}
	if l.Descricao == "" || utf8.RuneCountInString(l.Descricao) > 120 {
		return domainerr.NewValidation("informe a descrição da linha, com até 120 caracteres")
	}
	if !l.CapacidadeDiaria.IsPositive() {
		return domainerr.NewValidation("a capacidade diária da linha deve ser maior que zero")
	}
	if utf8.RuneCountInString(l.Unidade) > 20 {
		return domainerr.NewValidation("a unidade da capacidade tem no máximo 20 caracteres")
	}
	return nil
}
