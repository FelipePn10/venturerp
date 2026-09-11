package entity

import "strings"

// SetupTransicao é uma linha da matriz de tempo de preparação: quanto custa
// trocar de um item (ou família) para outro num centro de trabalho.
type SetupTransicao struct {
	ID           int64
	WorkCenterID int64
	FromItemCode *int64
	ToItemCode   *int64
	FromFamily   *string
	ToFamily     *string
	SetupMinutes float64
	IsActive     bool
}

// ContextoDeSetup descreve a transição concreta que o sequenciador precisa
// avaliar: o que estava na máquina e o que vai entrar.
type ContextoDeSetup struct {
	DeItem    *int64
	ParaItem  int64
	DeFamilia string
	ParaFam   string
}

// especificidade pontua o quanto a regra é específica. Item vale mais que
// família, e um lado preenchido vale mais que "qualquer": a regra mais
// específica ganha, como em qualquer tabela de exceções.
func especificidade(t SetupTransicao) int {
	pontos := 0
	if t.FromItemCode != nil {
		pontos += 8
	}
	if t.ToItemCode != nil {
		pontos += 8
	}
	if t.FromFamily != nil && *t.FromFamily != "" {
		pontos += 2
	}
	if t.ToFamily != nil && *t.ToFamily != "" {
		pontos += 2
	}
	return pontos
}

func casaItem(regra *int64, valor *int64) bool {
	if regra == nil {
		return true // "qualquer"
	}
	return valor != nil && *regra == *valor
}

func casaFamilia(regra *string, valor string) bool {
	if regra == nil || *regra == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(*regra), strings.TrimSpace(valor))
}

// SetupDaTransicao devolve o tempo de preparação da transição e se alguma regra
// foi encontrada. Entre as regras que casam, vence a mais específica.
//
// Sem regra o chamador mantém o setup fixo da operação — a matriz é um refinamento,
// nunca um requisito para sequenciar.
func SetupDaTransicao(regras []SetupTransicao, ctx ContextoDeSetup) (float64, bool) {
	melhorPontos := -1
	melhorMinutos := 0.0
	achou := false

	for _, r := range regras {
		if !r.IsActive {
			continue
		}
		if r.WorkCenterID != 0 && ctx.ParaItem == 0 {
			continue
		}
		para := ctx.ParaItem
		if !casaItem(r.FromItemCode, ctx.DeItem) || !casaItem(r.ToItemCode, &para) {
			continue
		}
		if !casaFamilia(r.FromFamily, ctx.DeFamilia) || !casaFamilia(r.ToFamily, ctx.ParaFam) {
			continue
		}
		if p := especificidade(r); p > melhorPontos {
			melhorPontos = p
			melhorMinutos = r.SetupMinutes
			achou = true
		}
	}
	return melhorMinutos, achou
}
