// Package garantia decide se um equipamento ou peça está em garantia e o que
// ela cobre (pós-venda, Fase 3.2). É uma regra pura: recebe o perfil do item e
// as datas do que foi vendido e devolve a resposta com o motivo.
package garantia

import (
	"fmt"
	"time"
)

const (
	InicioVenda      = "VENDA"
	InicioInstalacao = "INSTALACAO"

	OrigemSerie     = "SERIE"
	OrigemNF        = "NF"
	OrigemInformada = "INFORMADA"
)

// Regra: a garantia que o perfil do item dá.
type Regra struct {
	Meses             int    `json:"garantia_meses"`
	Inicio            string `json:"garantia_inicio"`
	CobrePecas        bool   `json:"cobre_pecas"`
	CobreMaoObra      bool   `json:"cobre_mao_obra"`
	CobreDeslocamento bool   `json:"cobre_deslocamento"`
	// DiasDoItem: o "tempo de garantia (dias)" do cadastro do item, usado
	// quando o perfil não define meses.
	DiasDoItem int `json:"dias_do_item,omitempty"`
}

// Base: as datas do que foi vendido.
type Base struct {
	Origem       string
	Venda        *time.Time
	Instalacao   *time.Time
	EstendidaAte *time.Time
}

type Resultado struct {
	Origem            string     `json:"origem"`
	InicioEm          *time.Time `json:"inicio_em,omitempty"`
	Ate               *time.Time `json:"ate,omitempty"`
	EmGarantia        bool       `json:"em_garantia"`
	Estendida         bool       `json:"estendida"`
	CobrePecas        bool       `json:"cobre_pecas"`
	CobreMaoObra      bool       `json:"cobre_mao_obra"`
	CobreDeslocamento bool       `json:"cobre_deslocamento"`
	Motivo            string     `json:"motivo"`
}

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

func br(t time.Time) string { return t.Format("02/01/2006") }

// Determinar: a garantia começa na venda (ou na instalação, se o perfil manda e
// a instalação foi registrada) e vale até o último dia do prazo, inclusive. A
// garantia estendida da série prevalece quando vai além.
func Determinar(r Regra, b Base, hoje time.Time) Resultado {
	out := Resultado{Origem: b.Origem, CobrePecas: r.CobrePecas, CobreMaoObra: r.CobreMaoObra, CobreDeslocamento: r.CobreDeslocamento}
	inicio := b.Venda
	if r.Inicio == InicioInstalacao && b.Instalacao != nil {
		inicio = b.Instalacao
	}
	if inicio == nil {
		out.Motivo = "sem data de venda: a garantia não pode ser calculada"
		return out
	}
	i := dia(*inicio)
	out.InicioEm = &i
	var ate *time.Time
	switch {
	case r.Meses > 0:
		a := i.AddDate(0, r.Meses, -1)
		ate = &a
	case r.DiasDoItem > 0:
		a := i.AddDate(0, 0, r.DiasDoItem-1)
		ate = &a
	}
	if b.EstendidaAte != nil && (ate == nil || dia(*b.EstendidaAte).After(*ate)) {
		e := dia(*b.EstendidaAte)
		ate = &e
		out.Estendida = true
	}
	if ate == nil {
		out.Motivo = "o item não tem garantia definida (perfil de pós-venda ou cadastro do item)"
		return out
	}
	out.Ate = ate
	out.EmGarantia = !dia(hoje).After(*ate)
	quando := "a partir da venda"
	if r.Inicio == InicioInstalacao && b.Instalacao != nil {
		quando = "a partir da instalação"
	}
	estado := "em garantia"
	if !out.EmGarantia {
		estado = "garantia vencida"
	}
	ext := ""
	if out.Estendida {
		ext = " (garantia estendida)"
	}
	out.Motivo = fmt.Sprintf("%s até %s, contada %s (%s)%s", estado, br(*ate), quando, br(i), ext)
	if !out.EmGarantia {
		out.CobrePecas, out.CobreMaoObra, out.CobreDeslocamento = false, false, false
	}
	return out
}
