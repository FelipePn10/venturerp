package aps_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

// filhasDasOrdens lê a cadeia OF filha → OF pai das ordens da rodada (vazio
// quando o repositório não a conhece).
// O segundo retorno é o término já programado das filhas que ficaram fora da
// rodada.
func (uc *APSUseCase) filhasDasOrdens(ctx context.Context, orders []repository.OrderRow) (map[int64][]int64, map[int64]time.Time, error) {
	cadeia, ok := uc.repo.(repository.CadeiaRepository)
	if !ok || len(orders) == 0 {
		return nil, nil, nil
	}
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	filhas, err := cadeia.OrdensFilhas(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("lendo a cadeia de fabricação das ordens: %w", err)
	}
	naRodada := make(map[int64]bool, len(ids))
	for _, id := range ids {
		naRodada[id] = true
	}
	var fora []int64
	for _, fs := range filhas {
		for _, f := range fs {
			if !naRodada[f] {
				fora = append(fora, f)
			}
		}
	}
	fins, err := cadeia.FimProgramado(ctx, fora)
	if err != nil {
		return nil, nil, err
	}
	return filhas, fins, nil
}

// ordenaPorCadeia põe cada OF filha (a bucha) antes da OF pai (o RN) que ela
// abastece, mantendo, no resto, a ordem de prioridade e entrega já decidida.
// Uma filha fora da rodada não muda nada; um ciclo (que o cadastro não deveria
// permitir) é quebrado na ordem original, sem travar o sequenciamento.
func ordenaPorCadeia(orders []repository.OrderRow, filhas map[int64][]int64) []repository.OrderRow {
	if len(filhas) == 0 {
		return orders
	}
	porID := make(map[int64]repository.OrderRow, len(orders))
	for _, o := range orders {
		porID[o.ID] = o
	}
	saida := make([]repository.OrderRow, 0, len(orders))
	feito := make(map[int64]bool, len(orders))
	visitando := make(map[int64]bool)
	var visitar func(id int64)
	visitar = func(id int64) {
		if feito[id] || visitando[id] {
			return
		}
		visitando[id] = true
		for _, f := range filhas[id] {
			if _, naRodada := porID[f]; naRodada {
				visitar(f)
			}
		}
		visitando[id] = false
		feito[id] = true
		saida = append(saida, porID[id])
	}
	for _, o := range orders {
		visitar(o.ID)
	}
	return saida
}

// inicioPelaCadeia: a OF pai só começa quando a última filha programada nesta
// rodada termina (a peça tem de existir para entrar na primeira etapa).
func inicioPelaCadeia(inicio time.Time, filhas []int64, fimDaOrdem map[int64]time.Time) time.Time {
	for _, f := range filhas {
		if fim, ok := fimDaOrdem[f]; ok {
			inicio = maxTime(inicio, fim)
		}
	}
	return inicio
}
