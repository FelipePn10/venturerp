package request

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DataFlexivel aceita tanto "2026-09-20" quanto o RFC3339 completo.
//
// Existe pelo mesmo motivo de CodigoFlexivel: o `<input type="date">` do
// navegador envia SOMENTE a data, e `time.Time` só sabe ler RFC3339. O
// resultado era um 400 com o erro cru do Go na cara do usuário —
// `parsing time "2026-09-20" as "2006-01-02T15:04:05Z07:00": cannot parse ""
// as "T"` —, que além de incompreensível está em inglês.
//
// Data sem hora vale meia-noite, que é o que "vigência a partir de" significa.
type DataFlexivel struct {
	time.Time
}

const formatoDataSimples = "2006-01-02"

func (d *DataFlexivel) UnmarshalJSON(dados []byte) error {
	texto := strings.TrimSpace(strings.Trim(string(dados), `"`))
	if texto == "" || texto == "null" {
		d.Time = time.Time{}
		return nil
	}
	if t, err := time.Parse(time.RFC3339, texto); err == nil {
		d.Time = t
		return nil
	}
	t, err := time.Parse(formatoDataSimples, texto)
	if err != nil {
		return fmt.Errorf("data %q inválida: use AAAA-MM-DD", texto)
	}
	d.Time = t
	return nil
}

func (d DataFlexivel) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time)
}

// Ponteiro devolve nil quando não veio data, que é o que os casos de uso esperam.
func (d *DataFlexivel) Ponteiro() *time.Time {
	if d == nil || d.Time.IsZero() {
		return nil
	}
	t := d.Time
	return &t
}
