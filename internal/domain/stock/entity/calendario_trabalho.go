package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// CalendarioDeTrabalho diz quais dias a empresa trabalha. O dia registrado no
// calendário industrial vale como está (sábado trabalhado, feriado parado); o
// que não está registrado é trabalhado de segunda a sexta — a mesma regra da
// função subtract_workdays do banco, que o MRP usa.
type CalendarioDeTrabalho struct {
	Excecoes map[string]bool // "2026-11-14" → trabalha?
}

func (c CalendarioDeTrabalho) Trabalha(d time.Time) bool {
	if v, ok := c.Excecoes[d.Format("2006-01-02")]; ok {
		return v
	}
	return d.Weekday() != time.Saturday && d.Weekday() != time.Sunday
}

// DiasTrabalhados conta os dias trabalhados de `de` até `ate`, inclusive.
func (c CalendarioDeTrabalho) DiasTrabalhados(de, ate time.Time) int {
	n := 0
	for d := dia(de); !d.After(dia(ate)); d = d.AddDate(0, 0, 1) {
		if c.Trabalha(d) {
			n++
		}
	}
	return n
}

// DiasDeConsumoNoPrazo: o fornecedor entrega em `prazo` dias úteis (segunda a
// sexta, a contar de amanhã); quantos desses dias a empresa trabalha e consome.
// Sábado trabalhado no meio do prazo aumenta o consumo; feriado parado diminui.
func (c CalendarioDeTrabalho) DiasDeConsumoNoPrazo(hoje time.Time, prazo int) int {
	if prazo <= 0 {
		return 0
	}
	fim, uteis := dia(hoje), 0
	for uteis < prazo {
		fim = fim.AddDate(0, 0, 1)
		if fim.Weekday() != time.Saturday && fim.Weekday() != time.Sunday {
			uteis++
		}
	}
	return c.DiasTrabalhados(dia(hoje).AddDate(0, 0, 1), fim)
}

// PontoPorDiaTrabalhado é consumo por dia × dias de consumo no prazo + segurança,
// arredondado para cima.
func PontoPorDiaTrabalhado(consumoPorDia decimal.Decimal, diasDeConsumo int, seguranca decimal.Decimal) decimal.Decimal {
	return consumoPorDia.Mul(decimal.NewFromInt(int64(diasDeConsumo))).Add(seguranca).Ceil()
}

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
