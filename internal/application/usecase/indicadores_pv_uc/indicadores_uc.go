// Package indicadores_pv_uc: painel de indicadores do pós-venda (Fase 3.5).
package indicadores_pv_uc

import (
	"context"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/indicadores"
)

type UseCase struct {
	Repo  dom.Repository
	Agora func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

func dia(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, errorsuc.NewValidationError("data inválida (use AAAA-MM-DD)")
	}
	return &t, nil
}

// Periodo: de 'de' até 'ate' inclusive; sem datas, os últimos 90 dias.
func (uc *UseCase) Periodo(de, ate string) (dom.Periodo, error) {
	d, err := dia(de)
	if err != nil {
		return dom.Periodo{}, err
	}
	a, err := dia(ate)
	if err != nil {
		return dom.Periodo{}, err
	}
	hoje := uc.agora()
	hoje = time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.Local)
	if a == nil {
		a = &hoje
	}
	if d == nil {
		v := a.AddDate(0, 0, -89)
		d = &v
	}
	if a.Before(*d) {
		return dom.Periodo{}, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	if a.Sub(*d) > 3*366*24*time.Hour {
		return dom.Periodo{}, errorsuc.NewValidationError("o período dos indicadores vai até 3 anos")
	}
	return dom.Periodo{De: *d, Ate: a.AddDate(0, 0, 1)}, nil
}

func (uc *UseCase) Painel(ctx context.Context, de, ate string) (*dom.Painel, error) {
	p, err := uc.Periodo(de, ate)
	if err != nil {
		return nil, err
	}
	out := &dom.Painel{Periodo: p}
	if out.Resumo, err = uc.Repo.Resumo(ctx, p); err != nil {
		return nil, err
	}
	cs, err := uc.Repo.ChamadosComSLA(ctx, p)
	if err != nil {
		return nil, err
	}
	out.SLA.Avaliar(cs, uc.agora())
	for dim, dst := range map[string]*[]dom.Grupo{"item": &out.PorItem, "defeito": &out.PorDefeito, "lote": &out.PorLote} {
		if *dst, err = uc.Repo.CustoPor(ctx, p, dim); err != nil {
			return nil, err
		}
	}
	if out.Confiabilidade, err = uc.Repo.Confiabilidade(ctx, p); err != nil {
		return nil, err
	}
	for i := range out.Confiabilidade {
		out.Confiabilidade[i].Calcular()
	}
	return out, nil
}
