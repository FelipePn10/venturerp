package entity

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// EventoDeEtapa é uma mudança de situação da etapa da OF, na ordem em que
// aconteceu (a trilha de auditoria da execução).
type EventoDeEtapa struct {
	Status string
	Quando time.Time
}

// InicioDoTrecho devolve o instante em que a etapa entrou em andamento pela
// última vez — o começo do trecho que está correndo agora. Pausa e
// interrupção fecham o trecho; retomar abre outro. Sem evento de andamento,
// devolve false (etapa anterior à trilha de auditoria).
func InicioDoTrecho(eventos []EventoDeEtapa) (time.Time, bool) {
	var inicio time.Time
	achou := false
	for _, e := range eventos {
		if e.Status == "IN_PROGRESS" {
			inicio, achou = e.Quando, true
		}
	}
	return inicio, achou
}

// HorasDoTrecho é o tempo corrido entre o início do trecho e o fim, em horas
// com 4 casas (a escala de actual_hours). É o cronômetro da etapa: as horas
// que o operador não precisa digitar. Relógio que volta dá zero, nunca
// negativo.
func HorasDoTrecho(inicio, fim time.Time) float64 {
	if inicio.IsZero() || !fim.After(inicio) {
		return 0
	}
	return math.Round(fim.Sub(inicio).Hours()*10000) / 10000
}

// HorasEtapaEsquecida: acima disso o cronômetro não grava sozinho — a etapa
// quase certamente ficou aberta (virou a noite) e o tempo parado entraria como
// trabalhado. Quem fecha precisa digitar as horas.
const HorasEtapaEsquecida = 12.0

// HorasPorExtenso: "15,0 h" — as horas com vírgula, para a mensagem ao usuário.
func HorasPorExtenso(horas float64) string {
	return strings.Replace(fmt.Sprintf("%.1f h", horas), ".", ",", 1)
}

// TrechoEsquecido diz se o tempo corrido passou do limite do cronômetro.
func TrechoEsquecido(horas float64) bool { return horas > HorasEtapaEsquecida }

// FechaTrecho diz se a transição encerra o trecho em andamento: pausar,
// interromper ou concluir. É nesse momento que o tempo corrido entra nas horas
// da etapa.
func FechaTrecho(de, para string) bool {
	return de == "IN_PROGRESS" && (para == "PAUSED" || para == "INTERRUPTED" || para == "DONE")
}
