// Package margin_uc orquestra a margem de contribuição: parâmetros do mês,
// geração a partir das notas e a apuração para análise.
package margin_uc

import (
	"context"
	"fmt"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/margin"
)

type UseCase struct{ Repo *repo.Repository }

func New(r *repo.Repository) *UseCase { return &UseCase{Repo: r} }

// SalvarParametros grava os percentuais e prazos do mês.
func (uc *UseCase) SalvarParametros(ctx context.Context, p entity.Parametros) error {
	if p.Mes < 1 || p.Mes > 12 {
		return errorsuc.NewValidationError("o mês deve estar entre 1 e 12")
	}
	if p.Ano < 2000 || p.Ano > 2199 {
		return errorsuc.NewValidationError("informe um ano válido")
	}
	for nome, valor := range map[string]float64{
		"o percentual de IR":          p.IRPct,
		"a incidência administrativa": p.AdminPct,
		"o percentual de frete":       p.FreightPct,
		"a taxa financeira":           p.FinancialRateMonthly,
	} {
		if valor < 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("%s não pode ser negativo", nome))
		}
		if valor > 100 {
			return errorsuc.NewValidationError(fmt.Sprintf("%s não pode passar de 100%%", nome))
		}
	}
	return uc.Repo.UpsertParametros(ctx, p)
}

// ParametrosDoMes devolve os parâmetros e o ciclo de caixa já calculado.
func (uc *UseCase) ParametrosDoMes(ctx context.Context, ano, mes int) (*entity.Parametros, error) {
	return uc.Repo.GetParametros(ctx, ano, mes)
}

func (uc *UseCase) ListarParametros(ctx context.Context, ano int) ([]entity.Parametros, error) {
	return uc.Repo.ListParametros(ctx, ano)
}

// ResumoGeracao é o retorno da apuração.
type ResumoGeracao struct {
	LinhasCalculadas int     `json:"linhas_calculadas"`
	FaturamentoTotal float64 `json:"faturamento_total"`
	MargemTotal      float64 `json:"margem_total"`
	MargemPct        float64 `json:"margem_pct"`
	ComPrejuizo      int     `json:"linhas_com_prejuizo"`
	CostBasis        string  `json:"cost_basis"`
}

// Gerar apura a margem das notas emitidas no período.
//
// Os parâmetros usados são os do mês da emissão de cada nota — um período que
// cruza a virada do mês usa os dois, porque taxa e prazos mudam de um mês para
// o outro e reaproveitar os de um só distorceria metade das linhas.
func (uc *UseCase) Gerar(ctx context.Context, de, ate time.Time, base string) (*ResumoGeracao, error) {
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("a data final não pode ser anterior à inicial")
	}
	if base != "MEDIO" && base != "PADRAO" {
		return nil, errorsuc.NewValidationError("a base de custo deve ser médio ou padrão")
	}

	vendas, err := uc.Repo.ListarVendasDoPeriodo(ctx, de.Format("2006-01-02"), ate.Format("2006-01-02"), base)
	if err != nil {
		return nil, err
	}
	if len(vendas) == 0 {
		return &ResumoGeracao{CostBasis: base}, nil
	}

	// Um cache por mês evita reler os parâmetros a cada linha.
	cache := map[string]*entity.Parametros{}
	parametrosDe := func(dataISO string) (*entity.Parametros, error) {
		chave := dataISO[:7]
		if p, ok := cache[chave]; ok {
			return p, nil
		}
		d, err := time.Parse("2006-01-02", dataISO)
		if err != nil {
			return nil, errorsuc.NewValidationError("data de emissão inválida: " + dataISO)
		}
		p, err := uc.Repo.GetParametros(ctx, d.Year(), int(d.Month()))
		if err != nil {
			return nil, err
		}
		cache[chave] = p
		return p, nil
	}

	resumo := &ResumoGeracao{CostBasis: base}
	for _, v := range vendas {
		p, err := parametrosDe(v.IssueDate)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"os parâmetros de margem de %s não estão cadastrados; cadastre-os antes de apurar",
				v.IssueDate[:7]))
		}
		res := entity.Calcular(v.Venda, *p)
		if err := uc.Repo.GravarResultado(ctx, v, res, base); err != nil {
			return nil, err
		}
		resumo.LinhasCalculadas++
		resumo.FaturamentoTotal += res.FaturamentoMercadoria
		resumo.MargemTotal += res.Margem
		if res.Margem < 0 {
			resumo.ComPrejuizo++
		}
	}
	if resumo.FaturamentoTotal != 0 {
		resumo.MargemPct = resumo.MargemTotal / resumo.FaturamentoTotal * 100
	}
	return resumo, nil
}

// Apuracao devolve as linhas já calculadas para análise.
func (uc *UseCase) Apuracao(ctx context.Context, de, ate time.Time, ordem string, itemCode, customerCode *int64) ([]repo.LinhaApurada, error) {
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("a data final não pode ser anterior à inicial")
	}
	return uc.Repo.ListarApuracao(ctx, de.Format("2006-01-02"), ate.Format("2006-01-02"), ordem, itemCode, customerCode)
}
