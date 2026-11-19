package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/shopspring/decimal"
)

// FatoresDaEstruturaRepository lê e regrava a quantidade convertida para a
// unidade de estoque das linhas de estrutura (ver structure_uc.RecalcularFatores).
type FatoresDaEstruturaRepository interface {
	LinhasParaConferir(ctx context.Context, filtro entity.FiltroDeFatores) ([]entity.LinhaDeFator, error)
	// GravarFator regrava quantidade e fator da linha e registra no histórico
	// da estrutura o antes e o depois. Só grava se a linha e a unidade de
	// estoque do componente ainda estão como foram lidas; gravou=false quando
	// alguém mudou no meio (a gravação mais nova vale) ou a linha é de outra
	// empresa.
	GravarFator(ctx context.Context, linha entity.LinhaDeFator, qtd, fator decimal.Decimal, motivo string) (gravou bool, err error)
}
