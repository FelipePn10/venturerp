// Package pendencias_uc: a central de pendências do dia, por perfil (melhoria 8).
package pendencias_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	domain "github.com/FelipePn10/panossoerp/internal/domain/pendencias"
	thirdparty "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
)

type Contador interface {
	Contagens(ctx context.Context) (map[string]domain.Contagem, error)
}

// EmPoder é o saldo com terceiros (OS de terceiro).
type EmPoder interface {
	EmPoder(ctx context.Context, f thirdparty.FiltroEmPoder) ([]thirdparty.EmPoder, error)
}

// Acesso diz se o usuário da sessão lê o módulo (o perfil dele).
type Acesso interface {
	PodeNoModulo(ctx context.Context, m acesso.Modulo, a acesso.Acao) (bool, error)
}

type CentralUseCase struct {
	Repo      Contador
	Terceiros EmPoder // opcional
	Acesso    Acesso  // opcional: sem ele, mostra tudo
}

// DoDia devolve as pendências que o perfil da sessão pode resolver.
func (uc *CentralUseCase) DoDia(ctx context.Context) ([]domain.Pendencia, error) {
	c, err := uc.Repo.Contagens(ctx)
	if err != nil {
		return nil, err
	}
	if uc.Terceiros != nil {
		vencidas, err := uc.Terceiros.EmPoder(ctx, thirdparty.FiltroEmPoder{OnlyOverdue: true})
		if err != nil {
			return nil, err
		}
		t := domain.Contagem{Quantidade: len(vencidas)}
		for _, v := range vencidas {
			t.Valor = t.Valor.Add(v.Value)
			if t.MaisAntiga == nil || v.FirstRemittance.Before(*t.MaisAntiga) {
				quando := v.FirstRemittance
				t.MaisAntiga = &quando
			}
		}
		c["remessas_terceiro_vencidas"] = t
	}
	var falha error
	pode := func(m acesso.Modulo) bool {
		if uc.Acesso == nil {
			return true
		}
		ok, err := uc.Acesso.PodeNoModulo(ctx, m, acesso.Ler)
		if err != nil && falha == nil {
			falha = err
		}
		return ok
	}
	out := domain.DoPerfil(c, pode)
	return out, falha
}
