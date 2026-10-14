// Package datetime provides tolerant parsing of date/time strings received
// from HTTP request payloads. Front-ends and integrations send dates in
// several shapes ("2006-01-02", RFC3339, "2006-01-02 15:04:05", ...); a
// strict single-layout time.Parse silently yields the zero value (0001-01-01)
// when the layout does not match, which is how emission/delivery dates were
// being persisted as 0001-01-01.
package datetime

import (
	"fmt"
	"strings"
	"time"
)

// acceptedLayouts is tried in order. Date-only comes first because that is the
// documented contract; the remaining layouts tolerate datetime payloads.
var acceptedLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"02/01/2006",
}

// ParseDate parses s using the accepted layouts. It returns the parsed time
// truncated to date precision and true on success; on empty input or when no
// layout matches it returns the zero time and false.
func ParseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ParseDateOrDefault parses s and falls back to def when s is empty or invalid.
func ParseDateOrDefault(s string, def time.Time) time.Time {
	if t, ok := ParseDate(s); ok {
		return t
	}
	return def
}

// ParseDatePtr parses an optional date string. A nil or unparseable pointer
// yields nil so callers can leave nullable DATE columns untouched.
func ParseDatePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	if t, ok := ParseDate(*s); ok {
		return &t
	}
	return nil
}

// CampoData é uma data em texto vinda da tela, com o nome que o usuário conhece.
type CampoData struct {
	Nome  string
	Valor *string
}

// Campo monta um CampoData.
func Campo(nome string, valor *string) CampoData { return CampoData{Nome: nome, Valor: valor} }

// PrimeiraInvalida devolve a mensagem da primeira data preenchida que não é uma
// data aceita, ou "" quando todas estão vazias ou válidas. ParseDatePtr devolve
// nil para texto inválido — sem esta conferência antes, a data mal digitada
// sumia em silêncio e o registro era gravado sem ela.
func PrimeiraInvalida(campos ...CampoData) string {
	for _, c := range campos {
		if c.Valor == nil || strings.TrimSpace(*c.Valor) == "" {
			continue
		}
		if _, ok := ParseDate(*c.Valor); !ok {
			return fmt.Sprintf("%s inválida (%q): use AAAA-MM-DD ou DD/MM/AAAA", c.Nome, strings.TrimSpace(*c.Valor))
		}
	}
	return ""
}

// Obrigatoria converte uma data obrigatória digitada na tela. msg vem
// preenchida quando a data falta ou não é uma data aceita — antes o
// time.Parse descartado gravava 0001-01-01 em silêncio.
func Obrigatoria(nome, valor string) (t time.Time, msg string) {
	if strings.TrimSpace(valor) == "" {
		return time.Time{}, "informe a " + nome
	}
	t, ok := ParseDate(valor)
	if !ok {
		return time.Time{}, fmt.Sprintf("%s inválida (%q): use AAAA-MM-DD ou DD/MM/AAAA", nome, strings.TrimSpace(valor))
	}
	return t, ""
}

// Opcional converte uma data opcional: vazia devolve nil; inválida devolve msg.
func Opcional(nome string, valor *string) (t *time.Time, msg string) {
	if valor == nil || strings.TrimSpace(*valor) == "" {
		return nil, ""
	}
	d, ok := ParseDate(*valor)
	if !ok {
		return nil, fmt.Sprintf("%s inválida (%q): use AAAA-MM-DD ou DD/MM/AAAA", nome, strings.TrimSpace(*valor))
	}
	return &d, ""
}

// Hora converte um horário opcional (HH:MM ou HH:MM:SS): vazio devolve nil.
func Hora(nome string, valor *string) (t *time.Time, msg string) {
	if valor == nil || strings.TrimSpace(*valor) == "" {
		return nil, ""
	}
	v := strings.TrimSpace(*valor)
	for _, layout := range []string{"15:04:05", "15:04"} {
		if h, err := time.Parse(layout, v); err == nil {
			return &h, ""
		}
	}
	return nil, fmt.Sprintf("%s inválido (%q): use HH:MM", nome, v)
}
