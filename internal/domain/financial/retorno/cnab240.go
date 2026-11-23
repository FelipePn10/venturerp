// Package retorno lê o arquivo de retorno do banco (CNAB 240, FEBRABAN) e devolve
// as liquidações de boleto (melhoria 10). A remessa fica em infrastructure/cnab.
//
// Layout dos segmentos de detalhe (posições 1-based do FEBRABAN 240):
//
//	T: 14 segmento, 16-17 movimento, 38-57 nosso número, 82-96 valor nominal
//	U: 14 segmento, 16-17 movimento, 18-32 juros/multa, 33-47 desconto,
//	   48-62 abatimento, 78-92 valor pago, 93-107 valor líquido,
//	   138-145 data da ocorrência, 146-153 data do crédito (DDMMAAAA)
//
// Os bancos divergem em posições do T (carteira, nosso número com DV): o
// arquivo de cada banco precisa ser homologado antes do primeiro uso.
package retorno

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Movimentos de liquidação: 06 (liquidação) e 17 (liquidação após baixa).
var movimentosDeLiquidacao = map[string]bool{"06": true, "17": true}

// Liquidacao é um boleto pago.
type Liquidacao struct {
	NossoNumero string
	Movimento   string
	ValorTitulo decimal.Decimal
	ValorPago   decimal.Decimal
	Juros       decimal.Decimal
	Desconto    decimal.Decimal
	Ocorrencia  time.Time
	Credito     time.Time
	LinhaDoT    int
}

// Ocorrencia é um registro que não é liquidação (entrada confirmada, baixa,
// rejeição): volta para quem processou ver.
type Ocorrencia struct {
	NossoNumero, Movimento string
	Linha                  int
}

// Ler devolve as liquidações e as demais ocorrências do arquivo.
func Ler(conteudo string) ([]Liquidacao, []Ocorrencia, error) {
	var liq []Liquidacao
	var outras []Ocorrencia
	var pendente *Liquidacao
	sc := bufio.NewScanner(strings.NewReader(conteudo))
	sc.Buffer(make([]byte, 0, 1024), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		l := strings.TrimRight(sc.Text(), "\r\n")
		if strings.TrimSpace(l) == "" {
			continue
		}
		if len(l) < 240 {
			return nil, nil, fmt.Errorf("linha %d tem %d posições: o retorno CNAB 240 tem 240", n, len(l))
		}
		if l[7] != '3' { // só detalhe
			continue
		}
		mov := l[15:17]
		switch l[13] {
		case 'T':
			nn := strings.TrimLeft(strings.TrimSpace(l[37:57]), "0")
			if !movimentosDeLiquidacao[mov] {
				outras = append(outras, Ocorrencia{NossoNumero: nn, Movimento: mov, Linha: n})
				pendente = nil
				continue
			}
			v, err := valor(l[81:96])
			if err != nil {
				return nil, nil, fmt.Errorf("linha %d: valor do título: %w", n, err)
			}
			liq = append(liq, Liquidacao{NossoNumero: nn, Movimento: mov, ValorTitulo: v, LinhaDoT: n})
			pendente = &liq[len(liq)-1]
		case 'U':
			if pendente == nil {
				continue
			}
			var err error
			campos := []struct {
				dst *decimal.Decimal
				s   string
			}{{&pendente.Juros, l[17:32]}, {&pendente.Desconto, l[32:47]}, {&pendente.ValorPago, l[77:92]}}
			for _, c := range campos {
				if *c.dst, err = valor(c.s); err != nil {
					return nil, nil, fmt.Errorf("linha %d: %w", n, err)
				}
			}
			if pendente.Ocorrencia, err = data(l[137:145]); err != nil {
				return nil, nil, fmt.Errorf("linha %d: data da ocorrência: %w", n, err)
			}
			if pendente.Credito, err = data(l[145:153]); err != nil || pendente.Credito.IsZero() {
				pendente.Credito = pendente.Ocorrencia
			}
			pendente = nil
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	for _, x := range liq {
		if x.Ocorrencia.IsZero() {
			return nil, nil, fmt.Errorf("linha %d: liquidação sem o segmento U (valor pago e data)", x.LinhaDoT)
		}
	}
	return liq, outras, nil
}

// valor: 15 posições, 2 decimais implícitas.
func valor(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, nil
	}
	v, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("número inválido %q", s)
	}
	return v.Shift(-2), nil
}

func data(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.Trim(s, "0") == "" {
		return time.Time{}, nil
	}
	return time.Parse("02012006", s)
}
