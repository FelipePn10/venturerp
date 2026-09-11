package entity

import (
	"math"
	"time"
)

// CargaNoDia é a parcela de horas de uma operação que cai num dia.
type CargaNoDia struct {
	Data  time.Time
	Horas float64
}

// DistribuiCarga reparte as horas de uma operação pelos dias que ela realmente
// ocupa, a partir de `inicio`, respeitando a capacidade diária do recurso e
// pulando fins de semana.
//
// Por que isto importa: a versão anterior jogava a carga inteira do roteiro no
// dia da ordem. Uma rota de três dias virava um pico num dia só — o gráfico de
// capacidade acusava sobrecarga onde não havia e escondia a carga dos dias
// seguintes. Distribuir é o que SAP (capacity requirements ao longo da duração
// da operação) e FoccoERP fazem, e é o que torna o gráfico utilizável para
// decidir hora extra ou terceirização.
//
// `capacidadeDia <= 0` significa recurso sem capacidade cadastrada: nesse caso
// a carga fica num dia só, porque não há base para espalhar.
func DistribuiCarga(inicio time.Time, horas, capacidadeDia float64) []CargaNoDia {
	if horas <= 0 {
		return nil
	}
	dia := truncaParaODia(inicio)
	if capacidadeDia <= 0 {
		return []CargaNoDia{{Data: proximoDiaUtil(dia), Horas: horas}}
	}

	// Limite de segurança: uma operação absurdamente longa não pode gerar
	// milhares de linhas de carga.
	const maximoDeDias = 365

	restante := horas
	out := make([]CargaNoDia, 0, int(math.Ceil(horas/capacidadeDia)))
	atual := proximoDiaUtil(dia)
	for i := 0; restante > 0 && i < maximoDeDias; i++ {
		parcela := capacidadeDia
		if restante < parcela {
			parcela = restante
		}
		out = append(out, CargaNoDia{Data: atual, Horas: parcela})
		restante -= parcela
		atual = proximoDiaUtil(atual.AddDate(0, 0, 1))
	}
	if restante > 0 && len(out) > 0 {
		out[len(out)-1].Horas += restante
	}
	return out
}

func truncaParaODia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// proximoDiaUtil pula sábado e domingo: capacidade de fim de semana é zero por
// padrão, e lançar carga nesses dias produz um gráfico que ninguém consegue ler.
func proximoDiaUtil(t time.Time) time.Time {
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.AddDate(0, 0, 1)
	}
	return t
}
