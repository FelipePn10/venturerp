package entity

import "github.com/shopspring/decimal"

// LinhaDeFator é o que o recálculo do fator de estoque precisa de uma linha
// de estrutura: o que a engenharia escreveu, a unidade de estoque ATUAL do
// componente e o que está gravado hoje (quantidade convertida e fator).
type LinhaDeFator struct {
	ID             int64
	ParentCode     int64
	ParentLabel    string
	ChildCode      int64
	ChildLabel     string
	Quantidade     decimal.Decimal
	Unidade        string
	UnidadeEstoque string
	QtdGravada     decimal.Decimal
	FatorGravado   decimal.Decimal
}

// FiltroDeFatores escolhe as linhas: dos componentes (Filhos), de um pai
// (Pai) ou, os dois vazios, todas as linhas ativas da empresa.
type FiltroDeFatores struct {
	Filhos []int64
	Pai    *int64
}

// Escala das colunas (NUMERIC(20,10)): a comparação com o gravado é feita
// nela, senão todo recálculo "mudaria" a 11ª casa.
const EscalaDoFator = 10

// Defasada diz se o gravado difere do recalculado.
func (l LinhaDeFator) Defasada(qtd, fator decimal.Decimal) bool {
	return !l.QtdGravada.Round(EscalaDoFator).Equal(qtd.Round(EscalaDoFator)) ||
		!l.FatorGravado.Round(EscalaDoFator).Equal(fator.Round(EscalaDoFator))
}
