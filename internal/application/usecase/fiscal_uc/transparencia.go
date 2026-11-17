package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	ibptentity "github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
)

// LeitorIBPT acha a alíquota aproximada de tributos (tabela IBPT, VFIS0110) do
// NCM na UF.
type LeitorIBPT interface {
	GetByNCM(ctx context.Context, ncm, uf string) (*ibptentity.IBPTRate, error)
}

// linhaTransparencia: o que a Lei 12.741/2012 soma de cada linha.
type linhaTransparencia struct {
	ncm, origem string
	valor       decimal.Decimal
}

// textoTransparencia: "valor aproximado dos tributos" da venda a consumidor
// final (Lei 12.741/2012; Decreto 8.264/2014 aceita o texto nas informações
// complementares). Federal pela alíquota nacional ou de importado conforme a
// origem; estadual e municipal da IBPT da UF da empresa. Linha sem IBPT
// vigente fica fora — e, sem nenhuma, não há texto.
func textoTransparencia(ctx context.Context, leitor LeitorIBPT, uf string, linhas []linhaTransparencia, hoje time.Time) (string, error) {
	if leitor == nil {
		return "", nil
	}
	importada := map[string]bool{"1": true, "2": true, "3": true, "8": true}
	federal, estadual, municipal := decimal.Zero, decimal.Zero, decimal.Zero
	fonte, achou := "", false
	for _, l := range linhas {
		ncm := soDigitos(l.ncm)
		if ncm == "" || !l.valor.IsPositive() {
			continue
		}
		r, err := leitor.GetByNCM(ctx, ncm, strings.ToUpper(strings.TrimSpace(uf)))
		if err != nil {
			if errorsuc.EhNaoEncontrado(err) {
				continue
			}
			return "", fmt.Errorf("lendo a IBPT do NCM %s: %w", ncm, err)
		}
		if r == nil || (r.VigenciaFim != nil && r.VigenciaFim.Before(time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, r.VigenciaFim.Location()))) {
			continue
		}
		pctFed := r.NacionalFederal
		if importada[strings.TrimSpace(l.origem)] {
			pctFed = r.ImportadoFederal
		}
		federal = federal.Add(l.valor.Mul(decimal.NewFromFloat(pctFed)).Div(decimal.NewFromInt(100)))
		estadual = estadual.Add(l.valor.Mul(decimal.NewFromFloat(r.Estadual)).Div(decimal.NewFromInt(100)))
		municipal = municipal.Add(l.valor.Mul(decimal.NewFromFloat(r.Municipal)).Div(decimal.NewFromInt(100)))
		achou = true
		if fonte == "" {
			fonte = strings.TrimSpace(r.Fonte)
			if r.Chave != nil && strings.TrimSpace(*r.Chave) != "" {
				fonte += " " + strings.TrimSpace(*r.Chave)
			}
		}
	}
	if !achou {
		return "", nil
	}
	if fonte == "" {
		fonte = "IBPT"
	}
	brl := func(d decimal.Decimal) string { return strings.Replace(d.Round(2).StringFixed(2), ".", ",", 1) }
	return fmt.Sprintf("Valor aproximado dos tributos: R$ %s federais, R$ %s estaduais e R$ %s municipais (Lei 12.741/2012). Fonte: %s.",
		brl(federal), brl(estadual), brl(municipal), fonte), nil
}
